package api

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

func codexWritableDirsForSessionWithRepair(session *model.AIDevSession) ([]string, error) {
	writableDirs, err := codexWritableDirsForSession(session)
	if !errors.Is(err, errCodeWorktreeBranchMismatch) {
		return writableDirs, err
	}
	if repairErr := repairCodeSessionWorktreeBranches(session); repairErr != nil {
		return nil, repairErr
	}
	return codexWritableDirsForSession(session)
}

func repairCodeSessionWorktreeBranches(session *model.AIDevSession) error {
	if session == nil || session.ID == 0 {
		return errCodeWorktreeBranchMismatch
	}
	if session.IsolationMode == codeIsolationMultiWorktree {
		return repairCodeMultiWorktreeBranches(session)
	}
	if session.SourceWorkDir == "" && session.WorktreeBranch == "" {
		return nil
	}
	if session.SourceWorkDir == "" || session.WorktreeBranch == "" {
		return errors.New("会话 Worktree 元数据不完整")
	}
	return repairCodeSingleWorktreeBranch(session)
}

func repairCodeSingleWorktreeBranch(session *model.AIDevSession) error {
	if skip, err := skipMissingDeliveredCodeWorktree(session.ID, session.WorkDir); err != nil || skip {
		return err
	}
	_, currentBranch, err := inspectCodexRepositoryWorktreeGitWritableDirs(session.SourceWorkDir, session.WorkDir)
	if err != nil {
		return err
	}
	branch, changed, err := reconcileCodeWorktreeBranch(session.ID, session.WorkDir, session.WorktreeBranch, currentBranch)
	if err != nil || !changed {
		return err
	}
	if branch == session.WorktreeBranch {
		return nil
	}
	oldBranch := session.WorktreeBranch
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.AIDevSession{}).
			Where("id = ? AND worktree_branch = ?", session.ID, oldBranch).
			Update("worktree_branch", branch)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			var stored model.AIDevSession
			if err := tx.First(&stored, session.ID).Error; err != nil {
				return err
			}
			if stored.WorktreeBranch != branch {
				return errors.New("会话 Worktree 分支记录已变化，请刷新后重试")
			}
		}
		return repairCodeDeliveryBranch(tx, session.ID, oldBranch, branch)
	}); err != nil {
		return err
	}
	if global.LOG != nil {
		global.LOG.Warnf("Repaired Code worktree branch for session %d: %s -> %s", session.ID, oldBranch, branch)
	}
	session.WorktreeBranch = branch
	return nil
}

func skipMissingDeliveredCodeWorktree(sessionID uint, workDir string) (bool, error) {
	if _, err := os.Stat(workDir); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	var delivery model.AICodeDelivery
	err := global.DB.Where("session_id = ?", sessionID).First(&delivery).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return delivery.Status == codeDeliveryMerged || delivery.Status == codeDeliveryWorktreeCleaned || delivery.Status == codeDeliveryCompleted, nil
}

func repairCodeDeliveryBranch(tx *gorm.DB, sessionID uint, oldBranch, newBranch string) error {
	var delivery model.AICodeDelivery
	err := tx.Where("session_id = ?", sessionID).First(&delivery).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if delivery.WorktreeBranch == newBranch {
		return nil
	}
	if delivery.WorktreeBranch != oldBranch {
		return errors.New("会话交付分支记录与 Worktree 不一致")
	}
	return tx.Model(&delivery).Update("worktree_branch", newBranch).Error
}

func repairCodeMultiWorktreeBranches(session *model.AIDevSession) error {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return errors.New("会话多仓库 Worktree 元数据不可用")
	}
	type branchUpdate struct {
		repository *model.AIDevSessionRepository
		oldBranch  string
		newBranch  string
	}
	updates := make([]branchUpdate, 0, len(repositories))
	for index := range repositories {
		repository := &repositories[index]
		if repository.Status == codeDeliveryCompleted {
			continue
		}
		if _, statErr := os.Stat(repository.WorktreeDir); errors.Is(statErr, os.ErrNotExist) && repository.Status == codeDeliveryMerged {
			continue
		} else if statErr != nil {
			return statErr
		}
		_, currentBranch, inspectErr := inspectCodexRepositoryWorktreeGitWritableDirs(repository.SourceDir, repository.WorktreeDir)
		if inspectErr != nil {
			return inspectErr
		}
		branch, changed, reconcileErr := reconcileCodeWorktreeBranch(session.ID, repository.WorktreeDir, repository.Branch, currentBranch)
		if reconcileErr != nil {
			return reconcileErr
		}
		if !changed || branch == repository.Branch {
			continue
		}
		updates = append(updates, branchUpdate{repository: repository, oldBranch: repository.Branch, newBranch: branch})
	}
	if len(updates) == 0 {
		return nil
	}
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			result := tx.Model(&model.AIDevSessionRepository{}).
				Where("id = ? AND session_id = ? AND branch = ?", update.repository.ID, session.ID, update.oldBranch).
				Update("branch", update.newBranch)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				var stored model.AIDevSessionRepository
				if err := tx.First(&stored, update.repository.ID).Error; err != nil {
					return err
				}
				if stored.SessionID != session.ID || stored.Branch != update.newBranch {
					return errors.New("会话仓库 Worktree 分支记录已变化，请刷新后重试")
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	repositories, err = loadCodeSessionRepositories(session.ID)
	if err != nil {
		return err
	}
	if err := writeCodeSessionManifest(session.WorkDir, repositories); err != nil {
		return fmt.Errorf("更新会话 Worktree 清单失败：%w", err)
	}
	for _, update := range updates {
		if global.LOG != nil {
			global.LOG.Warnf("Repaired Code repository worktree branch for session %d: %s -> %s", session.ID, update.oldBranch, update.newBranch)
		}
	}
	return nil
}

func reconcileCodeWorktreeBranch(sessionID uint, worktreeDir, recordedBranch, currentBranch string) (string, bool, error) {
	if currentBranch == recordedBranch {
		return recordedBranch, false, nil
	}
	if isCodeSessionWorktreeBranch(currentBranch, sessionID) {
		return currentBranch, true, nil
	}
	if !codeWorktreeBranchExists(worktreeDir, recordedBranch) || !codeWorktreeIsClean(worktreeDir) {
		return "", false, errCodeWorktreeBranchMismatch
	}
	if _, err := runCodeGit(worktreeDir, "switch", "--", recordedBranch); err != nil {
		return "", false, fmt.Errorf("恢复会话 Worktree 分支失败：%w", err)
	}
	return recordedBranch, true, nil
}

func codeWorktreeBranchExists(worktreeDir, branch string) bool {
	if strings.TrimSpace(branch) == "" {
		return false
	}
	_, err := runCodeGit(worktreeDir, "show-ref", "--verify", "refs/heads/"+branch)
	return err == nil
}

func codeWorktreeIsClean(worktreeDir string) bool {
	status, err := runCodeGit(worktreeDir, "status", "--porcelain")
	return err == nil && strings.TrimSpace(status) == ""
}

func isCodeSessionWorktreeBranch(branch string, sessionID uint) bool {
	prefix := fmt.Sprintf("gopanel/code-%d-", sessionID)
	return strings.HasPrefix(strings.TrimSpace(branch), prefix) && len(strings.TrimSpace(branch)) > len(prefix)
}
