package api

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	codeSessionStatusActive     = "active"
	codeSessionStatusDelivering = "delivering"
	codeSessionStatusDelivered  = "delivered"
)

type codeSessionLifecycleLocker struct {
	mu    sync.Mutex
	locks map[uint]*sync.Mutex
}

var codeSessionLifecycles = &codeSessionLifecycleLocker{locks: make(map[uint]*sync.Mutex)}

var errCodeSessionWorkspaceBusy = errors.New("当前会话正在执行 AI 指令或终端操作，请完成或停止后再修改工作区")

func codeSessionWorkspaceMutationError(err error) error {
	if errors.Is(err, errCodeExecutionBusy) {
		return errCodeSessionWorkspaceBusy
	}
	return err
}

func (locker *codeSessionLifecycleLocker) lock(sessionID uint) func() {
	locker.mu.Lock()
	lock := locker.locks[sessionID]
	if lock == nil {
		lock = &sync.Mutex{}
		locker.locks[sessionID] = lock
	}
	locker.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}

func validateCodeSessionDevelopmentOpen(session *model.AIDevSession) error {
	if session == nil {
		return errors.New("开发会话不可用")
	}
	switch strings.TrimSpace(session.Status) {
	case codeSessionStatusDelivering:
		return errors.New("当前会话正在统一交付，不能继续执行开发指令")
	case codeSessionStatusDelivered:
		return errors.New("当前会话已完成统一交付，请新建会话继续开发")
	default:
		return nil
	}
}

func lockCodeSessionForDevelopment(tx *gorm.DB, sessionID uint) (*model.AIDevSession, error) {
	var session model.AIDevSession
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&session, sessionID).Error; err != nil {
		return nil, err
	}
	if err := validateCodeSessionDevelopmentOpen(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

func updateCodeSessionDevelopmentState(tx *gorm.DB, sessionID uint, updates map[string]any) error {
	result := tx.Model(&model.AIDevSession{}).
		Where("id = ? AND status NOT IN ?", sessionID, []string{codeSessionStatusDelivering, codeSessionStatusDelivered}).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var session model.AIDevSession
	if err := tx.First(&session, sessionID).Error; err != nil {
		return err
	}
	if err := validateCodeSessionDevelopmentOpen(&session); err != nil {
		return err
	}
	return errors.New("开发会话状态已变化，请刷新后重试")
}

func runCodeSessionWorkspaceMutation(session *model.AIDevSession, operation func(*model.AIDevSession) error) error {
	return runCodeSessionWorkspaceMutationWithTx(session, func(_ *gorm.DB, current *model.AIDevSession) error {
		return operation(current)
	})
}

func runCodeSessionWorkspaceMutationWithTx(session *model.AIDevSession, operation func(*gorm.DB, *model.AIDevSession) error) error {
	if session == nil || session.ID == 0 {
		return errors.New("开发会话不可用")
	}
	unlockLifecycle := codeSessionLifecycles.lock(session.ID)
	defer unlockLifecycle()
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionMutation, false)
	if err != nil {
		return codeSessionWorkspaceMutationError(err)
	}
	defer lease.Release()
	return global.DB.Transaction(func(tx *gorm.DB) error {
		current, err := lockCodeSessionForDevelopment(tx, session.ID)
		if err != nil {
			return err
		}
		return operation(tx, current)
	})
}

func validateCodeSessionReadyForDelivery(tx *gorm.DB, session *model.AIDevSession) error {
	if err := validateCodeSessionDevelopmentOpen(session); err != nil {
		return err
	}
	var activeInstructions int64
	if err := tx.Model(&model.AIInstruction{}).Where(
		"session_id = ? AND status IN ?", session.ID,
		[]string{"queued", "running", "pending_approval"},
	).Count(&activeInstructions).Error; err != nil {
		return err
	}
	if activeInstructions > 0 {
		return errors.New("当前会话仍有待处理指令，请完成或停止后再统一交付")
	}
	return nil
}

