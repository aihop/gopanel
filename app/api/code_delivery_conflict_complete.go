package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

func completeCodeDeliveryConflict(job *model.AICodeDeliveryJob, contexts []codeDeliveryConflictContext) error {
	if len(contexts) == 0 {
		return errors.New("当前没有可完成的冲突仓库")
	}
	for index := range contexts {
		if conflicts := discoverCodeDeliveryConflictFiles(contexts[index].WorkDir); len(conflicts) > 0 {
			return fmt.Errorf("仓库 %s 仍有 %d 个冲突文件未解决", contexts[index].Name, len(conflicts))
		}
	}
	commits := make([]string, len(contexts))
	for index := range contexts {
		commit, err := finalizeCodeDeliveryConflictCommit(&contexts[index])
		if err != nil {
			return fmt.Errorf("完成仓库 %s 的冲突合并失败：%w", contexts[index].Name, err)
		}
		commits[index] = commit
	}
	for index := range contexts {
		if contexts[index].Repository == nil {
			continue
		}
		if err := cleanupExposedCodeRepositoryConflict(contexts[index].Repository); err != nil {
			return err
		}
	}
	now := time.Now()
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		for index := range contexts {
			context := &contexts[index]
			if context.Delivery != nil {
				if err := tx.Model(&model.AICodeDelivery{}).Where("id = ?", context.Delivery.ID).Updates(map[string]any{
					"status": codeDeliveryMerged, "merge_commit": commits[index], "merged_at": now, "error_message": "",
				}).Error; err != nil {
					return err
				}
			} else if err := tx.Model(&model.AIDevSessionRepository{}).Where("id = ?", context.Repository.ID).Updates(map[string]any{
				"status": codeDeliveryMerged, "merge_commit": commits[index], "merged_at": now, "error_message": "",
			}).Error; err != nil {
				return err
			}
		}
		return requeueCodeDeliveryConflictJob(tx, job)
	}); err != nil {
		return err
	}
	enqueueCodeDelivery(job.ID)
	return nil
}

func confirmManualCodeDeliveryConflict(job *model.AICodeDeliveryJob, contexts []codeDeliveryConflictContext) error {
	if len(contexts) == 0 {
		return errors.New("当前没有可核验的冲突仓库")
	}
	for index := range contexts {
		if err := verifyManualCodeDeliveryConflict(&contexts[index]); err != nil {
			return err
		}
	}
	for index := range contexts {
		if err := abortCodeDeliveryConflictWorktree(contexts[index].WorkDir); err != nil {
			return err
		}
	}
	for index := range contexts {
		if contexts[index].Repository == nil {
			continue
		}
		if err := cleanupExposedCodeRepositoryConflict(contexts[index].Repository); err != nil {
			return err
		}
	}
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		for index := range contexts {
			if contexts[index].Repository == nil {
				continue
			}
			if err := tx.Model(&model.AIDevSessionRepository{}).Where("id = ?", contexts[index].Repository.ID).Updates(map[string]any{
				"status": codeDeliveryPrepared, "error_message": "",
			}).Error; err != nil {
				return err
			}
		}
		return requeueCodeDeliveryConflictJob(tx, job)
	}); err != nil {
		return err
	}
	enqueueCodeDelivery(job.ID)
	return nil
}

func verifyManualCodeDeliveryConflict(context *codeDeliveryConflictContext) error {
	if context == nil || strings.TrimSpace(context.SourceDir) == "" || strings.TrimSpace(context.TargetBranch) == "" {
		return errors.New("手动合并核验所需的仓库信息不可用")
	}
	targetCommit, err := runCodeGit(context.SourceDir, "rev-parse", "refs/heads/"+context.TargetBranch)
	if err != nil {
		return fmt.Errorf("仓库 %s 的目标分支 %s 不可用", context.Name, context.TargetBranch)
	}
	for _, commit := range []string{context.SourceCommit, context.TaskCommit} {
		if _, err := runCodeGit(context.SourceDir, "merge-base", "--is-ancestor", commit, targetCommit); err != nil {
			return fmt.Errorf("仓库 %s 的目标分支 %s 尚未包含任务分支 %s，请完成合并后再核验", context.Name, context.TargetBranch, context.Branch)
		}
	}
	return nil
}

func abortCodeDeliveryConflictWorktree(workDir string) error {
	if _, err := runCodeGit(workDir, "rev-parse", "-q", "--verify", "MERGE_HEAD"); err == nil {
		if _, err := runCodeGit(workDir, "merge", "--abort"); err != nil {
			return fmt.Errorf("清理网页冲突现场失败：%w", err)
		}
		return nil
	}
	status, err := runCodeGit(workDir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) != "" {
		return errors.New("网页冲突 Worktree 已被其他操作修改，请刷新后重试")
	}
	return nil
}

func requeueCodeDeliveryConflictJob(tx *gorm.DB, job *model.AICodeDeliveryJob) error {
	updates := map[string]any{
		"status": codeDeliveryJobQueued, "stage": codeDeliveryStageQueued, "progress": 0,
		"result_commit": "", "result_type": "", "failure_code": "", "error_message": "",
		"conflict_files": "", "repository_results": "", "lease_owner": "", "lease_expires_at": nil,
		"started_at": nil, "completed_at": nil,
	}
	updated := tx.Model(&model.AICodeDeliveryJob{}).Where("id = ? AND status = ?", job.ID, codeDeliveryJobConflict).Updates(updates)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return errors.New("交付任务状态已变化，请刷新后重试")
	}
	var session model.AIDevSession
	if err := tx.First(&session, job.SessionID).Error; err != nil {
		return err
	}
	return markCodeSessionDelivering(tx, &session)
}

func cleanupExposedCodeRepositoryConflict(repository *model.AIDevSessionRepository) error {
	if repository == nil || strings.TrimSpace(repository.WorktreeDir) == "" {
		return nil
	}
	if _, err := runCodeGit(repository.WorktreeDir, "rev-parse", "-q", "--verify", "MERGE_HEAD"); err != nil {
		return nil
	}
	if _, err := runCodeGit(repository.WorktreeDir, "merge", "--abort"); err != nil {
		return fmt.Errorf("清理任务 Worktree 的冲突现场失败：%w", err)
	}
	head, err := runCodeGit(repository.WorktreeDir, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != strings.TrimSpace(repository.WorktreeCommit) {
		return errors.New("任务 Worktree 在冲突处理期间已变化，请刷新后重试")
	}
	return nil
}

func finalizeCodeDeliveryConflictCommit(context *codeDeliveryConflictContext) (string, error) {
	if _, err := runCodeGit(context.WorkDir, "rev-parse", "-q", "--verify", "MERGE_HEAD"); err == nil {
		if _, err := runCodeGit(
			context.WorkDir, "-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.local",
			"-c", "commit.gpgsign=false", "commit", "--no-edit",
		); err != nil {
			return "", err
		}
	}
	commit, err := runCodeGit(context.WorkDir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	for _, parent := range []string{context.SourceCommit, context.TaskCommit} {
		if _, err := runCodeGit(context.WorkDir, "merge-base", "--is-ancestor", parent, commit); err != nil {
			return "", errors.New("冲突解决提交没有同时包含主线与任务分支，请重新发起交付")
		}
	}
	return commit, nil
}
