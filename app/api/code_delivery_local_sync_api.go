package api

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func SyncCodeSessionDeliveryLocal(c fiber.Ctx) error {
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if !req.Confirm {
		return c.JSON(e.Fail(errors.New("合入本地主仓需要明确确认")))
	}
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	session, err := getCodeDeliveryLocalSyncSessionContext(c, claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionDelivery, false)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	defer lease.Release()
	result, err := syncCodeSessionDeliveryLocal(session)
	if err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "git_local_sync", "failed", "delivery", err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	auditStatus := "success"
	if result.Status != "completed" {
		auditStatus = "partial"
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "git_local_sync", auditStatus, "delivery", result.Status, c.IP(), startedAt, codeAuditMeta{"repositories": len(result.Repositories)})
	return c.JSON(e.Succ(result))
}

func getCodeDeliveryLocalSyncSessionContext(c fiber.Ctx, claims *token.CustomClaims) (*model.AIDevSession, error) {
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return nil, errors.New("会话 ID 无效")
	}
	return getCodeDeliveryLocalSyncSession(uint(sessionID), claims)
}

func getCodeDeliveryLocalSyncSession(sessionID uint, claims *token.CustomClaims) (*model.AIDevSession, error) {
	session, err := getAISessionWithPermission(sessionID, claims)
	if err != nil {
		return nil, err
	}
	if hasCodeMultiRepositoryDelivery(session.ID) {
		if err := validateCodeDeliveryPushSession(session, claims); err != nil {
			return nil, err
		}
		return session, nil
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		return nil, err
	}
	if delivery.UserID != session.UserID || delivery.ProjectID != session.ProjectID {
		return nil, errors.New("交付记录与当前会话不匹配")
	}
	sourceDirs, err := getAISessionSourceDirs(session.ProjectID, claims)
	if err != nil || len(sourceDirs) == 0 {
		return nil, errors.New("交付源目录不可用")
	}
	storedSource, err := filepath.EvalSymlinks(filepath.Clean(delivery.SourceWorkDir))
	if err != nil || !repositoryWithinSourceDirs(storedSource, sourceDirs) {
		return nil, errors.New("交付源目录与项目配置不一致")
	}
	if err := validateAIProjectWorkDirForClaims(storedSource, claims); err != nil {
		return nil, err
	}
	return session, nil
}

func syncCodeSessionDeliveryLocal(session *model.AIDevSession) (codeDeliveryLocalSyncResult, error) {
	if session == nil || session.ID == 0 {
		return codeDeliveryLocalSyncResult{}, errors.New("交付会话不可用")
	}
	var job model.AICodeDeliveryJob
	if err := global.DB.Where("session_id = ?", session.ID).First(&job).Error; err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	ready, err := codeDeliveryLocalSyncReady(session, &job)
	if err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	if !ready {
		return codeDeliveryLocalSyncResult{}, errors.New("当前交付尚未完成，不能合入本地主仓")
	}
	var result codeDeliveryLocalSyncResult
	if hasCodeMultiRepositoryDelivery(session.ID) {
		result, err = syncCodeMultiRepositoryDeliveryLocal(session)
	} else {
		result, err = syncCodeSingleRepositoryDeliveryLocal(session)
	}
	if err != nil || result.Status != "completed" || job.Status == codeDeliveryJobCompleted {
		return result, err
	}
	if err := completeRecoveredCodeDeliveryLocalSync(session, &job); err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	return result, nil
}

func codeDeliveryLocalSyncReady(session *model.AIDevSession, job *model.AICodeDeliveryJob) (bool, error) {
	if job != nil && job.Status == codeDeliveryJobCompleted {
		return true, nil
	}
	if session == nil || job == nil || !hasCodeMultiRepositoryDelivery(session.ID) ||
		(job.Status != codeDeliveryJobFailed && job.Status != codeDeliveryJobPartial) ||
		job.Stage != codeDeliveryStageCleaning {
		return false, nil
	}
	repositories, err := loadCodeDeliverySessionRepositories(session)
	if err != nil {
		return false, err
	}
	return codeMultiRepositoryDeliveryFrozen(repositories), nil
}

func completeRecoveredCodeDeliveryLocalSync(session *model.AIDevSession, job *model.AICodeDeliveryJob) error {
	repositories, err := loadCodeDeliverySessionRepositories(session)
	if err != nil {
		return err
	}
	repositoryResults, err := json.Marshal(codeStoredRepositoryDeliveryResults(repositories))
	if err != nil {
		return err
	}
	completedAt := time.Now()
	continueDevelopment := shouldContinueCodeSessionAfterDelivery(global.DB, session.ID)
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		updated := tx.Model(&model.AICodeDeliveryJob{}).Where(
			"id = ? AND status IN ?", job.ID, []string{codeDeliveryJobFailed, codeDeliveryJobPartial},
		).Updates(map[string]any{
			"status": codeDeliveryJobCompleted, "stage": codeDeliveryStageCompleted, "progress": 100,
			"result_type":  codeMultiRepositoryResultType(codeStoredRepositoryDeliveryResults(repositories)),
			"failure_code": "", "error_message": "", "conflict_files": "",
			"repository_results": string(repositoryResults), "completed_at": completedAt,
		})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return errors.New("交付状态已变化，请刷新后重试")
		}
		return applyCodeSessionLifecycleFinalization(tx, session.ID, completedAt, continueDevelopment)
	}); err != nil {
		return err
	}
	if !continueDevelopment {
		cleanupFinalizedCodeSessionWorktrees(session.ID)
	}
	return nil
}

