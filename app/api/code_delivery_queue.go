package api

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

const (
	codeDeliveryJobQueued    = "queued"
	codeDeliveryJobRunning   = "running"
	codeDeliveryJobCompleted = "completed"
	codeDeliveryJobPartial   = "partial"
	codeDeliveryJobConflict  = "conflict"
	codeDeliveryJobFailed    = "failed"

	codeDeliveryStageQueued           = "queued"
	codeDeliveryStageStoppingTerminal = "stopping_terminal"
	codeDeliveryStageSyncing          = "syncing"
	codeDeliveryStageMerging          = "merging"
	codeDeliveryStageQualityCheck     = "quality_check"
	codeDeliveryStagePushing          = "pushing"
	codeDeliveryStageVerifying        = "verifying"
	codeDeliveryStageCleaning         = "cleaning"
	codeDeliveryStageCompleted        = "completed"

	codeDeliveryLeaseDuration = 45 * time.Second
)

type codeDeliveryProgressReporter func(stage string, progress int)

type codeDeliveryJobView struct {
	ID                    uint                           `json:"id"`
	SessionID             uint                           `json:"sessionId"`
	TaskID                uint                           `json:"taskId,omitempty"`
	Status                string                         `json:"status"`
	Stage                 string                         `json:"stage"`
	Progress              int                            `json:"progress"`
	Attempt               int                            `json:"attempt"`
	QueuePosition         int                            `json:"queuePosition"`
	TargetBranch          string                         `json:"targetBranch,omitempty"`
	ResultCommit          string                         `json:"resultCommit,omitempty"`
	ResultType            string                         `json:"resultType,omitempty"`
	FailureCode           string                         `json:"failureCode,omitempty"`
	HasPendingChanges     bool                           `json:"hasPendingChanges"`
	HasPendingCommits     bool                           `json:"hasPendingCommits"`
	HasUncommittedChanges bool                           `json:"hasUncommittedChanges"`
	Repositories          []codeRepositoryDeliveryResult `json:"repositories,omitempty"`
	ErrorMessage          string                         `json:"errorMessage,omitempty"`
	ConflictFiles         []string                       `json:"conflictFiles"`
	CreatedAt             time.Time                      `json:"createdAt"`
	UpdatedAt             time.Time                      `json:"updatedAt"`
	StartedAt             *time.Time                     `json:"startedAt,omitempty"`
	CompletedAt           *time.Time                     `json:"completedAt,omitempty"`
}

type codeDeliveryRunner struct {
	mu        sync.Mutex
	queued    map[uint]struct{}
	cancelled map[uint]struct{}
	owner     string
}

var backgroundCodeDelivery = &codeDeliveryRunner{
	queued: make(map[uint]struct{}), cancelled: make(map[uint]struct{}), owner: newCodeRepositoryLeaseOwner("delivery"),
}
var codeDeliveryRecoveryOnce sync.Once

func codeDeliveryRepositoryKeys(session *model.AIDevSession) ([]string, string, error) {
	if session.IsolationMode == codeIsolationMultiWorktree || hasCodeMultiRepositoryDelivery(session.ID) {
		repositories, err := loadCodeSessionRepositories(session.ID)
		if err != nil || len(repositories) == 0 {
			return nil, "", errors.New("会话多仓库交付记录不可用")
		}
		keys := make([]string, 0, len(repositories))
		for _, repository := range repositories {
			targetBranch := repository.TargetBranch
			if strings.TrimSpace(targetBranch) == "" {
				targetBranch, _ = runCodeGit(repository.SourceDir, "branch", "--show-current")
			}
			keys = append(keys, codeDeliveryRepositoryKey(repository.SourceDir, repository.RemoteName, targetBranch))
		}
		sort.Strings(keys)
		return keys, "", nil
	}
	sourceDir, remoteName, targetBranch := session.SourceWorkDir, session.RemoteName, session.TargetBranch
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err == nil {
		sourceDir, remoteName, targetBranch = delivery.SourceWorkDir, delivery.RemoteName, delivery.TargetBranch
	}
	if strings.TrimSpace(sourceDir) == "" {
		return nil, "", errors.New("交付源仓库不可用")
	}
	if strings.TrimSpace(targetBranch) == "" {
		targetBranch, _ = runCodeGit(sourceDir, "branch", "--show-current")
	}
	if strings.TrimSpace(targetBranch) == "" {
		return nil, "", errors.New("交付目标分支不可用")
	}
	return []string{codeDeliveryRepositoryKey(sourceDir, remoteName, targetBranch)}, targetBranch, nil
}

