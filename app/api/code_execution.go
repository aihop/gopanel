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
	command, preparedSessionID, buildErr := buildCodeExecutorCommand(ctx, executorID, workDir, prompt, nativeSessionID, executorSessionKey)
	if preparedSessionID != "" {
		run.NativeSessionID = preparedSessionID
	}
	rawOutput := []byte{}
	execErr := buildErr
	if execErr == nil {
		rawOutput, execErr = command.CombinedOutput()
	}
	parsed := parseCodeExecutorOutput(executorID, rawOutput, run.NativeSessionID)
	completedAt := time.Now()
	run.CompletedAt = &completedAt
	run.DurationMS = completedAt.Sub(startedAt).Milliseconds()
	run.RawOutput = parsed.RawOutput
	run.Output = parsed.Message
	run.NativeSessionID = parsed.NativeSessionID
	if execErr != nil {
		run.Status = "failed"
		run.ExitCode = executionExitCode(execErr)
		run.ErrorMessage = execErr.Error()
		if strings.TrimSpace(run.Output) == "" {
			run.Output = fmt.Sprintf("执行错误: %v", execErr)
		}
	} else {
		run.Status = "completed"
	}
	if err := sessionRepo.UpdateExecutionRun(run); err != nil {
		return run, run.Output, errors.Join(execErr, err)
	}
	if task != nil {
		task.NativeSessionID = run.NativeSessionID
		if err := taskRepo.UpdateTask(task); err != nil {
			return run, run.Output, errors.Join(execErr, err)
		}
		if err := taskRepo.CreateMessage(&model.AIMessage{SessionID: sessionID, TaskID: task.ID, RunID: run.ID, Role: "agent", Content: run.Output}); err != nil {
			return run, run.Output, errors.Join(execErr, err)
		}
	}
	if session != nil && run.NativeSessionID != "" && session.NativeSessionID != run.NativeSessionID {
		session.NativeSessionID = run.NativeSessionID
		if err := sessionRepo.UpdateSession(session); err != nil {
			return run, run.Output, errors.Join(execErr, err)
		}
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
