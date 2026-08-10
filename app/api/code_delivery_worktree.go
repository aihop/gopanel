package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func aiSessionDeliveryWorktreeDir(userID, sessionID uint) string {
	return filepath.Join(aiProjectWorktreeRoot(userID), fmt.Sprintf("delivery_%d", sessionID))
}

func isManagedAIDeliveryWorkDir(workDir string, userID, sessionID uint) bool {
	return filepath.Clean(strings.TrimSpace(workDir)) == aiSessionDeliveryWorktreeDir(userID, sessionID)
}

func ensureCodeDeliveryWorktree(delivery *model.AICodeDelivery, targetCommit string) error {
	if delivery == nil || strings.TrimSpace(targetCommit) == "" {
		return errors.New("交付 Worktree 基线不可用")
	}
	deliveryDir := strings.TrimSpace(delivery.DeliveryWorkDir)
	if deliveryDir == "" {
		deliveryDir = aiSessionDeliveryWorktreeDir(delivery.UserID, delivery.SessionID)
	}
	if !isManagedAIDeliveryWorkDir(deliveryDir, delivery.UserID, delivery.SessionID) {
		return errors.New("交付 Worktree 不在 GoPanel 管理目录中")
	}
	if _, err := os.Lstat(deliveryDir); err == nil {
		if _, err := runCodeGit(deliveryDir, "rev-parse", "--is-inside-work-tree"); err != nil {
			return errors.New("交付 Worktree 目录已存在但不是有效 Git 工作区")
		}
		return persistCodeDeliveryWorktreeDir(delivery, deliveryDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(deliveryDir), 0750); err != nil {
		return err
	}
	if _, err := runCodeGit(delivery.SourceWorkDir, "worktree", "add", "--detach", deliveryDir, targetCommit); err != nil {
		return err
	}
	if err := persistCodeDeliveryWorktreeDir(delivery, deliveryDir); err != nil {
		_, _ = runCodeGit(delivery.SourceWorkDir, "worktree", "remove", "--force", deliveryDir)
		return err
	}
	return nil
}

func persistCodeDeliveryWorktreeDir(delivery *model.AICodeDelivery, deliveryDir string) error {
	if delivery.DeliveryWorkDir == deliveryDir {
		return nil
	}
	if err := global.DB.Model(delivery).Update("delivery_work_dir", deliveryDir).Error; err != nil {
		return err
	}
	delivery.DeliveryWorkDir = deliveryDir
	return nil
}

func resetCodeDeliveryWorktree(delivery *model.AICodeDelivery, targetCommit string) error {
	if err := ensureCodeDeliveryWorktree(delivery, targetCommit); err != nil {
		return err
	}
	_, _ = runCodeGit(delivery.DeliveryWorkDir, "merge", "--abort")
	if _, err := runCodeGit(delivery.DeliveryWorkDir, "reset", "--hard", targetCommit); err != nil {
		return err
	}
	_, err := runCodeGit(delivery.DeliveryWorkDir, "clean", "-d", "-f")
	return err
}

func prepareCodeDeliveryMergeWorktree(delivery *model.AICodeDelivery, targetCommit string) error {
	if err := ensureCodeDeliveryWorktree(delivery, targetCommit); err != nil {
		return err
	}
	if conflicts := codeGitConflictFiles(delivery.DeliveryWorkDir); len(conflicts) > 0 {
		return nil
	}
	status, err := runCodeGit(delivery.DeliveryWorkDir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) != "" {
		return nil
	}
	if _, err := runCodeGit(delivery.DeliveryWorkDir, "merge-base", "--is-ancestor", delivery.WorktreeCommit, "HEAD"); err == nil {
		return nil
	}
	return resetCodeDeliveryWorktree(delivery, targetCommit)
}

func cleanupCodeIntegrationWorktree(delivery *model.AICodeDelivery) error {
	if delivery == nil || strings.TrimSpace(delivery.DeliveryWorkDir) == "" {
		return nil
	}
	if !isManagedAIDeliveryWorkDir(delivery.DeliveryWorkDir, delivery.UserID, delivery.SessionID) {
		return errors.New("交付 Worktree 不在 GoPanel 管理目录中")
	}
	if _, err := os.Lstat(delivery.DeliveryWorkDir); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if _, err := runCodeGit(delivery.SourceWorkDir, "worktree", "remove", delivery.DeliveryWorkDir); err != nil {
		return err
	}
	return nil
}

func fastForwardCodeDeliverySource(delivery *model.AICodeDelivery) error {
	status, err := runCodeGit(delivery.SourceWorkDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		return errors.New("本地主仓在交付期间出现未提交变更，已保留交付 Worktree")
	}
	branch, err := runCodeGit(delivery.SourceWorkDir, "branch", "--show-current")
	if err != nil {
		return err
	}
	branch = strings.TrimSpace(branch)
	if delivery.TargetBranch == "" {
		delivery.TargetBranch = branch
		if err := global.DB.Model(delivery).Update("target_branch", branch).Error; err != nil {
			return err
		}
	}
	if branch != delivery.TargetBranch {
		return fmt.Errorf("本地主仓当前分支不是交付目标 %s，已保留交付 Worktree", delivery.TargetBranch)
	}
	localCommit, err := runCodeGit(delivery.SourceWorkDir, "rev-parse", "refs/heads/"+delivery.TargetBranch)
	if err != nil {
		return err
	}
	if localCommit == delivery.MergeCommit {
		return nil
	}
	sourceCommit := strings.TrimSpace(delivery.SourceCommit)
	if sourceCommit == "" {
		sourceCommit = strings.TrimSpace(delivery.RemoteCommit)
	}
	if localCommit != sourceCommit {
		return errors.New("本地主仓在交付期间已推进，无法安全快进")
	}
	if _, err := runCodeGit(delivery.SourceWorkDir, "merge", "--ff-only", delivery.MergeCommit); err != nil {
		return err
	}
	updated, err := runCodeGit(delivery.SourceWorkDir, "rev-parse", "HEAD")
	if err != nil || updated != delivery.MergeCommit {
		return errors.New("本地主仓快进结果核验失败")
	}
	return nil
}

func markCodeDeliveryLocalPush(delivery *model.AICodeDelivery) error {
	now := time.Now()
	if err := global.DB.Model(delivery).Updates(map[string]any{
		"push_status": "local", "pushed_commit": delivery.MergeCommit, "pushed_at": now, "push_error": "",
	}).Error; err != nil {
		return err
	}
	delivery.PushStatus, delivery.PushedCommit, delivery.PushedAt = "local", delivery.MergeCommit, &now
	return nil
}