func syncCodeSingleRepositoryDeliveryLocal(session *model.AIDevSession) (codeDeliveryLocalSyncResult, error) {
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	release, err := acquireCodeDeliveryLocalSyncLeases([]codeDeliveryLocalSyncTarget{{
		SourceDir: delivery.SourceWorkDir, RemoteName: delivery.RemoteName, TargetBranch: delivery.TargetBranch,
	}})
	if err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	defer release()
	if delivery.SourceAppliedAt == nil && strings.TrimSpace(delivery.MergeCommit) != "" {
		syncErr := syncCodeDeliveryTargetOnDemand(delivery.SourceWorkDir, delivery.TargetBranch, delivery.MergeCommit)
		if syncErr != nil {
			if conflicts := codeDeliveryLocalSyncConflictFiles(syncErr); len(conflicts) > 0 {
				if err := prepareCodeDeliveryLocalSyncConflict(session, &delivery, nil, conflicts); err != nil {
					return codeDeliveryLocalSyncResult{}, err
				}
				view := codeDeliveryLocalSyncDeliveryView(&delivery)
				view.ConflictFiles = conflicts
				return codeDeliveryLocalSyncResult{Status: "conflict", Repositories: []codeDeliveryLocalSyncRepository{view}}, nil
			}
			if err := persistCodeDeliveryLocalSync(&delivery, nil, codeDeliveryLocalSyncReason(syncErr.Error())); err != nil {
				return codeDeliveryLocalSyncResult{}, err
			}
		} else {
			appliedAt := time.Now()
			if err := persistCodeDeliveryLocalSync(&delivery, &appliedAt, ""); err != nil {
				return codeDeliveryLocalSyncResult{}, err
			}
		}
	}
	return summarizeCodeDeliveryLocalSync([]codeDeliveryLocalSyncRepository{codeDeliveryLocalSyncDeliveryView(&delivery)}), nil
}

func syncCodeMultiRepositoryDeliveryLocal(session *model.AIDevSession) (codeDeliveryLocalSyncResult, error) {
	repositories, err := loadCodeDeliverySessionRepositories(session)
	if err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	repositories, err = codeDeliveryRepositoriesInOrder(repositories, false)
	if err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	targets := make([]codeDeliveryLocalSyncTarget, 0, len(repositories))
	for index := range repositories {
		if strings.TrimSpace(repositories[index].MergeCommit) == "" {
			continue
		}
		targets = append(targets, codeDeliveryLocalSyncTarget{
			SourceDir: repositories[index].SourceDir, RemoteName: repositories[index].RemoteName,
			TargetBranch: repositories[index].TargetBranch,
		})
	}
	if len(targets) == 0 {
		return codeDeliveryLocalSyncResult{}, errors.New("当前交付没有可合入的提交")
	}
	release, err := acquireCodeDeliveryLocalSyncLeases(targets)
	if err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	defer release()
	if hasCodeDeliveryNestedRepositories(repositories) {
		for index := range repositories {
			repository := &repositories[index]
			if repository.SourceAppliedAt != nil || strings.TrimSpace(repository.MergeCommit) == "" {
				continue
			}
			if err := applyCodeRepositoryLocalSync(repository, repositories); err != nil {
				return summarizeCodeDeliveryLocalSync(codeDeliveryLocalSyncRepositoryViews(repositories)), nil
			}
		}
		result := summarizeCodeDeliveryLocalSync(codeDeliveryLocalSyncRepositoryViews(repositories))
		if result.Status != "completed" {
			return result, nil
		}
		if err := completeCodeMultiRepositorySources(repositories); err != nil {
			return codeDeliveryLocalSyncResult{}, err
		}
		if err := cleanupCodeMultiRepositoryIntegrationWorktrees(session, repositories); err != nil {
			return codeDeliveryLocalSyncResult{}, err
		}
		return result, nil
	}
	for index := range repositories {
		repository := &repositories[index]
		if repository.SourceAppliedAt != nil || strings.TrimSpace(repository.MergeCommit) == "" {
			continue
		}
		syncErr := syncCodeDeliveryTargetOnDemand(repository.SourceDir, repository.TargetBranch, repository.MergeCommit)
		if syncErr != nil {
			if conflicts := codeDeliveryLocalSyncConflictFiles(syncErr); len(conflicts) > 0 {
				if err := prepareCodeDeliveryLocalSyncConflict(session, nil, repository, conflicts); err != nil {
					return codeDeliveryLocalSyncResult{}, err
				}
				views := codeDeliveryLocalSyncRepositoryViews(repositories)
				for viewIndex := range views {
					if views[viewIndex].RepositoryID == codeSessionRepositoryID(repository.ID) {
						views[viewIndex].ConflictFiles = conflicts
					}
				}
				return codeDeliveryLocalSyncResult{Status: "conflict", Repositories: views}, nil
			}
			if err := persistCodeRepositoryLocalSync(repository, nil, codeDeliveryLocalSyncReason(syncErr.Error())); err != nil {
				return codeDeliveryLocalSyncResult{}, err
			}
			continue
		}
		appliedAt := time.Now()
		if err := persistCodeRepositoryLocalSync(repository, &appliedAt, ""); err != nil {
			return codeDeliveryLocalSyncResult{}, err
		}
	}
	result := summarizeCodeDeliveryLocalSync(codeDeliveryLocalSyncRepositoryViews(repositories))
	if result.Status != "completed" {
		return result, nil
	}
	if err := completeCodeMultiRepositorySources(repositories); err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	if err := cleanupCodeMultiRepositoryIntegrationWorktrees(session, repositories); err != nil {
		return codeDeliveryLocalSyncResult{}, err
	}
	return result, nil
}

