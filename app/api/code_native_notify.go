package api

import (
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
)

const nativeCodeNotifyPollInterval = time.Second

type nativeCodeNotifyTracker struct {
	initialized bool
	activeTurn  bool
	lastState   string
	taskStatus  string
	lastTaskID  uint
}

func (tracker *nativeCodeNotifyTracker) observe(state *codexRuntimeState) string {
	if state == nil {
		return ""
	}
	current := state.ResponseState
	if !tracker.initialized {
		tracker.initialized = true
		tracker.lastState = current
		tracker.activeTurn = current == "responding"
		return ""
	}
	if current == tracker.lastState {
		return ""
	}
	tracker.lastState = current
	if current == "responding" {
		tracker.activeTurn = true
		return ""
	}
	if !tracker.activeTurn {
		return ""
	}
	switch current {
	case "needsInput":
		tracker.activeTurn = false
		return service.CodeNotifyApproval
	case "completed":
		tracker.activeTurn = false
		return service.CodeNotifyCompleted
	case "failed":
		tracker.activeTurn = false
		return service.CodeNotifyFailed
	default:
		return ""
	}
}

func watchNativeCodeNotifications(sessionID uint, done <-chan struct{}) {
	tracker := &nativeCodeNotifyTracker{}
	check := func() {
		session, err := repo.NewAIDevSessionRepo().GetSessionByID(sessionID)
		if err != nil {
			return
		}
		state := getCodexRuntimeState(session)
		status := nativeCodeTaskStatus(state)
		if status != "" && (session.LastTaskID != tracker.lastTaskID || status != tracker.taskStatus) {
			if syncNativeCodeTaskStatus(session, state, status) {
				tracker.lastTaskID = session.LastTaskID
				tracker.taskStatus = status
			}
		}
		notifyState := tracker.observe(state)
		if notifyState == "" {
			return
		}
		go service.NotifyCodeSession(session, nil, notifyState, codeRuntimeNotifySummary(state))
	}
	check()
	ticker := time.NewTicker(nativeCodeNotifyPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			check()
		case <-done:
			check()
			return
		}
	}
}

func nativeCodeTaskStatus(state *codexRuntimeState) string {
	if state == nil {
		return ""
	}
	switch state.ResponseState {
	case "responding":
		return "running"
	case "needsInput":
		return "pending_approval"
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	default:
		return ""
	}
}

func syncNativeCodeTaskStatus(session *model.AIDevSession, state *codexRuntimeState, status string) bool {
	if session == nil || session.LastTaskID == 0 {
		return false
	}
	if state == nil || state.UpdatedAt.IsZero() || status == "" {
		return false
	}
	taskRepo := repo.NewAITaskRepo()
	task, err := taskRepo.GetTaskByID(session.LastTaskID)
	if err != nil {
		return false
	}
	if !nativeCodeRuntimeIsFresh(session, task, state) {
		return false
	}
	if task.Status == status {
		return true
	}
	task.Status = status
	return taskRepo.UpdateTask(task) == nil
}

func nativeCodeRuntimeIsFresh(session *model.AIDevSession, task *model.AITask, state *codexRuntimeState) bool {
	if session == nil || task == nil || state == nil || state.UpdatedAt.IsZero() {
		return false
	}
	freshAfter := task.CreatedAt
	if session.LastInstructionAt != nil && session.LastInstructionAt.After(freshAfter) {
		freshAfter = *session.LastInstructionAt
	}
	return !state.UpdatedAt.Before(freshAfter.Add(-time.Second))
}

func codeRuntimeNotifySummary(state *codexRuntimeState) string {
	if state == nil {
		return ""
	}
	return state.LastAssistantPreview
}