func StartCodeDeliveryRecovery() {
	codeDeliveryRecoveryOnce.Do(func() {
		enqueuePersistedCodeDeliveries()
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					enqueuePersistedCodeDeliveries()
				case <-codeExecutions.stop:
					return
				}
			}
		}()
	})
}

func enqueuePersistedCodeDeliveries() {
	if global.DB == nil {
		return
	}
	restoreCodeDeliverySessionLifecycles()
	var ids []uint
	now := time.Now()
	err := global.DB.Model(&model.AICodeDeliveryJob{}).Where(
		"status = ? OR (status = ? AND (lease_expires_at IS NULL OR lease_expires_at < ?))",
		codeDeliveryJobQueued, codeDeliveryJobRunning, now,
	).Order("created_at ASC").Limit(500).Pluck("id", &ids).Error
	if err != nil {
		global.LOG.Errorf("Load queued Code deliveries failed: %v", err)
		return
	}
	for _, id := range ids {
		enqueueCodeDelivery(id)
	}
}

func enqueueCodeDelivery(jobID uint) {
	if codeExecutions.isStopping() {
		return
	}
	backgroundCodeDelivery.mu.Lock()
	if _, exists := backgroundCodeDelivery.queued[jobID]; exists {
		backgroundCodeDelivery.mu.Unlock()
		return
	}
	backgroundCodeDelivery.queued[jobID] = struct{}{}
	backgroundCodeDelivery.mu.Unlock()
	go backgroundCodeDelivery.run(jobID)
}

func decodeCodeDeliveryKeys(job *model.AICodeDeliveryJob) ([]string, error) {
	var keys []string
	if err := json.Unmarshal([]byte(job.RepositoryKeys), &keys); err != nil || len(keys) == 0 {
		return nil, errors.New("交付仓库租约信息无效")
	}
	sort.Strings(keys)
	return keys, nil
}

func (runner *codeDeliveryRunner) claim(jobID uint) (*model.AICodeDeliveryJob, []string, bool, error) {
	now, expiresAt := time.Now(), time.Now().Add(codeDeliveryLeaseDuration)
	result := global.DB.Model(&model.AICodeDeliveryJob{}).Where(
		"id = ? AND (status = ? OR (status = ? AND (lease_expires_at IS NULL OR lease_expires_at < ?)))",
		jobID, codeDeliveryJobQueued, codeDeliveryJobRunning, now,
	).Updates(map[string]any{
		"status": codeDeliveryJobRunning, "lease_owner": runner.owner, "lease_expires_at": expiresAt,
		"completed_at": nil,
	})
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, nil, false, result.Error
	}
	var job model.AICodeDeliveryJob
	if err := global.DB.First(&job, jobID).Error; err != nil {
		return nil, nil, false, err
	}
	keys, err := decodeCodeDeliveryKeys(&job)
	return &job, keys, true, err
}

func (runner *codeDeliveryRunner) acquireRepositoryLeases(job *model.AICodeDeliveryJob, keys []string) (bool, error) {
	return acquireCodeRepositoryLeases(runner.owner, job.ID, keys)
}

func (runner *codeDeliveryRunner) deferJob(jobID uint, err error) {
	updates := map[string]any{"status": codeDeliveryJobQueued, "stage": codeDeliveryStageQueued, "lease_owner": "", "lease_expires_at": nil}
	if err != nil {
		updates["error_message"] = err.Error()
	}
	_ = global.DB.Model(&model.AICodeDeliveryJob{}).Where("id = ? AND lease_owner = ?", jobID, runner.owner).Updates(updates).Error
	_ = global.DB.Where("job_id = ? AND lease_owner = ?", jobID, runner.owner).Delete(&model.AICodeDeliveryLease{}).Error
}

