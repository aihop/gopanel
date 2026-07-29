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
	if claim {
		claimed, err := sessionRepo.ClaimInstruction(instruction.ID)
		if err != nil {
			return codeInstructionResult{Err: err}
		}
		if !claimed {
			return codeInstructionResult{Err: errors.New("开发指令已被处理")}
		}
	}

	instruction.Status = "running"
	task.Status = "running"
	now := time.Now()
	session.Status = "active"
	session.CurrentStage = "executing"
	session.LastTaskID = task.ID
	session.LastInstructionAt = &now
	_ = taskRepo.UpdateTask(task)
	_ = sessionRepo.UpdateSession(session)
	createAITimelineEvent(sessionRepo, &model.AITimelineEvent{
		SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID,
		EventType: "execution_started", Stage: "executing", Title: "开始执行开发任务",
		Content: buildTimelineContent(instruction.Content), Status: "running",
	})

	_, output, execErr := executeCodeAgentRun(
		ctx, sessionRepo, taskRepo, session, task, instruction,
		session.AgentName, session.WorkDir, instruction.Content,
	)
	previews, previewErr := upsertAIPreviews(sessionRepo, session, task, instruction, output)
	if previewErr != nil {
		global.LOG.Errorf("Failed to upsert AI previews: %v", previewErr)
	}
	resultErr := errors.Join(execErr, previewErr)
	finishCodeInstruction(sessionRepo, taskRepo, session, task, instruction, output, previews, resultErr, ctx.Err() != nil)
	return codeInstructionResult{Output: output, Previews: previews, Err: resultErr}
}

func finishCodeInstruction(
	sessionRepo repo.IAIDevSessionRepo,
	taskRepo repo.IAITaskRepo,
	session *model.AIDevSession,
	task *model.AITask,
	instruction *model.AIInstruction,
	output string,
	previews []*model.AIPreview,
	execErr error,
	cancelled bool,
) {
	stage, title, status := "completed", "开发任务已完成", "success"
	task.Status = "completed"
	instruction.Status = "completed"
	if cancelled {
		stage, title, status = "cancelled", "开发任务已停止", "warning"
		task.Status = "cancelled"
		instruction.Status = "cancelled"
	} else if execErr != nil {
		stage, title, status = "failed", "开发任务执行失败", "error"
		task.Status = "failed"
		instruction.Status = "failed"
	} else if len(previews) > 0 {
		stage, title = "preview_ready", "开发预览已生成"
	}
	session.CurrentStage = stage
	createAITimelineEvent(sessionRepo, &model.AITimelineEvent{
		SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID,
		EventType: "execution_result", Stage: stage, Title: title,
		Content: summarizeAIRecentOutput(output), Status: status,
	})
	for _, preview := range previews {
		if preview == nil {
			continue
		}
		createAITimelineEvent(sessionRepo, &model.AITimelineEvent{
			SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID,
			EventType: "preview_ready", Stage: "preview_ready", Title: "预览已生成",
			Content: buildTimelineContent(preview.URL), Status: "success",
		})
	}
	if strings.TrimSpace(output) == "" && execErr != nil {
		output = execErr.Error()
	}
	_ = taskRepo.UpdateTask(task)
	_ = sessionRepo.UpdateInstruction(instruction)
	_ = sessionRepo.UpdateSession(session)
	notifyState := service.CodeNotifyCompleted
	if stage == "failed" || stage == "cancelled" {
		notifyState = service.CodeNotifyFailed
	}
	go service.NotifyCodeSession(session, task, notifyState, summarizeAIRecentOutput(output))
}
