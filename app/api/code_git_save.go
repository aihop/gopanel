package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

const defaultCodeGitSaveMessage = "chore: 保存会话修改"

func codeGitSaveMessage(message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return defaultCodeGitSaveMessage, nil
	}
	return validateCodeGitCommitMessage(message)
}

func saveCodeGitRepository(workDir, message string) (string, bool, error) {
	status, err := runCodeGit(workDir, "status", "--porcelain")
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(status) == "" {
		commit, headErr := runCodeGit(workDir, "rev-parse", "HEAD")
		return commit, false, headErr
	}
	if err := validateCodeGitSaveFiles(workDir); err != nil {
		return "", false, err
	}
	if _, err := runCodeGit(workDir, "add", "-A", "--", "."); err != nil {
		return "", false, err
	}
	if err := validateCodeGitStagedChanges(workDir); err != nil {
		return "", false, err
	}
	if _, err := runCodeGit(
		workDir,
		codeGitAuthoredArgs(
			"-c", "commit.gpgsign=false", "commit", "-m", message,
		)...,
	); err != nil {
		return "", false, err
	}
	commit, err := runCodeGit(workDir, "rev-parse", "HEAD")
	return commit, true, err
}

func saveCodeSessionWorktree(session *model.AIDevSession, message string) (codeGitDeliveryResult, error) {
	if err := validateCodeWorktreeDeliverySession(session); err != nil {
		return codeGitDeliveryResult{}, err
	}
	message, err := codeGitSaveMessage(message)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	commit, changed, err := saveCodeGitRepository(session.WorkDir, message)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if !changed {
		return codeGitDeliveryResult{}, errors.New("当前没有需要保存的修改")
	}
	return codeGitDeliveryResult{Status: "committed", Commit: commit, Branch: session.WorktreeBranch}, nil
}

func saveCodeSessionRepositories(session *model.AIDevSession, message string) (codeGitDeliveryResult, error) {
	message, err := codeGitSaveMessage(message)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	repositories, err := loadCodeDeliverySessionRepositories(session)
	if err != nil || len(repositories) == 0 {
		return codeGitDeliveryResult{}, errors.New("会话多仓库 Worktree 元数据不可用")
	}
	if _, err := codeMultiRepositoryWorkspaceDir(session, repositories); err != nil {
		return codeGitDeliveryResult{}, err
	}
	repositories, err = codeDeliveryRepositoriesInOrder(repositories, false)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}

	changedCount := 0
	results := make([]codeRepositoryDeliveryResult, 0, len(repositories))
	for index := range repositories {
		repository := &repositories[index]
		if err := validateCodeSessionRepositoryWorktree(session, repository); err != nil {
			return codeGitDeliveryResult{}, err
		}
		if err := syncSavedCodeRepositoryGitlinks(repository, repositories); err != nil {
			return codeGitDeliveryResult{}, err
		}
		commit, changed, err := saveCodeGitRepository(repository.WorktreeDir, message)
		if err != nil {
			return codeGitDeliveryResult{}, err
		}
		if changed {
			changedCount++
		}
		if err := global.DB.Model(repository).Updates(map[string]any{
			"status": "committed", "worktree_commit": commit, "error_message": "",
		}).Error; err != nil {
			return codeGitDeliveryResult{}, err
		}
		repository.Status, repository.WorktreeCommit, repository.ErrorMessage = "committed", commit, ""
		results = append(results, codeRepositoryDeliveryResult{
			RepositoryID: codeSessionRepositoryID(repository.ID), RepositoryName: repository.LinkName,
			Status: "committed", Branch: repository.Branch, TargetBranch: repository.TargetBranch,
			Commit: commit, PushStatus: repository.PushStatus,
		})
	}
	if changedCount == 0 {
		return codeGitDeliveryResult{}, errors.New("当前没有需要保存的修改")
	}
	return codeGitDeliveryResult{Status: "committed", Repositories: results}, nil
}

