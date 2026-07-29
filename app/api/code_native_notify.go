package api

import (
	"time"

	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
)

const nativeCodeNotifyPollInterval = time.Second

type nativeCodeNotifyTracker struct {
	initialized bool
	activeTurn  bool
	lastState   string
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

func codeRuntimeNotifySummary(state *codexRuntimeState) string {
	if state == nil {
		return ""
	}
	return state.LastAssistantPreview
}