func (runner *codeDeliveryRunner) heartbeat(ctx context.Context, jobID uint, keys []string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			expiresAt := time.Now().Add(codeDeliveryLeaseDuration)
			_ = global.DB.Model(&model.AICodeDeliveryJob{}).Where("id = ? AND lease_owner = ?", jobID, runner.owner).Update("lease_expires_at", expiresAt).Error
			_ = global.DB.Model(&model.AICodeDeliveryLease{}).Where("job_id = ? AND lease_owner = ? AND repository_key IN ?", jobID, runner.owner, keys).Update("lease_expires_at", expiresAt).Error
		case <-ctx.Done():
			return
		}
	}
}

func (runner *codeDeliveryRunner) updateProgress(jobID uint, stage string, progress int) {
	_ = global.DB.Model(&model.AICodeDeliveryJob{}).Where("id = ? AND lease_owner = ?", jobID, runner.owner).
		Updates(map[string]any{"stage": stage, "progress": progress, "error_message": ""}).Error
}

func (runner *codeDeliveryRunner) finish(job *model.AICodeDeliveryJob, result codeGitDeliveryResult, runErr error) {
	status, stage, progress := codeDeliveryJobCompleted, codeDeliveryStageCompleted, 100
	failureCode := ""
	if result.Status == codeDeliveryJobPartial {
		status, stage, progress, failureCode = codeDeliveryJobPartial, job.Stage, job.Progress, "partial"
	} else if runErr != nil {
		status, stage, progress, failureCode = codeDeliveryJobFailed, job.Stage, job.Progress, codeDeliveryFailureCode(job.Stage, runErr)
	} else if result.Status == codeDeliveryJobConflict {
		status, stage, progress, failureCode = codeDeliveryJobConflict, job.Stage, job.Progress, "conflict"
	}
	conflicts, _ := json.Marshal(result.ConflictFiles)
	repositories, _ := json.Marshal(result.Repositories)
	now := time.Now()
	updates := map[string]any{
		"status": status, "stage": stage, "progress": progress, "result_commit": result.Commit,
		"result_type": result.ResultType, "failure_code": failureCode, "repository_results": string(repositories),
		"conflict_files": string(conflicts), "lease_owner": "", "lease_expires_at": nil, "completed_at": now,
	}
	if runErr != nil {
		updates["error_message"] = runErr.Error()
	} else {
		updates["error_message"] = ""
	}
	continueDevelopment := false
	finishErr := func() error {
		unlockLifecycle := codeSessionLifecycles.lock(job.SessionID)
		defer unlockLifecycle()
		continueDevelopment = status == codeDeliveryJobCompleted && shouldContinueCodeSessionAfterDelivery(global.DB, job.SessionID)
		return global.DB.Transaction(func(tx *gorm.DB) error {
			updated := tx.Model(&model.AICodeDeliveryJob{}).Where("id = ? AND lease_owner = ?", job.ID, runner.owner).Updates(updates)
			if updated.Error != nil {
				return updated.Error
			}
			if updated.RowsAffected != 1 {
				return errors.New("交付任务租约已失效")
			}
			if err := tx.Where("job_id = ? AND lease_owner = ?", job.ID, runner.owner).Delete(&model.AICodeDeliveryLease{}).Error; err != nil {
				return err
			}
			if status == codeDeliveryJobCompleted {
				return applyCodeSessionLifecycleFinalization(tx, job.SessionID, now, continueDevelopment)
			}
			return reopenCodeSessionLifecycle(tx, job.SessionID)
		})
	}()
	if finishErr != nil {
		global.LOG.Errorf("Finish Code delivery %d failed: %v", job.ID, finishErr)
		return
	}
	if status == codeDeliveryJobCompleted && !continueDevelopment {
		cleanupFinalizedCodeSessionWorktrees(job.SessionID)
	}
	auditStatus, detail := "success", result.Status
	if status == codeDeliveryJobPartial {
		auditStatus, detail = "partial", result.ErrorMessage
		if runErr != nil {
			detail = runErr.Error()
		}
	} else if status == codeDeliveryJobConflict {
		auditStatus, detail = "conflict", strings.Join(result.ConflictFiles, ", ")
	} else if runErr != nil {
		auditStatus, detail = "failed", runErr.Error()
	}
	duration := time.Duration(0)
	if job.StartedAt != nil {
		duration = time.Since(*job.StartedAt)
	}
	recordCodeAudit(job.UserID, job.ProjectID, job.SessionID, "worktree_merge", auditStatus, "delivery", detail, job.RequestIP, time.Now().Add(-duration), codeAuditMeta{"commit": result.Commit, "conflictFiles": result.ConflictFiles})
}

