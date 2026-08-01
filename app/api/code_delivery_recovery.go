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
	targetCommit, err := refreshCodeRepositoryTarget(snapshot.SourceWorkDir, targetBranch, snapshot.RemoteName)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if err := global.DB.Model(delivery).Update("remote_commit", targetCommit).Error; err != nil {
		return codeGitDeliveryResult{}, err
	}
	delivery.RemoteCommit = targetCommit
	if report != nil {
		report(codeDeliveryStageMerging, 55)
	}
	if _, err := runCodeGit(snapshot.SourceWorkDir, "merge-base", "--is-ancestor", delivery.WorktreeCommit, "HEAD"); err != nil {
		if _, err := runCodeGit(snapshot.SourceWorkDir, "merge", "--no-ff", "--no-edit", delivery.WorktreeCommit); err != nil {
			conflicts := codeGitConflictFiles(snapshot.SourceWorkDir)
			_, _ = runCodeGit(snapshot.SourceWorkDir, "merge", "--abort")
			if len(conflicts) > 0 {
				return codeGitDeliveryResult{Status: "conflict", Branch: snapshot.WorktreeBranch, ConflictFiles: conflicts}, nil
			}
			return codeGitDeliveryResult{}, err
		}
	}
	commit, err := runCodeGit(snapshot.SourceWorkDir, "rev-parse", "HEAD")
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
	snapshot := codeDeliverySessionSnapshot(delivery)
	if _, err := os.Stat(snapshot.WorkDir); err == nil {
		if err := cleanupCodeSessionWorktree(snapshot); err != nil {
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

func resumeCodeSessionDeliveryWithProgress(session *model.AIDevSession, userID uint, report codeDeliveryProgressReporter) (codeGitDeliveryResult, error) {
	delivery, err := loadOrCreateCodeDeliveryWithProgress(session, userID, report)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	result, err := integrateAndPushCodeDeliveryWithProgress(delivery, report)
	if err != nil || result.Status == "conflict" {
		result.Repositories = []codeRepositoryDeliveryResult{codeSingleRepositoryDeliveryResult(delivery, result)}
		return result, err
	}
	if delivery.Status == codeDeliveryMerged {
		if err := updateCodeDeliveryMetadata(delivery); err != nil {
			return codeGitDeliveryResult{}, err
		}
		delivery.Status = codeDeliveryCompleted
	}
	if delivery.Status == codeDeliveryWorktreeCleaned {
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
		PushStatus: pushStatus, PushedCommit: delivery.PushedCommit, ErrorMessage: delivery.PushError,
		ConflictFiles: result.ConflictFiles,
	}
}
