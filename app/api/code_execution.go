package api

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

func executeCodeAgentRun(
	ctx context.Context,
	sessionRepo repo.IAIDevSessionRepo,
	taskRepo repo.IAITaskRepo,
	session *model.AIDevSession,
	task *model.AITask,
	instruction *model.AIInstruction,
	executorID string,
	workDir string,
	prompt string,
) (*model.AIExecutionRun, string, error) {
	startedAt := time.Now()
	sessionID := uint(0)
	taskID := uint(0)
	instructionID := uint(0)
	nativeSessionID := ""
	if session != nil {
		sessionID = session.ID
		nativeSessionID = session.NativeSessionID
	}
	if task != nil {
		taskID = task.ID
		if sessionID == 0 {
			sessionID = task.SessionID
		}
		if nativeSessionID == "" {
			nativeSessionID = task.NativeSessionID
		}
	}
	if instruction != nil {
		instructionID = instruction.ID
	}
	run := &model.AIExecutionRun{
		SessionID:       sessionID,
		TaskID:          taskID,
		InstructionID:   instructionID,
		ExecutorID:      executorID,
		NativeSessionID: nativeSessionID,
		Prompt:          prompt,
		Status:          "running",
		StartedAt:       startedAt,
	}
	if err := sessionRepo.CreateExecutionRun(run); err != nil {
		return nil, "", err
	}
	if instruction == nil && task != nil {
		if err := taskRepo.CreateMessage(&model.AIMessage{SessionID: sessionID, TaskID: task.ID, RunID: run.ID, Role: "user", Content: prompt}); err != nil {
			return failCodeExecutionRun(sessionRepo, run, startedAt, err)
		}
	}

	executorSessionKey := sessionID
	if executorSessionKey == 0 {
		executorSessionKey = taskID
	}
	if err := ensureCodeManagedPushGuards(session); err != nil {
		return failCodeExecutionRun(sessionRepo, run, startedAt, err)
	}
	executionPrompt := codeMemoryPrompt(session, codeManagedDeliveryPrompt(session, prompt))
	command, preparedSessionID, buildErr := buildCodeExecutorCommand(ctx, executorID, workDir, executionPrompt, nativeSessionID, executorSessionKey, session)
	if preparedSessionID != "" {
		run.NativeSessionID = preparedSessionID
	}
	rawOutput := []byte{}
	execErr := buildErr
	if execErr == nil {
		output := &boundedCodeOutput{}
		command.Stdout = output
		command.Stderr = output
		execErr = command.Run()
		rawOutput = output.Bytes()
	}
	parsed := parseCodeExecutorOutput(executorID, rawOutput, run.NativeSessionID)
	completedAt := time.Now()
	run.CompletedAt = &completedAt
	run.DurationMS = completedAt.Sub(startedAt).Milliseconds()
	run.RawOutput = parsed.RawOutput
	run.Output = parsed.Message
	run.NativeSessionID = parsed.NativeSessionID
	run.Model = parsed.Model
	run.InputTokens = parsed.InputTokens
	run.OutputTokens = parsed.OutputTokens
	run.CachedInputTokens = parsed.CachedInputTokens
	run.ReasoningTokens = parsed.ReasoningTokens
	run.TotalTokens = parsed.TotalTokens
	run.TokenUsageStatus = codeTokenUsageUnavailable
	if parsed.TokenUsageReported {
		run.TokenUsageStatus = codeTokenUsageRecorded
	}
	if execErr != nil {
		run.Status = "failed"
		if ctx.Err() != nil {
			run.Status = "cancelled"
		}
		run.ExitCode = executionExitCode(execErr)
		run.ErrorMessage = execErr.Error()
		if strings.TrimSpace(run.Output) == "" {
			run.Output = fmt.Sprintf("执行错误: %v", execErr)
		}
	} else {
		run.Status = "completed"
	}
	persistErr := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(run).Error; err != nil {
			return err
		}
		if task != nil {
			task.NativeSessionID = run.NativeSessionID
			if err := tx.Save(task).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.AIMessage{SessionID: sessionID, TaskID: task.ID, RunID: run.ID, Role: "agent", Content: run.Output}).Error; err != nil {
				return err
			}
		}
		if session != nil && run.NativeSessionID != "" && session.NativeSessionID != run.NativeSessionID {
			session.NativeSessionID = run.NativeSessionID
			if err := tx.Save(session).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if persistErr != nil {
		return run, run.Output, errors.Join(execErr, persistErr)
	}
	// 一次执行结束就顺手沉淀一次。放在这里而不是交付时：多数会话最终并不
	// 走到交付（实测 58 个会话只有 1 个到达终态），等交付再抽就几乎什么都留不下。
	if execErr == nil {
		enqueueCodeMemoryExtraction(sessionID, codeMemoryTriggerAutomatic, false)
	}
	return run, run.Output, execErr
}

func failCodeExecutionRun(sessionRepo repo.IAIDevSessionRepo, run *model.AIExecutionRun, startedAt time.Time, runErr error) (*model.AIExecutionRun, string, error) {
	completedAt := time.Now()
	run.CompletedAt = &completedAt
	run.DurationMS = completedAt.Sub(startedAt).Milliseconds()
	run.Status = "failed"
	run.ExitCode = -1
	run.ErrorMessage = runErr.Error()
	run.Output = fmt.Sprintf("执行错误: %v", runErr)
	run.TokenUsageStatus = codeTokenUsageUnavailable
	if updateErr := sessionRepo.UpdateExecutionRun(run); updateErr != nil {
		return run, run.Output, errors.Join(runErr, updateErr)
	}
	return run, run.Output, runErr
}

func executionExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
