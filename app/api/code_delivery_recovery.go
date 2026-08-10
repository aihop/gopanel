package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

const (
	codeDeliveryPrepared        = "prepared"
	codeDeliveryMerged          = "merged"
	codeDeliveryWorktreeCleaned = "worktree_cleaned"
	codeDeliveryCompleted       = "completed"
)

func loadOrCreateCodeDelivery(session *model.AIDevSession, userID uint) (*model.AICodeDelivery, error) {
	return loadOrCreateCodeDeliveryWithProgress(session, userID, nil)
}

func loadOrCreateCodeDeliveryWithProgress(session *model.AIDevSession, userID uint, report codeDeliveryProgressReporter) (*model.AICodeDelivery, error) {
	var delivery model.AICodeDelivery
	err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error
	if err == nil {
		return &delivery, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := validateCodeWorktreeDeliverySession(session); err != nil {
		return nil, err
	}
	if err := captureCodeDeliverySnapshot(session, userID); err != nil {
		return nil, err
	}
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		return nil, err
	}
	return &delivery, nil
}

func codeDeliverySessionSnapshot(delivery *model.AICodeDelivery) *model.AIDevSession {
	return &model.AIDevSession{
		ID: delivery.SessionID, UserID: delivery.UserID, ProjectID: delivery.ProjectID,
		WorkDir: delivery.WorkDir, SourceWorkDir: delivery.SourceWorkDir,
		WorktreeBranch: delivery.WorktreeBranch, TargetBranch: delivery.TargetBranch,
		BaseCommit: delivery.BaseCommit, RemoteName: delivery.RemoteName,
		RemoteBranch: delivery.RemoteBranch, RemoteCommit: delivery.RemoteCommit,
	}
}

func mergePreparedCodeDelivery(delivery *model.AICodeDelivery) (codeGitDeliveryResult, error) {
	return mergePreparedCodeDeliveryWithProgress(delivery, nil)
}

func mergePreparedCodeDeliveryWithProgress(delivery *model.AICodeDelivery, report codeDeliveryProgressReporter) (codeGitDeliveryResult, error) {
	snapshot := codeDeliverySessionSnapshot(delivery)
	if report != nil {
		report(codeDeliveryStageSyncing, 20)
	}
	targetBranch := snapshot.TargetBranch
	if targetBranch == "" {
		targetBranch, _ = runCodeGit(snapshot.SourceWorkDir, "branch", "--show-current")
	}
	targetCommit, err := codeLocalDeliveryTarget(snapshot.SourceWorkDir, targetBranch)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if err := global.DB.Model(delivery).Update("source_commit", targetCommit).Error; err != nil {
		return codeGitDeliveryResult{}, err
	}
	delivery.SourceCommit = targetCommit
	if err := prepareCodeDeliveryMergeWorktree(delivery, targetCommit); err != nil {
		return codeGitDeliveryResult{}, err
	}
	if report != nil {
		report(codeDeliveryStageMerging, 55)
	}
	if conflicts := codeGitConflictFiles(delivery.DeliveryWorkDir); len(conflicts) > 0 {
		return codeGitDeliveryResult{Status: "conflict", Branch: snapshot.WorktreeBranch, ConflictFiles: conflicts}, nil
	}
	status, err := runCodeGit(delivery.DeliveryWorkDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		return codeGitDeliveryResult{}, errors.New("交付 Worktree 存在未提交修改，请完成冲突修复并提交后重试")
	}
	if _, err := runCodeGit(delivery.DeliveryWorkDir, "merge-base", "--is-ancestor", targetCommit, "HEAD"); err != nil {
		if result, mergeErr := mergeCodeDeliveryCommit(delivery, targetCommit, snapshot.WorktreeBranch); result != nil || mergeErr != nil {
			if result != nil {
				return *result, mergeErr
			}
			return codeGitDeliveryResult{}, mergeErr
		}
	}
	if _, err := runCodeGit(delivery.DeliveryWorkDir, "merge-base", "--is-ancestor", delivery.WorktreeCommit, "HEAD"); err != nil {
		if result, mergeErr := mergeCodeDeliveryCommit(delivery, delivery.WorktreeCommit, snapshot.WorktreeBranch); result != nil || mergeErr != nil {
			if result != nil {
				return *result, mergeErr
			}
			return codeGitDeliveryResult{}, mergeErr
		}
	}
	commit, err := runCodeGit(delivery.DeliveryWorkDir, "rev-parse", "HEAD")
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	now := time.Now()
	if err := global.DB.Model(delivery).Updates(map[string]any{
		"status": codeDeliveryMerged, "merge_commit": commit, "merged_at": now, "error_message": "",
	}).Error; err != nil {
		return codeGitDeliveryResult{}, err
	}
	delivery.Status, delivery.MergeCommit, delivery.MergedAt = codeDeliveryMerged, commit, &now
	return codeGitDeliveryResult{Status: "merged", Commit: commit, Branch: snapshot.WorktreeBranch}, nil
}

