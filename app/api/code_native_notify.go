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

// 每多少个轮询周期兜底固化一次原生历史。
// 轮询是 1 秒一跳，固化要读并解析 rollout 文件，不能每跳都做；
// 30 秒一次足够把进程被 kill 的损失限制在可接受范围，开销也可以忽略。
const nativeCodeHistoryPersistTicks = 30

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
		state := getCodeRuntimeState(session, false)
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
	ticks := 0
	for {
		select {
		case <-ticker.C:
			check()
			// 定期落库。原先只在「跑完/失败」和「优雅退出」两个时机固化，
			// 面板被直接 kill 时 done 分支根本不会执行，整段历史就只剩在内存的 PTY 缓冲里。
			// 这里按周期兜底，把进程异常退出的损失从「全部」压到「最后一个周期」。
			ticks++
			if ticks%nativeCodeHistoryPersistTicks == 0 {
				persistNativeCodeHistory(sessionID)
			}
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
