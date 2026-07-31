package api

import (
	"errors"
	"fmt"
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
	return repairCodeSingleWorktreeBranch(session)
}

func repairCodeSingleWorktreeBranch(session *model.AIDevSession) error {
	_, currentBranch, err := inspectCodexRepositoryWorktreeGitWritableDirs(session.SourceWorkDir, session.WorkDir)
	if err != nil {
		return err
	}
	branch, changed, err := reconcileCodeWorktreeBranch(session.ID, session.WorktreeBranch, currentBranch)
	if err != nil || !changed {
		return err
	}
	if branch == session.WorktreeBranch {
		return nil
	}
	result := global.DB.Model(&model.AIDevSession{}).
		Where("id = ? AND worktree_branch = ?", session.ID, session.WorktreeBranch).
		Update("worktree_branch", branch)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("会话 Worktree 分支记录已变化，请刷新后重试")
	}
	if global.LOG != nil {
		global.LOG.Warnf("Repaired Code worktree branch for session %d: %s -> %s", session.ID, session.WorktreeBranch, branch)
	}
	session.WorktreeBranch = branch
	return nil
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
		_, currentBranch, inspectErr := inspectCodexRepositoryWorktreeGitWritableDirs(repository.SourceDir, repository.WorktreeDir)
		if inspectErr != nil {
			return inspectErr
		}
		branch, changed, reconcileErr := reconcileCodeWorktreeBranch(session.ID, repository.Branch, currentBranch)
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
	for _, update := range updates {
		update.repository.Branch = update.newBranch
	}
	if err := writeCodeSessionManifest(session.WorkDir, repositories); err != nil {
		return fmt.Errorf("更新会话 Worktree 清单失败：%w", err)
	}
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			result := tx.Model(&model.AIDevSessionRepository{}).
				Where("id = ? AND session_id = ? AND branch = ?", update.repository.ID, session.ID, update.oldBranch).
				Update("branch", update.newBranch)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("会话仓库 Worktree 分支记录已变化，请刷新后重试")
			}
		}
		return nil
	}); err != nil {
		for _, update := range updates {
			update.repository.Branch = update.oldBranch
		}
		_ = writeCodeSessionManifest(session.WorkDir, repositories)
		return err
	}
	for _, update := range updates {
		if global.LOG != nil {
			global.LOG.Warnf("Repaired Code repository worktree branch for session %d: %s -> %s", session.ID, update.oldBranch, update.newBranch)
		}
	}
	return nil
}

func reconcileCodeWorktreeBranch(sessionID uint, recordedBranch, currentBranch string) (string, bool, error) {
	if currentBranch == recordedBranch {
		return recordedBranch, false, nil
	}
	if !isCodeSessionWorktreeBranch(currentBranch, sessionID) {
		return "", false, errCodeWorktreeBranchMismatch
	}
	return currentBranch, true, nil
}

func isCodeSessionWorktreeBranch(branch string, sessionID uint) bool {
	prefix := fmt.Sprintf("gopanel/code-%d-", sessionID)
	return strings.HasPrefix(strings.TrimSpace(branch), prefix) && len(strings.TrimSpace(branch)) > len(prefix)
}