// codeLocalDeliveryTarget 只读取交付基线 ref。
// 合并在独立的交付 Worktree 中进行，不触碰本地主仓工作区，
// 因此主仓有未提交变更或已切换到其它分支，都不影响本次交付。
func codeLocalDeliveryTarget(sourceDir, targetBranch string) (string, error) {
	targetBranch = strings.TrimSpace(targetBranch)
	if targetBranch == "" {
		return "", errors.New("本地主仓没有可用的交付目标分支")
	}
	commit, err := runCodeGit(sourceDir, "rev-parse", "refs/heads/"+targetBranch)
	if err != nil {
		return "", errors.New("本地主仓的交付目标分支 " + targetBranch + " 不可用")
	}
	return commit, nil
}

func mergeCodeDeliveryCommit(delivery *model.AICodeDelivery, commit, branch string) (*codeGitDeliveryResult, error) {
	if _, err := runCodeGit(
		delivery.DeliveryWorkDir,
		"-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.local",
		"-c", "commit.gpgsign=false", "merge", "--no-ff", "--no-edit", commit,
	); err != nil {
		conflicts := codeGitConflictFiles(delivery.DeliveryWorkDir)
		if len(conflicts) > 0 {
			return &codeGitDeliveryResult{Status: "conflict", Branch: branch, ConflictFiles: conflicts}, nil
		}
		return nil, err
	}
	return nil, nil
}

func updateCodeDeliveryMetadata(delivery *model.AICodeDelivery) error {
	now := time.Now()
	if delivery.Status == codeDeliveryWorktreeCleaned {
		return global.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.AIDevSession{}).Where("id = ?", delivery.SessionID).Updates(map[string]any{
				"work_dir": delivery.SourceWorkDir, "source_work_dir": "", "worktree_branch": "", "isolation_mode": "",
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.AITask{}).Where("session_id = ?", delivery.SessionID).
				Update("work_dir", delivery.SourceWorkDir).Error; err != nil {
				return err
			}
			return tx.Model(delivery).Updates(map[string]any{"status": codeDeliveryCompleted, "completed_at": now, "error_message": ""}).Error
		})
	}
	return global.DB.Model(delivery).Updates(map[string]any{"status": codeDeliveryCompleted, "completed_at": now, "error_message": ""}).Error
}

