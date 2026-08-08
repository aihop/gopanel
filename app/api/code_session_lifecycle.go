package api

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	codeSessionStatusInitializing = "initializing"
	codeSessionStatusActive       = "active"
	codeSessionStatusFailed       = "failed"
	codeSessionStatusDelivering   = "delivering"
	codeSessionStatusDelivered    = "delivered"
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
	case codeSessionStatusInitializing:
		return errors.New("当前会话正在同步远端并创建隔离工作区，请稍后重试")
	case codeSessionStatusFailed:
		return errors.New("当前会话初始化失败，请重试初始化或新建会话")
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

func runCodeSessionGitMutation(session *model.AIDevSession, operation func(*model.AIDevSession) error) error {
	if session == nil || session.ID == 0 {
		return errors.New("开发会话不可用")
	}
	unlockLifecycle := codeSessionLifecycles.lock(session.ID)
	defer unlockLifecycle()
	return global.DB.Transaction(func(tx *gorm.DB) error {
		current, err := lockCodeSessionForDevelopment(tx, session.ID)
		if err != nil {
			return err
		}
		if err := validateCodeSessionGitMutationIdle(tx, current.ID); err != nil {
			return err
		}
		return operation(current)
	})
}

func validateCodeSessionGitMutationIdle(tx *gorm.DB, sessionID uint) error {
	var activeInstructions int64
	if err := tx.Model(&model.AIInstruction{}).Where(
		"session_id = ? AND status IN ?", sessionID,
		[]string{"queued", "running", "pending_approval"},
	).Count(&activeInstructions).Error; err != nil {
		return err
	}
	if activeInstructions > 0 {
		return errors.New("当前会话仍有待处理指令，请完成或停止后再修改 Git 状态")
	}
	return nil
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
		"status": codeSessionStatusDelivered, "current_stage": codeDeliveryStageCompleted, "delivered_at": deliveredAt,
	})
	if updated.Error != nil || updated.RowsAffected == 0 {
		return updated.Error
	}
	return tx.Model(&model.AITask{}).Where("session_id = ?", sessionID).Update("status", "completed").Error
}

func continueCodeSessionAfterDelivery(tx *gorm.DB, sessionID uint, deliveredAt time.Time) error {
	updated := tx.Model(&model.AIDevSession{}).Where("id = ?", sessionID).Updates(map[string]any{
		"status": codeSessionStatusActive, "current_stage": codeDeliveryStageCompleted, "delivered_at": deliveredAt,
	})
	if updated.Error != nil || updated.RowsAffected == 0 {
		return updated.Error
	}
	return tx.Model(&model.AITask{}).Where("session_id = ? AND status = ?", sessionID, codeSessionStatusDelivering).
		Update("status", "completed").Error
}

func finalizeCodeSessionLifecycle(db *gorm.DB, sessionID uint, deliveredAt time.Time) error {
	unlockLifecycle := codeSessionLifecycles.lock(sessionID)
	defer unlockLifecycle()
	continueDevelopment := shouldContinueCodeSessionAfterDelivery(db, sessionID)
	return db.Transaction(func(tx *gorm.DB) error {
		return applyCodeSessionLifecycleFinalization(tx, sessionID, deliveredAt, continueDevelopment)
	})
}

func shouldContinueCodeSessionAfterDelivery(db *gorm.DB, sessionID uint) bool {
	return codeExecutions.hasSessionKind(sessionID, codeExecutionInteractive) || codeSessionHasPostSnapshotChanges(db, sessionID)
}

func applyCodeSessionLifecycleFinalization(tx *gorm.DB, sessionID uint, deliveredAt time.Time, continueDevelopment bool) error {
	if continueDevelopment {
		return continueCodeSessionAfterDelivery(tx, sessionID, deliveredAt)
	}
	return completeCodeSessionLifecycle(tx, sessionID, deliveredAt)
}

type codeSessionPostSnapshotStatus struct {
	HasChanges            bool
	HasCommits            bool
	HasUncommittedChanges bool
}

func codeSessionHasPostSnapshotChanges(tx *gorm.DB, sessionID uint) bool {
	return inspectCodeSessionPostSnapshotStatus(tx, sessionID).HasChanges
}

