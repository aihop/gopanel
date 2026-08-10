package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

func captureCodeDeliverySnapshot(session *model.AIDevSession, userID uint) error {
	if session.IsolationMode == codeIsolationMultiWorktree || hasCodeMultiRepositoryDelivery(session.ID) {
		return captureCodeMultiRepositoryDeliverySnapshot(session)
	}
	if err := validateCodeWorktreeDeliverySession(session); err != nil {
		return err
	}
	commit, err := cleanCodeDeliveryCommit(session.WorkDir, "Worktree")
	if err != nil {
		return err
	}
	targetBranch := strings.TrimSpace(session.TargetBranch)
	if targetBranch == "" {
		targetBranch, _ = runCodeGit(session.SourceWorkDir, "branch", "--show-current")
	}
	if strings.TrimSpace(targetBranch) == "" {
		return errors.New("交付目标分支不可用")
	}
	if err := verifyCodeDeliveryCommit(session.WorkDir, commit, "Worktree"); err != nil {
		return err
	}
	stats, _ := loadCodeTaskDiffStats(
		session.SourceWorkDir, session.BaseCommit, commit, map[string]codeTaskDiffStats{},
	)
	values := map[string]any{
		"project_id": session.ProjectID, "user_id": userID, "status": codeDeliveryPrepared,
		"source_work_dir": session.SourceWorkDir, "work_dir": session.WorkDir,
		"worktree_branch": session.WorktreeBranch, "target_branch": targetBranch,
		"base_commit": session.BaseCommit, "remote_name": session.RemoteName,
		"remote_branch": session.RemoteBranch, "remote_commit": session.RemoteCommit,
		"source_commit": "", "worktree_commit": commit,
		"merge_commit": "", "error_message": "", "merged_at": nil, "completed_at": nil,
		"push_status": codePushPending, "pushed_commit": "", "push_error": "", "pushed_at": nil,
		"stat_additions": stats.Additions, "stat_deletions": stats.Deletions, "stat_files": stats.Files,
	}
	var delivery model.AICodeDelivery
	err = global.DB.Where("session_id = ?", session.ID).First(&delivery).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		delivery = model.AICodeDelivery{
			SessionID: session.ID, ProjectID: session.ProjectID, UserID: userID,
			Status: codeDeliveryPrepared, SourceWorkDir: session.SourceWorkDir,
			WorkDir: session.WorkDir, WorktreeBranch: session.WorktreeBranch,
			TargetBranch: targetBranch, BaseCommit: session.BaseCommit,
			RemoteName: session.RemoteName, RemoteBranch: session.RemoteBranch,
			RemoteCommit: session.RemoteCommit, WorktreeCommit: commit, PushStatus: codePushPending,
			StatAdditions: stats.Additions, StatDeletions: stats.Deletions, StatFiles: stats.Files,
		}
		return global.DB.Create(&delivery).Error
	}
	if err != nil {
		return err
	}
	return global.DB.Model(&delivery).Updates(values).Error
}

func commitCodeSourceGitlinkUpdates(repository *model.AIDevSessionRepository, repositories []model.AIDevSessionRepository) error {
	expected := make(map[string]struct{})
	for _, child := range repositories {
		if child.ParentSourceDir != repository.SourceDir || child.GitlinkPath == "" || child.MergeCommit == "" {
			continue
		}
		path := strings.TrimSpace(child.GitlinkPath)
		if err := updateCodeGitlink(repository.SourceDir, path, child.MergeCommit); err != nil {
			return fmt.Errorf("同步父仓库 %s 的子仓库指针失败：%w", repository.LinkName, err)
		}
		expected[path] = struct{}{}
	}
	if len(expected) == 0 {
		return nil
	}
	staged, err := runCodeGit(repository.SourceDir, "diff", "--cached", "--name-only")
	if err != nil || strings.TrimSpace(staged) == "" {
		return err
	}
	for _, path := range strings.Split(staged, "\n") {
		if _, allowed := expected[strings.TrimSpace(path)]; !allowed {
			return fmt.Errorf("仓库 %s 包含非预期的暂存修改：%s", repository.LinkName, path)
		}
	}
	_, err = runCodeGit(
		repository.SourceDir,
		"-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.local",
		"-c", "commit.gpgsign=false", "commit", "-m", "chore: update nested repository pointers",
	)
	return err
}