func hasCodeDeliveryNestedRepositories(repositories []model.AIDevSessionRepository) bool {
	for index := range repositories {
		if strings.TrimSpace(repositories[index].ParentSourceDir) != "" || strings.TrimSpace(repositories[index].GitlinkPath) != "" {
			return true
		}
	}
	return false
}

func codeDeliveryLocalSyncRepositoryViews(repositories []model.AIDevSessionRepository) []codeDeliveryLocalSyncRepository {
	views := make([]codeDeliveryLocalSyncRepository, 0, len(repositories))
	for index := range repositories {
		if strings.TrimSpace(repositories[index].MergeCommit) != "" {
			views = append(views, codeDeliveryLocalSyncRepositoryView(&repositories[index]))
		}
	}
	return views
}

type codeDeliveryLocalSyncTarget struct {
	SourceDir, RemoteName, TargetBranch string
}

func acquireCodeDeliveryLocalSyncLeases(targets []codeDeliveryLocalSyncTarget) (func(), error) {
	keys := make([]string, 0, len(targets))
	for _, target := range targets {
		keys = append(keys, codeDeliveryRepositoryKey(target.SourceDir, target.RemoteName, target.TargetBranch))
	}
	owner := newCodeRepositoryLeaseOwner("local-sync")
	acquired, err := acquireCodeRepositoryLeases(owner, 0, keys)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, errors.New("项目主仓正在执行其它同步或交付，请稍后重试")
	}
	ctx, cancel := context.WithCancel(context.Background())
	go heartbeatCodeRepositoryLeases(ctx, owner, keys)
	return func() {
		cancel()
		_ = releaseCodeRepositoryLeases(owner, keys)
	}, nil
}

func codeDeliveryLocalSyncDeliveryView(delivery *model.AICodeDelivery) codeDeliveryLocalSyncRepository {
	return codeDeliveryLocalSyncRepository{
		RepositoryID: "session", RepositoryName: filepath.Base(delivery.SourceWorkDir), Commit: delivery.MergeCommit,
		LocalSynced: delivery.SourceAppliedAt != nil, LocalSyncError: delivery.LocalSyncError,
		LocalSyncCommand: codeDeliveryLocalSyncCommand(delivery.SourceWorkDir, delivery.TargetBranch, delivery.MergeCommit),
	}
}

func codeDeliveryLocalSyncRepositoryView(repository *model.AIDevSessionRepository) codeDeliveryLocalSyncRepository {
	return codeDeliveryLocalSyncRepository{
		RepositoryID: codeSessionRepositoryID(repository.ID), RepositoryName: repository.LinkName, Commit: repository.MergeCommit,
		LocalSynced: repository.SourceAppliedAt != nil, LocalSyncError: repository.LocalSyncError,
		LocalSyncCommand: codeDeliveryLocalSyncCommand(repository.SourceDir, repository.TargetBranch, repository.MergeCommit),
	}
}

func summarizeCodeDeliveryLocalSync(repositories []codeDeliveryLocalSyncRepository) codeDeliveryLocalSyncResult {
	synced := 0
	for _, repository := range repositories {
		if repository.LocalSynced {
			synced++
		}
	}
	status := "blocked"
	if synced == len(repositories) && len(repositories) > 0 {
		status = "completed"
	} else if synced > 0 {
		status = "partial"
	}
	return codeDeliveryLocalSyncResult{Status: status, Repositories: repositories}
}
