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
	values := map[string]any{
		"project_id": session.ProjectID, "user_id": userID, "status": codeDeliveryPrepared,
		"source_work_dir": session.SourceWorkDir, "work_dir": session.WorkDir,
		"worktree_branch": session.WorktreeBranch, "target_branch": targetBranch,
		"base_commit": session.BaseCommit, "remote_name": session.RemoteName,
		"remote_branch": session.RemoteBranch, "remote_commit": "", "worktree_commit": commit,
		"merge_commit": "", "error_message": "", "merged_at": nil, "completed_at": nil,
		"push_status": codePushPending, "pushed_commit": "", "push_error": "", "pushed_at": nil,
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
			WorktreeCommit: commit, PushStatus: codePushPending,
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
	for index := range repositories {
		repository := &repositories[index]
		commit, err := cleanCodeDeliveryCommit(repository.WorktreeDir, "仓库 "+repository.LinkName)
		if err != nil {
			return err
		}
		targetBranch := strings.TrimSpace(repository.TargetBranch)
		if targetBranch == "" {
			if commit != strings.TrimSpace(repository.BaseCommit) {
				return fmt.Errorf("仓库 %s 已产生新提交，但源仓库处于 detached HEAD，请先配置交付分支", repository.LinkName)
			}
			repository.Status = codeDeliveryCompleted
		} else {
			repository.Status = codeDeliveryPrepared
		}
		repository.TargetBranch, repository.WorktreeCommit = targetBranch, commit
	}
	for index := range repositories {
		if err := verifyCodeDeliveryCommit(repositories[index].WorktreeDir, repositories[index].WorktreeCommit, "仓库 "+repositories[index].LinkName); err != nil {
			return err
		}
	}
	return global.DB.Transaction(func(tx *gorm.DB) error {
		for index := range repositories {
			repository := &repositories[index]
			updates := map[string]any{
				"status": repository.Status, "target_branch": repository.TargetBranch,
				"remote_commit": "", "worktree_commit": repository.WorktreeCommit,
				"merge_commit": "", "error_message": "", "merged_at": nil, "completed_at": nil,
				"push_status": codePushPending, "pushed_commit": "", "push_error": "", "pushed_at": nil,
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
