package api

import (
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
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
		status, _ := nativeCodeTaskState(state)
		if status != "" && (session.LastTaskID != tracker.lastTaskID || status != tracker.taskStatus) {
			if syncNativeCodeTaskStatus(session, state) {
				tracker.lastTaskID = session.LastTaskID
				tracker.taskStatus = status
			}
		}
		notifyState := tracker.observe(state)
		if notifyState == "" {
			return
		}
		if notifyState == service.CodeNotifyCompleted || notifyState == service.CodeNotifyFailed {
			persistNativeCodeHistory(session.ID)
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
			persistNativeCodeHistory(sessionID)
			return
		}
	}
}

func persistNativeCodeHistory(sessionID uint) {
	session, err := repo.NewAIDevSessionRepo().GetSessionByID(sessionID)
	if err != nil || !supportsNativeCodeHistory(session.AgentName) {
		return
	}
	messages, err := getNativeCodeMessages(session)
	if err != nil {
		global.LOG.Warnf("Read native %s history for session %d failed: %v", session.AgentName, session.ID, err)
		return
	}
	if err := persistNativeCodexMessages(session.ID, messages); err != nil {
		global.LOG.Warnf("Persist native %s history for session %d failed: %v", session.AgentName, session.ID, err)
	}
}

func nativeCodeTaskState(state *codexRuntimeState) (string, string) {
	if state == nil {
		return "", ""
	}
	switch state.ResponseState {
	case "responding":
		return "running", "executing"
	case "needsInput":
		return "pending_approval", "awaiting_approval"
	case "completed":
		return "completed", "completed"
	case "failed":
		return "failed", "failed"
	default:
		return "", ""
	}
}

func syncNativeCodeTaskStatus(session *model.AIDevSession, state *codexRuntimeState) bool {
	if session == nil || session.LastTaskID == 0 {
		return false
	}
	if state == nil || state.UpdatedAt.IsZero() {
		return false
	}
	status, sessionStage := nativeCodeTaskState(state)
	if status == "" || sessionStage == "" {
		return false
	}
	var current *model.AIDevSession
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		lockedSession, err := lockCodeSessionForDevelopment(tx, session.ID)
		if err != nil {
			return err
		}
		var task model.AITask
		if err := tx.Where("id = ? AND session_id = ?", lockedSession.LastTaskID, lockedSession.ID).First(&task).Error; err != nil {
			return err
		}
		if !nativeCodeRuntimeIsFresh(lockedSession, &task, state) {
			return gorm.ErrInvalidData
		}
		if err := reconcileCodeTaskState(tx, lockedSession, &task, status, sessionStage); err != nil {
			return err
		}
		current = lockedSession
		return nil
	})
	if err != nil {
		return false
	}
	session.Status = current.Status
	session.CurrentStage = current.CurrentStage
	session.LastTaskID = current.LastTaskID
	return true
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