func cleanupFinalizedCodeSessionWorktrees(sessionID uint) {
	repositories, err := loadCodeSessionRepositories(sessionID)
	if err != nil {
		global.LOG.Warnf("Load finalized Code session %d repositories for cleanup failed: %v", sessionID, err)
		return
	}
	if len(repositories) == 0 {
		return
	}
	keys := make([]string, 0, len(repositories))
	for index := range repositories {
		repository := &repositories[index]
		keys = append(keys, codeDeliveryRepositoryKey(repository.SourceDir, repository.RemoteName, repository.TargetBranch))
	}
	owner := newCodeRepositoryLeaseOwner("delivery-cleanup")
	acquired, err := acquireCodeRepositoryLeases(owner, 0, keys)
	if err != nil || !acquired {
		global.LOG.Warnf("Acquire finalized Code session %d cleanup lease failed: %v", sessionID, err)
		return
	}
	defer func() { _ = releaseCodeRepositoryLeases(owner, keys) }()
	var session model.AIDevSession
	if err := global.DB.First(&session, sessionID).Error; err != nil {
		global.LOG.Warnf("Load finalized Code session %d for cleanup failed: %v", sessionID, err)
		return
	}
	if session.Status != codeSessionStatusDelivered {
		return
	}
	if err := cleanupDeliveredCodeSessionWorktrees(&session); err != nil {
		global.LOG.Warnf("Cleanup finalized Code session %d worktrees skipped: %v", sessionID, err)
	}
}

func codeDeliveryFailureCode(stage string, err error) string {
	if errors.Is(err, errCodeGitAuthentication) {
		return "authentication_failed"
	}
	if errors.Is(err, errCodeGitRepository) {
		return "repository_unavailable"
	}
	if errors.Is(err, errCodeGitNetwork) {
		return "network_failed"
	}
	if errors.Is(err, errCodePushRemoteAdvanced) {
		return "remote_advanced"
	}
	switch stage {
	case codeDeliveryStageSyncing:
		return "source_dirty"
	case codeDeliveryStageMerging:
		return "conflict"
	case codeDeliveryStageQualityCheck:
		return "quality_failed"
	case codeDeliveryStagePushing, codeDeliveryStageVerifying:
		return "push_failed"
	default:
		return "delivery_failed"
	}
}