func inspectCodeSessionPostSnapshotStatus(tx *gorm.DB, sessionID uint) codeSessionPostSnapshotStatus {
	var repositories []model.AIDevSessionRepository
	if err := tx.Where("session_id = ?", sessionID).Find(&repositories).Error; err != nil {
		return codeSessionPostSnapshotStatus{HasChanges: true, HasUncommittedChanges: true}
	}
	if len(repositories) > 0 {
		result := codeSessionPostSnapshotStatus{}
		for index := range repositories {
			state := inspectCodeDeliveryWorktreeChanges(repositories[index].WorktreeDir, repositories[index].WorktreeCommit)
			result.HasCommits = result.HasCommits || state.HasCommits
			result.HasUncommittedChanges = result.HasUncommittedChanges || state.HasUncommittedChanges
		}
		result.HasChanges = result.HasCommits || result.HasUncommittedChanges
		return result
	}
	var delivery model.AICodeDelivery
	if err := tx.Where("session_id = ?", sessionID).First(&delivery).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return codeSessionPostSnapshotStatus{}
	} else if err != nil {
		return codeSessionPostSnapshotStatus{HasChanges: true, HasUncommittedChanges: true}
	}
	return inspectCodeDeliveryWorktreeChanges(delivery.WorkDir, delivery.WorktreeCommit)
}

func inspectCodeDeliveryWorktreeChanges(workDir, snapshotCommit string) codeSessionPostSnapshotStatus {
	workDir, snapshotCommit = strings.TrimSpace(workDir), strings.TrimSpace(snapshotCommit)
	if workDir == "" || snapshotCommit == "" {
		return codeSessionPostSnapshotStatus{}
	}
	if _, err := os.Stat(workDir); errors.Is(err, os.ErrNotExist) {
		return codeSessionPostSnapshotStatus{}
	} else if err != nil {
		return codeSessionPostSnapshotStatus{HasChanges: true, HasUncommittedChanges: true}
	}
	status, err := runCodeGit(workDir, "status", "--porcelain")
	if err != nil {
		return codeSessionPostSnapshotStatus{HasChanges: true, HasUncommittedChanges: true}
	}
	result := codeSessionPostSnapshotStatus{HasUncommittedChanges: strings.TrimSpace(status) != ""}
	commit, err := runCodeGit(workDir, "rev-parse", "HEAD")
	if err != nil {
		result.HasChanges, result.HasUncommittedChanges = true, true
		return result
	}
	result.HasCommits = strings.TrimSpace(commit) != snapshotCommit
	result.HasChanges = result.HasCommits || result.HasUncommittedChanges
	return result
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
	var activeJobs []model.AICodeDeliveryJob
	if err := global.DB.Model(&model.AICodeDeliveryJob{}).
		Where("status IN ?", []string{codeDeliveryJobQueued, codeDeliveryJobRunning}).
		Find(&activeJobs).Error; err != nil {
		global.LOG.Errorf("Restore active Code delivery lifecycles failed: %v", err)
		return
	}
	for index := range activeJobs {
		if err := restoreActiveCodeDeliveryLifecycle(&activeJobs[index]); err != nil {
			global.LOG.Errorf("Restore active Code delivery lifecycle %d failed: %v", activeJobs[index].ID, err)
		}
	}
	var completedJobs []model.AICodeDeliveryJob
	if err := global.DB.Model(&model.AICodeDeliveryJob{}).
		Joins("JOIN ai_dev_sessions ON ai_dev_sessions.id = ai_code_delivery_jobs.session_id").
		Where(
			"ai_code_delivery_jobs.status = ? AND (ai_dev_sessions.current_stage <> ? OR ai_dev_sessions.status = ?)",
			codeDeliveryJobCompleted, codeDeliveryStageCompleted, codeSessionStatusDelivering,
		).
		Find(&completedJobs).Error; err != nil {
		global.LOG.Errorf("Load completed Code delivery lifecycles failed: %v", err)
		return
	}
	for _, job := range completedJobs {
		deliveredAt := job.UpdatedAt
		if job.CompletedAt != nil {
			deliveredAt = *job.CompletedAt
		}
		if err := finalizeCodeSessionLifecycle(global.DB, job.SessionID, deliveredAt); err != nil {
			global.LOG.Errorf("Restore completed Code delivery lifecycle %d failed: %v", job.ID, err)
			continue
		}
		cleanupFinalizedCodeSessionWorktrees(job.SessionID)
	}
}

func restoreActiveCodeDeliveryLifecycle(job *model.AICodeDeliveryJob) error {
	unlockLifecycle := codeSessionLifecycles.lock(job.SessionID)
	defer unlockLifecycle()
	return global.DB.Transaction(func(tx *gorm.DB) error {
		updated := tx.Model(&model.AIDevSession{}).
			Where("id = ? AND status <> ?", job.SessionID, codeSessionStatusDelivered).
			Updates(map[string]any{"status": codeSessionStatusDelivering, "current_stage": "delivery_queued"})
		if updated.Error != nil || updated.RowsAffected == 0 || job.TaskID == 0 {
			return updated.Error
		}
		return tx.Model(&model.AITask{}).Where("id = ? AND session_id = ?", job.TaskID, job.SessionID).
			Update("status", codeSessionStatusDelivering).Error
	})
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
