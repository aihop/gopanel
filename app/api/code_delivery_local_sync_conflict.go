package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

const codeDeliveryFailureLocalSyncConflict = "local_sync_conflict"

func prepareCodeDeliveryLocalSyncConflict(
	session *model.AIDevSession,
	delivery *model.AICodeDelivery,
	repository *model.AIDevSessionRepository,
	conflicts []string,
) error {
	if session == nil || (delivery == nil && repository == nil) {
		return errors.New("本地同步冲突上下文不可用")
	}
	conflicts = normalizedCodeConflictFiles(conflicts)
	if len(conflicts) == 0 {
		return errors.New("本地同步冲突文件不可用")
	}
	if delivery != nil {
		if err := prepareCodeDeliveryLocalSyncConflictWorktree(
			delivery.SourceWorkDir, delivery.TargetBranch, delivery.MergeCommit,
			func(targetCommit string) error { return resetCodeDeliveryWorktree(delivery, targetCommit) },
			func() string { return delivery.DeliveryWorkDir },
		); err != nil {
			return err
		}
	} else {
		repositories, err := loadCodeDeliverySessionRepositories(session)
		if err != nil {
			return err
		}
		if err := ensureCodeRepositoryIntegrationWorktree(session, repository, repositories); err != nil {
			return err
		}
		if err := prepareCodeDeliveryLocalSyncConflictWorktree(
			repository.SourceDir, repository.TargetBranch, repository.MergeCommit,
			func(targetCommit string) error {
				_, _ = runCodeGit(repository.IntegrationWorkDir, "merge", "--abort")
				if _, err := runCodeGit(repository.IntegrationWorkDir, "reset", "--hard", targetCommit); err != nil {
					return err
				}
				_, err := runCodeGit(repository.IntegrationWorkDir, "clean", "-d", "-f")
				return err
			},
			func() string { return repository.IntegrationWorkDir },
		); err != nil {
			return err
		}
	}
	return markCodeDeliveryLocalSyncConflict(session.ID, delivery, repository, conflicts)
}

func prepareCodeDeliveryLocalSyncConflictWorktree(
	sourceDir, targetBranch, deliveryCommit string,
	reset func(string) error,
	workDir func() string,
) error {
	targetCommit, err := runCodeGit(sourceDir, "rev-parse", "refs/heads/"+strings.TrimSpace(targetBranch))
	if err != nil {
		return errors.New("本地主仓目标分支不可用")
	}
	if err := reset(strings.TrimSpace(targetCommit)); err != nil {
		return err
	}
	if _, err := runCodeGit(
		workDir(), "-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.local",
		"-c", "commit.gpgsign=false", "merge", "--no-edit", deliveryCommit,
	); err == nil || len(codeGitConflictFiles(workDir())) == 0 {
		return errors.New("本地同步冲突现场创建失败，请刷新后重试")
	}
	return nil
}

func markCodeDeliveryLocalSyncConflict(
	sessionID uint,
	delivery *model.AICodeDelivery,
	repository *model.AIDevSessionRepository,
	conflicts []string,
) error {
	var job model.AICodeDeliveryJob
	if err := global.DB.Where("session_id = ?", sessionID).First(&job).Error; err != nil {
		return err
	}
	var results []codeRepositoryDeliveryResult
	if delivery != nil {
		result := codeSingleRepositoryDeliveryResult(delivery, codeGitDeliveryResult{
			Status: codeDeliveryJobConflict, Commit: delivery.MergeCommit, ConflictFiles: conflicts,
		})
		results = []codeRepositoryDeliveryResult{result}
	} else {
		repositories, err := loadCodeDeliverySessionRepositories(&model.AIDevSession{ID: sessionID})
		if err != nil {
			return err
		}
		for index := range repositories {
			result := codeStoredRepositoryDeliveryResult(&repositories[index])
			if repository != nil && repositories[index].ID == repository.ID {
				result.Status, result.ConflictFiles = codeDeliveryJobConflict, conflicts
			}
			results = append(results, result)
		}
	}
	encodedConflicts, _ := json.Marshal(conflicts)
	encodedResults, _ := json.Marshal(results)
	now := time.Now()
	updated := global.DB.Model(&job).Where("status = ?", codeDeliveryJobCompleted).Updates(map[string]any{
		"status": codeDeliveryJobConflict, "stage": codeDeliveryStageMerging, "progress": 95,
		"failure_code": codeDeliveryFailureLocalSyncConflict, "error_message": "",
		"conflict_files": string(encodedConflicts), "repository_results": string(encodedResults), "completed_at": now,
	})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return errors.New("交付状态已变化，请刷新后重试")
	}
	return nil
}

func isCodeDeliveryLocalSyncConflict(job *model.AICodeDeliveryJob) bool {
	return job != nil && job.Status == codeDeliveryJobConflict && job.FailureCode == codeDeliveryFailureLocalSyncConflict
}

