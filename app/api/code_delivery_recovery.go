package api

import (
	"errors"
	"os"
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
	status, err := runCodeGit(session.WorkDir, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(status) != "" {
		return nil, errors.New("Worktree 仍有未提交变更，请先提交")
	}
	targetBranch := session.TargetBranch
	if targetBranch == "" {
		targetBranch, _ = runCodeGit(session.SourceWorkDir, "branch", "--show-current")
	}
	targetCommit, err := refreshCodeRepositoryTarget(session.SourceWorkDir, targetBranch, session.RemoteName)
	if err != nil {
		return nil, err
	}
	if err := syncCodeWorktreeWithTarget(session.WorkDir, targetBranch); err != nil {
		return nil, err
	}
	if err := validateCodeQualityGate(session); err != nil {
		return nil, err
	}
	commit, err := runCodeGit(session.WorkDir, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	delivery = model.AICodeDelivery{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: userID,
		Status: codeDeliveryPrepared, SourceWorkDir: session.SourceWorkDir,
		WorkDir: session.WorkDir, WorktreeBranch: session.WorktreeBranch,
		TargetBranch: targetBranch, BaseCommit: session.BaseCommit,
		RemoteName: session.RemoteName, RemoteCommit: targetCommit,
		WorktreeCommit: commit,
	}
	if err := global.DB.Create(&delivery).Error; err != nil {
		if loadErr := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; loadErr == nil {
			return &delivery, nil
		}
		return nil, err
	}
	return &delivery, nil
}

func codeDeliverySessionSnapshot(delivery *model.AICodeDelivery) *model.AIDevSession {
	return &model.AIDevSession{
		ID: delivery.SessionID, UserID: delivery.UserID, ProjectID: delivery.ProjectID,
		WorkDir: delivery.WorkDir, SourceWorkDir: delivery.SourceWorkDir,
		WorktreeBranch: delivery.WorktreeBranch, TargetBranch: delivery.TargetBranch,
		BaseCommit: delivery.BaseCommit, RemoteName: delivery.RemoteName, RemoteCommit: delivery.RemoteCommit,
	}
}

func mergePreparedCodeDelivery(delivery *model.AICodeDelivery) (codeGitDeliveryResult, error) {
	snapshot := codeDeliverySessionSnapshot(delivery)
	targetBranch := snapshot.TargetBranch
	if targetBranch == "" {
		targetBranch, _ = runCodeGit(snapshot.SourceWorkDir, "branch", "--show-current")
	}
	targetCommit, err := refreshCodeRepositoryTarget(snapshot.SourceWorkDir, targetBranch, snapshot.RemoteName)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if _, err := runCodeGit(snapshot.WorkDir, "merge-base", "--is-ancestor", targetCommit, delivery.WorktreeCommit); err != nil {
		if err := syncCodeWorktreeWithTarget(snapshot.WorkDir, targetBranch); err != nil {
			return codeGitDeliveryResult{}, err
		}
		if err := validateCodeQualityGate(snapshot); err != nil {
			return codeGitDeliveryResult{}, err
		}
		worktreeCommit, err := runCodeGit(snapshot.WorkDir, "rev-parse", "HEAD")
		if err != nil {
			return codeGitDeliveryResult{}, err
		}
		if err := global.DB.Model(delivery).Updates(map[string]any{
			"worktree_commit": worktreeCommit, "remote_commit": targetCommit,
		}).Error; err != nil {
			return codeGitDeliveryResult{}, err
		}
		delivery.WorktreeCommit, delivery.RemoteCommit = worktreeCommit, targetCommit
	}
	if _, err := runCodeGit(snapshot.SourceWorkDir, "merge-base", "--is-ancestor", delivery.WorktreeCommit, "HEAD"); err != nil {
		if _, err := runCodeGit(snapshot.SourceWorkDir, "merge", "--no-ff", "--no-edit", snapshot.WorktreeBranch); err != nil {
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
		now := time.Now()
		return tx.Model(delivery).Updates(map[string]any{"status": codeDeliveryCompleted, "completed_at": now, "error_message": ""}).Error
	})
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
	return global.DB.Model(delivery).Updates(map[string]any{
		"status": codeDeliveryWorktreeCleaned, "error_message": "",
	}).Error
}

func resumeCodeSessionDelivery(session *model.AIDevSession, userID uint) (codeGitDeliveryResult, error) {
	delivery, err := loadOrCreateCodeDelivery(session, userID)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	result := codeGitDeliveryResult{Status: "merged", Commit: delivery.MergeCommit, Branch: delivery.WorktreeBranch}
	if delivery.Status == codeDeliveryPrepared {
		result, err = mergePreparedCodeDelivery(delivery)
		if err != nil || result.Status == "conflict" {
			return result, err
		}
	}
	if delivery.Status == codeDeliveryMerged {
		if err := cleanupCodeDeliveryWorktree(delivery); err != nil {
			return codeGitDeliveryResult{}, err
		}
		delivery.Status = codeDeliveryWorktreeCleaned
	}
	if delivery.Status == codeDeliveryWorktreeCleaned {
		if err := updateCodeDeliveryMetadata(delivery); err != nil {
			return codeGitDeliveryResult{}, err
		}
		delivery.Status = codeDeliveryCompleted
	}
	return result, nil
}