func captureCodeMultiRepositoryDeliverySnapshot(session *model.AIDevSession) error {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return errors.New("会话多仓库 Worktree 元数据不可用")
	}
	if _, err := codeMultiRepositoryWorkspaceDir(session, repositories); err != nil {
		return err
	}
	if _, err := codeDeliveryRepositoriesInOrder(repositories, true); err != nil {
		return err
	}
	commits := make([]string, len(repositories))
	reset := make([]bool, len(repositories))
	repositoryBySource := make(map[string]int, len(repositories))
	for index := range repositories {
		repository := &repositories[index]
		repositoryBySource[filepath.Clean(repository.SourceDir)] = index
		commit, err := cleanCodeDeliveryCommit(repository.WorktreeDir, "仓库 "+repository.LinkName)
		if err != nil {
			return err
		}
		commits[index] = commit
		targetBranch := strings.TrimSpace(repository.TargetBranch)
		if targetBranch == "" {
			if commit != strings.TrimSpace(repository.BaseCommit) {
				return fmt.Errorf("仓库 %s 已产生新提交，但源仓库处于 detached HEAD，请先配置交付分支", repository.LinkName)
			}
		}
		previousCommit := strings.TrimSpace(repository.WorktreeCommit)
		reset[index] = previousCommit == "" || commit != previousCommit ||
			(repository.Status != codeDeliveryCompleted && repository.Status != codeDeliveryMerged &&
				repository.Status != codeDeliveryPrepared)
	}
	for changed := true; changed; {
		changed = false
		for index := range repositories {
			if !reset[index] || strings.TrimSpace(repositories[index].ParentSourceDir) == "" {
				continue
			}
			parentIndex, exists := repositoryBySource[filepath.Clean(repositories[index].ParentSourceDir)]
			if exists && !reset[parentIndex] {
				reset[parentIndex] = true
				changed = true
			}
		}
	}
	for index := range repositories {
		if !reset[index] {
			continue
		}
		repository := &repositories[index]
		targetBranch := strings.TrimSpace(repository.TargetBranch)
		if targetBranch == "" {
			if codeRepositoryHasResetDescendant(index, repositories, reset, repositoryBySource) {
				repository.Status = codeDeliveryPrepared
			} else {
				repository.Status = codeDeliveryCompleted
			}
		} else {
			repository.Status = codeDeliveryPrepared
		}
		repository.TargetBranch, repository.WorktreeCommit = targetBranch, commits[index]
	}
	for index := range repositories {
		if err := verifyCodeDeliveryCommit(repositories[index].WorktreeDir, repositories[index].WorktreeCommit, "仓库 "+repositories[index].LinkName); err != nil {
			return err
		}
	}
	// 交付快照确定后固化各仓库的产出行数，之后任务列表直接读库。
	diffCache := map[string]codeTaskDiffStats{}
	return global.DB.Transaction(func(tx *gorm.DB) error {
		for index := range repositories {
			if !reset[index] {
				if repositories[index].Status == codeDeliveryCompleted {
					if err := tx.Model(&repositories[index]).Updates(map[string]any{
						"publish_started_at": nil, "source_applied_at": nil,
					}).Error; err != nil {
						return err
					}
				}
				continue
			}
			repository := &repositories[index]
			stats, _ := loadCodeTaskDiffStats(
				repository.SourceDir, repository.BaseCommit, repository.WorktreeCommit, diffCache,
			)
			repository.StatAdditions, repository.StatDeletions = stats.Additions, stats.Deletions
			repository.StatFiles = stats.Files
			updates := map[string]any{
				"status": repository.Status, "target_branch": repository.TargetBranch,
				"worktree_commit": repository.WorktreeCommit, "source_commit": "",
				"merge_commit": "", "error_message": "", "merged_at": nil, "publish_started_at": nil,
				"source_applied_at": nil, "completed_at": nil,
				"push_status": codePushPending, "pushed_commit": "", "push_error": "", "pushed_at": nil,
				"stat_additions": stats.Additions, "stat_deletions": stats.Deletions, "stat_files": stats.Files,
			}
			if repository.Status != codeDeliveryCompleted {
				updates["status"] = codeDeliveryPrepared
			} else {
				updates["completed_at"] = time.Now()
			}
			if err := tx.Model(repository).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func codeRepositoryHasResetDescendant(
	index int,
	repositories []model.AIDevSessionRepository,
	reset []bool,
	repositoryBySource map[string]int,
) bool {
	for childIndex := range repositories {
		if childIndex == index || !reset[childIndex] {
			continue
		}
		parentSource := strings.TrimSpace(repositories[childIndex].ParentSourceDir)
		for parentSource != "" {
			parentIndex, exists := repositoryBySource[filepath.Clean(parentSource)]
			if !exists {
				break
			}
			if parentIndex == index {
				return true
			}
			parentSource = strings.TrimSpace(repositories[parentIndex].ParentSourceDir)
		}
	}
	return false
}

func cleanCodeDeliveryCommit(workDir, label string) (string, error) {
	status, err := runCodeGit(workDir, "status", "--porcelain")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(status) != "" {
		return "", fmt.Errorf("%s 仍有未提交变更，请先提交", label)
	}
	commit, err := runCodeGit(workDir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(commit), nil
}

func verifyCodeDeliveryCommit(workDir, expected, label string) error {
	commit, err := cleanCodeDeliveryCommit(workDir, label)
	if err != nil {
		return err
	}
	if commit != expected {
		return fmt.Errorf("%s 在交付快照创建期间发生变化，请重新交付", label)
	}
	return nil
}