func completeCodeDeliveryLocalSyncConflict(job *model.AICodeDeliveryJob, contexts []codeDeliveryConflictContext) error {
	if len(contexts) == 0 {
		return errors.New("当前没有可完成的本地同步冲突")
	}
	for index := range contexts {
		if conflicts := discoverCodeDeliveryConflictFiles(contexts[index].WorkDir); len(conflicts) > 0 {
			return fmt.Errorf("仓库 %s 仍有 %d 个冲突文件未解决", contexts[index].Name, len(conflicts))
		}
	}
	targets := make([]codeDeliveryLocalSyncTarget, 0, len(contexts))
	for index := range contexts {
		targets = append(targets, codeDeliveryLocalSyncTarget{SourceDir: contexts[index].SourceDir, TargetBranch: contexts[index].TargetBranch})
	}
	release, err := acquireCodeDeliveryLocalSyncLeases(targets)
	if err != nil {
		return err
	}
	defer release()
	for index := range contexts {
		commit, err := finalizeCodeDeliveryConflictCommit(&contexts[index])
		if err != nil {
			return fmt.Errorf("完成仓库 %s 的冲突合并失败：%w", contexts[index].Name, err)
		}
		if err := syncCodeDeliveryTargetOnDemand(contexts[index].SourceDir, contexts[index].TargetBranch, commit); err != nil {
			return fmt.Errorf("更新仓库 %s 的目标分支失败：%w", contexts[index].Name, err)
		}
	}
	now := time.Now()
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		for index := range contexts {
			context := &contexts[index]
			if context.Delivery != nil {
				if err := tx.Model(&model.AICodeDelivery{}).Where("id = ?", context.Delivery.ID).Updates(map[string]any{
					"source_applied_at": now, "local_sync_error": "",
				}).Error; err != nil {
					return err
				}
			} else if err := tx.Model(&model.AIDevSessionRepository{}).Where("id = ?", context.Repository.ID).Updates(map[string]any{
				"source_applied_at": now, "local_sync_error": "",
			}).Error; err != nil {
				return err
			}
		}
		results, err := codeDeliveryLocalSyncConflictResults(tx, job.SessionID, contexts)
		if err != nil {
			return err
		}
		return completeCodeDeliveryLocalSyncConflictJob(tx, job, results)
	}); err != nil {
		return err
	}
	cleanupCodeDeliveryLocalSyncConflictWorktrees(job.SessionID, contexts)
	return nil
}

func codeDeliveryLocalSyncConflictResults(
	tx *gorm.DB,
	sessionID uint,
	contexts []codeDeliveryConflictContext,
) (string, error) {
	var results []codeRepositoryDeliveryResult
	if len(contexts) == 1 && contexts[0].Delivery != nil {
		var delivery model.AICodeDelivery
		if err := tx.Where("id = ?", contexts[0].Delivery.ID).First(&delivery).Error; err != nil {
			return "", err
		}
		results = []codeRepositoryDeliveryResult{codeSingleRepositoryDeliveryResult(&delivery, codeGitDeliveryResult{
			Status: codeDeliveryJobCompleted, Commit: delivery.MergeCommit,
		})}
	} else {
		var repositories []model.AIDevSessionRepository
		if err := tx.Where("session_id = ?", sessionID).Order("id asc").Find(&repositories).Error; err != nil {
			return "", err
		}
		results = codeStoredRepositoryDeliveryResults(repositories)
	}
	encoded, err := json.Marshal(results)
	return string(encoded), err
}

func completeCodeDeliveryLocalSyncConflictJob(
	tx *gorm.DB,
	job *model.AICodeDeliveryJob,
	repositoryResults string,
) error {
	now := time.Now()
	updated := tx.Model(&model.AICodeDeliveryJob{}).Where(
		"id = ? AND status = ? AND failure_code = ?", job.ID, codeDeliveryJobConflict, codeDeliveryFailureLocalSyncConflict,
	).Updates(map[string]any{
		"status": codeDeliveryJobCompleted, "stage": codeDeliveryStageCompleted, "progress": 100,
		"failure_code": "", "error_message": "", "conflict_files": "",
		"repository_results": repositoryResults, "completed_at": now,
	})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return errors.New("本地同步冲突状态已变化，请刷新后重试")
	}
	return nil
}

func cleanupCodeDeliveryLocalSyncConflictWorktrees(sessionID uint, contexts []codeDeliveryConflictContext) {
	for index := range contexts {
		context := &contexts[index]
		var err error
		if context.Delivery != nil {
			err = cleanupCodeIntegrationWorktree(context.Delivery)
			if err == nil {
				err = global.DB.Model(context.Delivery).Update("delivery_work_dir", "").Error
			}
		} else if context.Repository != nil {
			_, err = runCodeGit(context.SourceDir, "worktree", "remove", "--force", context.WorkDir)
			if err == nil {
				_ = global.DB.Model(context.Repository).Update("integration_work_dir", "").Error
			}
		}
		if err != nil && global.LOG != nil {
			global.LOG.Warnf("Cleanup local sync conflict worktree for session %d failed: %v", sessionID, err)
		}
	}
}