func cleanupCodeDeliveryWorktree(delivery *model.AICodeDelivery) error {
	if err := cleanupCodeIntegrationWorktree(delivery); err != nil {
		return err
	}
	snapshot := codeDeliverySessionSnapshot(delivery)
	if _, err := os.Stat(snapshot.WorkDir); err == nil {
		if _, sourceErr := os.Stat(snapshot.SourceWorkDir); errors.Is(sourceErr, os.ErrNotExist) {
			if !isManagedAISessionWorkDir(snapshot.WorkDir, snapshot.UserID) {
				return errors.New("会话 Worktree 不在 GoPanel 管理目录中")
			}
			if err := os.RemoveAll(snapshot.WorkDir); err != nil {
				return err
			}
		} else if sourceErr != nil {
			return sourceErr
		} else if err := cleanupCodeSessionWorktree(snapshot); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if branches, err := runCodeGit(snapshot.SourceWorkDir, "branch", "--list", snapshot.WorktreeBranch); err == nil && strings.TrimSpace(branches) != "" {
		if _, err := runCodeGit(snapshot.SourceWorkDir, "branch", "-d", "--", snapshot.WorktreeBranch); err != nil {
			return err
		}
	}
	if delivery.Status == codeDeliveryCompleted {
		return nil
	}
	return global.DB.Model(delivery).Updates(map[string]any{
		"status": codeDeliveryWorktreeCleaned, "error_message": "",
	}).Error
}

func resumeCodeSessionDelivery(session *model.AIDevSession, userID uint) (codeGitDeliveryResult, error) {
	return resumeCodeSessionDeliveryWithProgress(session, userID, nil)
}

func prepareCodeSessionDeliveryWithProgress(session *model.AIDevSession, userID uint, report codeDeliveryProgressReporter) (*model.AICodeDelivery, codeGitDeliveryResult, error) {
	delivery, err := loadOrCreateCodeDeliveryWithProgress(session, userID, report)
	if err != nil {
		return nil, codeGitDeliveryResult{}, err
	}
	result := codeGitDeliveryResult{Status: "merged", Commit: delivery.MergeCommit, Branch: delivery.WorktreeBranch}
	if delivery.Status == codeDeliveryPrepared {
		result, err = mergePreparedCodeDeliveryWithProgress(delivery, report)
		if err != nil || result.Status == "conflict" {
			return delivery, result, err
		}
	}
	if delivery.Status == codeDeliveryMerged {
		if err := ensureCodeDeliveryWorktree(delivery, delivery.MergeCommit); err != nil {
			return delivery, codeGitDeliveryResult{}, err
		}
	}
	return delivery, result, nil
}

func resumeCodeSessionDeliveryWithProgress(session *model.AIDevSession, userID uint, report codeDeliveryProgressReporter) (codeGitDeliveryResult, error) {
	delivery, err := loadOrCreateCodeDeliveryWithProgress(session, userID, report)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	result, err := integrateCodeDeliveryLocallyWithProgress(delivery, report)
	if err != nil || result.Status == "conflict" {
		result.Repositories = []codeRepositoryDeliveryResult{codeSingleRepositoryDeliveryResult(delivery, result)}
		return result, err
	}
	if delivery.Status == codeDeliveryMerged {
		if report != nil {
			report(codeDeliveryStageCleaning, 90)
		}
		if codeExecutions.hasSessionKind(session.ID, codeExecutionInteractive) {
			delivery.ErrorMessage = ""
		} else if err := cleanupCodeDeliveryWorktree(delivery); err != nil {
			delivery.ErrorMessage = err.Error()
			_ = global.DB.Model(delivery).Update("error_message", delivery.ErrorMessage).Error
		} else {
			delivery.Status = codeDeliveryWorktreeCleaned
		}
	}
	if delivery.Status == codeDeliveryMerged || delivery.Status == codeDeliveryWorktreeCleaned {
		if err := updateCodeDeliveryMetadata(delivery); err != nil {
			return codeGitDeliveryResult{}, err
		}
		delivery.Status = codeDeliveryCompleted
	}
	result.Repositories = []codeRepositoryDeliveryResult{codeSingleRepositoryDeliveryResult(delivery, result)}
	return result, nil
}

func codeSingleRepositoryDeliveryResult(delivery *model.AICodeDelivery, result codeGitDeliveryResult) codeRepositoryDeliveryResult {
	pushStatus := delivery.PushStatus
	if !codeDeliveryHasRemote(delivery.RemoteName, deliveryRemoteBranch(delivery.RemoteBranch, delivery.TargetBranch)) {
		pushStatus = "local"
	}
	return codeRepositoryDeliveryResult{
		RepositoryID: "session", RepositoryName: filepath.Base(delivery.SourceWorkDir), Status: result.Status,
		Branch: delivery.WorktreeBranch, TargetBranch: delivery.TargetBranch, Remote: delivery.RemoteName,
		RemoteBranch: deliveryRemoteBranch(delivery.RemoteBranch, delivery.TargetBranch), Commit: result.Commit,
		SnapshotReady:   strings.TrimSpace(delivery.WorktreeCommit) != "",
		MergeReady:      strings.TrimSpace(delivery.MergeCommit) != "",
		SourceAppliedAt:  delivery.SourceAppliedAt,
		LocalSynced:      delivery.SourceAppliedAt != nil,
		LocalSyncError:   delivery.LocalSyncError,
		LocalSyncCommand: codeDeliveryLocalSyncCommand(delivery.SourceWorkDir, delivery.MergeCommit),
		PushStatus:       pushStatus, PushedCommit: delivery.PushedCommit, ErrorMessage: delivery.PushError,
		ConflictFiles:    result.ConflictFiles,
	}
}
