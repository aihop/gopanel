package api

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type codeInstructionResult struct {
	Output   string
	Previews []*model.AIPreview
	Err      error
}

func executeCodeInstruction(
	ctx context.Context,
	sessionRepo repo.IAIDevSessionRepo,
	taskRepo repo.IAITaskRepo,
	session *model.AIDevSession,
	task *model.AITask,
	instruction *model.AIInstruction,
	claim bool,
) codeInstructionResult {
	if instruction == nil || session == nil || task == nil {
		return codeInstructionResult{Err: errors.New("开发指令缺少会话或任务")}
	}
	executionContext, cancelExecution := context.WithCancel(ctx)
	defer cancelExecution()
	lease, err := codeExecutions.acquireSession(executionContext, session, codeExecutionInstruction, true)
	if err != nil {
		return codeInstructionResult{Err: err}
	}
	defer lease.Release()
	lease.SetCancel(cancelExecution)
	if err := executionContext.Err(); err != nil {
		return codeInstructionResult{Err: err}
	}
	if err := validateCodeTokenBudget(session); err != nil {
		finishErr := finishCodeInstruction(session, task, instruction, err.Error(), nil, err, false)
		return codeInstructionResult{Output: err.Error(), Err: errors.Join(err, finishErr)}
	}
	if err := startCodeInstructionExecution(session, task, instruction, claim); err != nil {
		return codeInstructionResult{Err: err}
	}

	_, output, execErr := executeCodeAgentRun(
		executionContext, sessionRepo, taskRepo, session, task, instruction,
		session.AgentName, session.WorkDir, instruction.Content,
	)
	previews, previewErr := upsertAIPreviews(sessionRepo, session, task, instruction, output)
	if previewErr != nil {
		global.LOG.Errorf("Failed to upsert AI previews: %v", previewErr)
	}
	resultErr := errors.Join(execErr, previewErr)
	finishErr := finishCodeInstruction(session, task, instruction, output, previews, resultErr, executionContext.Err() != nil)
	return codeInstructionResult{Output: output, Previews: previews, Err: errors.Join(resultErr, finishErr)}
}

func startCodeInstructionExecution(session *model.AIDevSession, task *model.AITask, instruction *model.AIInstruction, claim bool) error {
	now := time.Now()
	return global.DB.Transaction(func(tx *gorm.DB) error {
		lockedSession, err := lockCodeSessionForDevelopment(tx, session.ID)
		if err != nil {
			return err
		}
		session = lockedSession
		query := tx.Model(&model.AIInstruction{}).Where("id = ?", instruction.ID)
		if claim {
			query = query.Where("status = ?", "queued")
		}
		result := query.Update("status", "running")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("开发指令已被处理")
		}
		instruction.Status = "running"
		task.Status = "running"
		session.Status = "active"
		session.CurrentStage = "executing"
		session.LastTaskID = task.ID
		session.LastInstructionAt = &now
		if err := tx.Save(task).Error; err != nil {
			return err
		}
		if err := tx.Save(session).Error; err != nil {
			return err
		}
		return tx.Create(&model.AITimelineEvent{
			SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID,
			EventType: "execution_started", Stage: "executing", Title: "开始执行开发任务",
			Content: buildTimelineContent(instruction.Content), Status: "running",
		}).Error
	})
}

func finishCodeInstruction(
	session *model.AIDevSession,
	task *model.AITask,
	instruction *model.AIInstruction,
	output string,
	previews []*model.AIPreview,
	execErr error,
	cancelled bool,
) error {
	stage, title, status := "completed", "开发任务已完成", "success"
	instruction.Status = "completed"
	if cancelled {
		stage, title, status = "cancelled", "开发任务已停止", "warning"
		instruction.Status = "cancelled"
	} else if execErr != nil {
		stage, title, status = "failed", "开发任务执行失败", "error"
		instruction.Status = "failed"
	} else if len(previews) > 0 {
		stage, title = "preview_ready", "开发预览已生成"
	}
	if strings.TrimSpace(output) == "" && execErr != nil {
		output = execErr.Error()
	}
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(instruction).Error; err != nil {
			return err
		}
		if err := reconcileCodeTaskState(tx, session, task, instruction.Status, stage); err != nil {
			return err
		}
		if err := tx.Create(&model.AITimelineEvent{
			SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID,
			EventType: "execution_result", Stage: stage, Title: title,
			Content: summarizeAIRecentOutput(output), Status: status,
		}).Error; err != nil {
			return err
		}
		for _, preview := range previews {
			if preview == nil {
				continue
			}
			if err := tx.Create(&model.AITimelineEvent{
				SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID,
				EventType: "preview_ready", Stage: "preview_ready", Title: "预览已生成",
				Content: buildTimelineContent(preview.URL), Status: "success",
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	notifyState := service.CodeNotifyCompleted
	if stage == "failed" || stage == "cancelled" {
		notifyState = service.CodeNotifyFailed
	}
	go service.NotifyCodeSession(session, task, notifyState, summarizeAIRecentOutput(output))
	return nil
}
