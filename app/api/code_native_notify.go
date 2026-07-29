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
			syncNativeCodeTaskStatus(session, status)
			tracker.lastTaskID = session.LastTaskID
			tracker.taskStatus = status
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

func syncNativeCodeTaskStatus(session *model.AIDevSession, status string) {
	if session == nil || session.LastTaskID == 0 {
		return
	}
	if status == "" {
		return
	}
	taskRepo := repo.NewAITaskRepo()
	task, err := taskRepo.GetTaskByID(session.LastTaskID)
	if err != nil || task.Status == status {
		return
	}
	task.Status = status
	_ = taskRepo.UpdateTask(task)
}

func codeRuntimeNotifySummary(state *codexRuntimeState) string {
	if state == nil {
		return ""
	}
	return state.LastAssistantPreview
}