func markCodeSessionDelivering(tx *gorm.DB, session *model.AIDevSession) error {
	updates := map[string]any{"status": codeSessionStatusDelivering, "current_stage": "delivery_queued"}
	result := tx.Model(&model.AIDevSession{}).
		Where("id = ? AND status NOT IN ?", session.ID, []string{codeSessionStatusDelivering, codeSessionStatusDelivered}).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("开发会话状态已变化，请刷新后重试")
	}
	session.Status, session.CurrentStage = codeSessionStatusDelivering, "delivery_queued"
	if session.LastTaskID == 0 {
		return nil
	}
	return tx.Model(&model.AITask{}).Where("id = ?", session.LastTaskID).Update("status", codeSessionStatusDelivering).Error
}

func completeCodeSessionLifecycle(tx *gorm.DB, sessionID uint, deliveredAt time.Time) error {
	updated := tx.Model(&model.AIDevSession{}).Where("id = ?", sessionID).Updates(map[string]any{
		"status": codeSessionStatusActive, "current_stage": codeDeliveryStageCompleted, "delivered_at": deliveredAt,
	})
	if updated.Error != nil || updated.RowsAffected == 0 {
		return updated.Error
	}
	return tx.Model(&model.AITask{}).Where("session_id = ?", sessionID).Update("status", "completed").Error
}

func reopenCodeSessionLifecycle(tx *gorm.DB, sessionID uint) error {
	if err := tx.Model(&model.AIDevSession{}).Where("id = ? AND status = ?", sessionID, codeSessionStatusDelivering).
		Updates(map[string]any{"status": codeSessionStatusActive, "current_stage": "delivery_failed"}).Error; err != nil {
		return err
	}
	return tx.Model(&model.AITask{}).Where("session_id = ? AND status = ?", sessionID, codeSessionStatusDelivering).
		Update("status", "completed").Error
}

func restoreCodeDeliverySessionLifecycles() {
	var sessionIDs []uint
	err := global.DB.Model(&model.AICodeDeliveryJob{}).
		Where("status IN ?", []string{codeDeliveryJobQueued, codeDeliveryJobRunning}).
		Distinct().Pluck("session_id", &sessionIDs).Error
	if err != nil {
		return
	}
	if len(sessionIDs) > 0 {
		_ = global.DB.Model(&model.AIDevSession{}).Where("id IN ? AND status <> ?", sessionIDs, codeSessionStatusDelivered).
			Updates(map[string]any{"status": codeSessionStatusDelivering, "current_stage": "delivery_queued"}).Error
		_ = global.DB.Model(&model.AITask{}).Where("session_id IN ?", sessionIDs).
			Update("status", codeSessionStatusDelivering).Error
	}
	var completedJobs []model.AICodeDeliveryJob
	if err := global.DB.Model(&model.AICodeDeliveryJob{}).
		Joins("JOIN ai_dev_sessions ON ai_dev_sessions.id = ai_code_delivery_jobs.session_id").
		Where("ai_code_delivery_jobs.status = ? AND ai_dev_sessions.current_stage <> ?", codeDeliveryJobCompleted, codeDeliveryStageCompleted).
		Find(&completedJobs).Error; err != nil {
		return
	}
	for _, job := range completedJobs {
		deliveredAt := job.UpdatedAt
		if job.CompletedAt != nil {
			deliveredAt = *job.CompletedAt
		}
		_ = global.DB.Transaction(func(tx *gorm.DB) error {
			return completeCodeSessionLifecycle(tx, job.SessionID, deliveredAt)
		})
	}
}

func (runner *codeDeliveryRunner) cancelSession(sessionID uint) {
	if sessionID == 0 {
		return
	}
	runner.mu.Lock()
	if runner.cancelled == nil {
		runner.cancelled = make(map[uint]struct{})
	}
	runner.cancelled[sessionID] = struct{}{}
	runner.mu.Unlock()
}

func (runner *codeDeliveryRunner) isSessionCancelled(sessionID uint) bool {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	_, cancelled := runner.cancelled[sessionID]
	return cancelled
}
