package api

import (
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

const (
	codeSessionStatusActive     = "active"
	codeSessionStatusDelivering = "delivering"
	codeSessionStatusDelivered  = "delivered"
)

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
	if err := tx.Model(&model.AIDevSession{}).Where("id = ?", session.ID).Updates(updates).Error; err != nil {
		return err
	}
	session.Status, session.CurrentStage = codeSessionStatusDelivering, "delivery_queued"
	if session.LastTaskID == 0 {
		return nil
	}
	return tx.Model(&model.AITask{}).Where("id = ?", session.LastTaskID).Update("status", codeSessionStatusDelivering).Error
}

func completeCodeSessionLifecycle(tx *gorm.DB, sessionID uint, deliveredAt time.Time) error {
	updated := tx.Model(&model.AIDevSession{}).Where("id = ? AND status <> ?", sessionID, codeSessionStatusDelivered).Updates(map[string]any{
		"status": codeSessionStatusDelivered, "current_stage": codeDeliveryStageCompleted, "delivered_at": deliveredAt,
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
		Where("ai_code_delivery_jobs.status = ? AND ai_dev_sessions.status <> ?", codeDeliveryJobCompleted, codeSessionStatusDelivered).
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