func saveCodeDirectRepositories(
	session *model.AIDevSession, project *model.AIProject, message string,
) (codeGitDeliveryResult, error) {
	if session == nil || project == nil || session.IsolationMode != codeIsolationDirect {
		return codeGitDeliveryResult{}, errors.New("当前会话不是直连项目目录模式")
	}
	message, err := codeGitSaveMessage(message)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	repositories := discoverCodeGitRepositories(session, project.SourceDirs, project.ExcludedRepositories)
	if len(project.SourceDirs) == 0 && strings.TrimSpace(project.WorkDir) != "" {
		repositories = discoverCodeGitRepositories(session, []string{project.WorkDir}, project.ExcludedRepositories)
	}
	if len(repositories) == 0 {
		return codeGitDeliveryResult{}, errors.New("项目目录内未发现可提交的 Git 仓库")
	}
	sort.SliceStable(repositories, func(i, j int) bool {
		return len(filepath.Clean(repositories[i].root)) > len(filepath.Clean(repositories[j].root))
	})
	changedCount := 0
	results := make([]codeRepositoryDeliveryResult, 0, len(repositories))
	for _, repository := range repositories {
		commit, changed, saveErr := saveCodeGitRepository(repository.root, message)
		if saveErr != nil {
			return codeGitDeliveryResult{}, fmt.Errorf("保存仓库 %s 失败：%w", repository.Name, saveErr)
		}
		if changed {
			changedCount++
		}
		results = append(results, codeRepositoryDeliveryResult{
			RepositoryID: repository.ID, RepositoryName: repository.Name,
			Status: "committed", Branch: repository.Branch, Commit: commit,
		})
	}
	if changedCount == 0 {
		return codeGitDeliveryResult{}, errors.New("当前没有需要保存的修改")
	}
	result := codeGitDeliveryResult{Status: "committed", Repositories: results}
	if len(results) == 1 {
		result.RepositoryID, result.RepositoryName = results[0].RepositoryID, results[0].RepositoryName
		result.Commit, result.Branch = results[0].Commit, results[0].Branch
	}
	return result, nil
}

func syncSavedCodeRepositoryGitlinks(parent *model.AIDevSessionRepository, repositories []model.AIDevSessionRepository) error {
	for index := range repositories {
		child := &repositories[index]
		if child.ParentSourceDir != parent.SourceDir || strings.TrimSpace(child.GitlinkPath) == "" {
			continue
		}
		commit, err := runCodeGit(child.WorktreeDir, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		if err := updateCodeGitlink(parent.WorktreeDir, child.GitlinkPath, commit); err != nil {
			return err
		}
	}
	return nil
}

func SaveCodeGitChanges(c fiber.Ctx) error {
	var req struct {
		Message string `json:"message"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return runCodeGitDelivery(c, "git_save_all", func(session *model.AIDevSession, project *model.AIProject) (codeGitDeliveryResult, error) {
		if err := validateCodeSessionDeliveryConflictIdle(session.ID); err != nil {
			return codeGitDeliveryResult{}, err
		}
		if session.IsolationMode == codeIsolationMultiWorktree {
			return saveCodeSessionRepositories(session, req.Message)
		}
		if session.IsolationMode == codeIsolationDirect {
			return saveCodeDirectRepositories(session, project, req.Message)
		}
		return saveCodeSessionWorktree(session, req.Message)
	})
}

func validateCodeSessionDeliveryConflictIdle(sessionID uint) error {
	var conflictJobs int64
	if err := global.DB.Model(&model.AICodeDeliveryJob{}).Where(
		"session_id = ? AND status = ?", sessionID, codeDeliveryJobConflict,
	).Count(&conflictJobs).Error; err != nil {
		return err
	}
	if conflictJobs > 0 {
		return errors.New("当前交付存在合并冲突，请使用“在网页处理”，或在终端合并后点击“核验手动合并”")
	}
	return nil
}