func (runner *codeDeliveryRunner) run(jobID uint) {
	defer func() {
		runner.mu.Lock()
		delete(runner.queued, jobID)
		runner.mu.Unlock()
	}()
	job, keys, claimed, err := runner.claim(jobID)
	if err != nil {
		if claimed && job != nil {
			runner.finish(job, codeGitDeliveryResult{}, err)
		}
		return
	}
	if !claimed {
		return
	}
	if runner.isSessionCancelled(job.SessionID) {
		return
	}
	acquired, err := runner.acquireRepositoryLeases(job, keys)
	if err != nil || !acquired {
		runner.deferJob(job.ID, err)
		return
	}
	startedAt := time.Now()
	if err := global.DB.Model(&model.AICodeDeliveryJob{}).Where("id = ? AND lease_owner = ?", job.ID, runner.owner).
		Updates(map[string]any{"started_at": startedAt, "attempt": gorm.Expr("attempt + 1")}).Error; err != nil {
		runner.deferJob(job.ID, err)
		return
	}
	job.StartedAt = &startedAt
	job.Attempt++
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.heartbeat(ctx, job.ID, keys)
	reporter := func(stage string, progress int) {
		job.Stage, job.Progress = stage, progress
		runner.updateProgress(job.ID, stage, progress)
	}
	reporter(codeDeliveryStageQueued, 5)
	var session model.AIDevSession
	if err := global.DB.First(&session, job.SessionID).Error; err != nil {
		runner.finish(job, codeGitDeliveryResult{}, err)
		return
	}
	lease, err := codeExecutions.acquireSession(ctx, &session, codeExecutionDelivery, true)
	if err != nil {
		runner.finish(job, codeGitDeliveryResult{}, err)
		return
	}
	defer lease.Release()
	var result codeGitDeliveryResult
	if session.IsolationMode == codeIsolationMultiWorktree || hasCodeMultiRepositoryDelivery(session.ID) {
		lease.SetCancel(cancel)
		result, err = resumeCodeMultiRepositoryDeliveryWithProgress(&session, job.UserID, lease, reporter)
	} else {
		var delivery *model.AICodeDelivery
		delivery, result, err = prepareCodeSessionDeliveryWithProgress(&session, job.UserID, reporter)
		if err != nil || result.Status == "conflict" {
			runner.finish(job, result, err)
			return
		}
		qualitySession := session
		qualitySession.WorkDir = delivery.DeliveryWorkDir
		lease.SetCancel(cancel)
		if err := runCodeDeliveryQualityGate(&qualitySession, job.UserID, lease, reporter); err != nil {
			runner.finish(job, codeGitDeliveryResult{}, err)
			return
		}
		result, err = resumeCodeSessionDeliveryWithProgress(&session, job.UserID, reporter)
	}
	if runner.isSessionCancelled(job.SessionID) {
		return
	}
	runner.finish(job, result, err)
}

func loadCodeDeliveryJobView(sessionID uint) (*codeDeliveryJobView, error) {
	var job model.AICodeDeliveryJob
	if err := global.DB.Where("session_id = ?", sessionID).First(&job).Error; err != nil {
		return nil, err
	}
	var conflicts []string
	_ = json.Unmarshal([]byte(job.ConflictFiles), &conflicts)
	var repositories []codeRepositoryDeliveryResult
	_ = json.Unmarshal([]byte(job.RepositoryResults), &repositories)
	position := 0
	if job.Status == codeDeliveryJobQueued {
		var count int64
		_ = global.DB.Model(&model.AICodeDeliveryJob{}).Where(
			"status IN ? AND (created_at < ? OR (created_at = ? AND id < ?))",
			[]string{codeDeliveryJobQueued, codeDeliveryJobRunning}, job.CreatedAt, job.CreatedAt, job.ID,
		).Count(&count).Error
		position = int(count) + 1
	}
	pending := codeSessionPostSnapshotStatus{}
	if job.Status == codeDeliveryJobCompleted {
		pending = inspectCodeSessionPostSnapshotStatus(global.DB, sessionID)
	}
	return &codeDeliveryJobView{
		ID: job.ID, SessionID: job.SessionID, TaskID: job.TaskID, Status: job.Status, Stage: job.Stage, Progress: job.Progress,
		Attempt: job.Attempt, QueuePosition: position, TargetBranch: job.TargetBranch, ResultCommit: job.ResultCommit,
		ResultType: job.ResultType, FailureCode: job.FailureCode,
		HasPendingChanges: pending.HasChanges, HasPendingCommits: pending.HasCommits,
		HasUncommittedChanges: pending.HasUncommittedChanges,
		Repositories:          repositories,
		ErrorMessage:          job.ErrorMessage, ConflictFiles: conflicts, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
		StartedAt: job.StartedAt, CompletedAt: job.CompletedAt,
	}, nil
}
