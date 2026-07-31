package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

const (
	codeIsolationMultiWorktree = "multi_worktree"
	codeSessionManifestName    = ".gopanel-session.json"
)

type codeSessionRepositoryManifest struct {
	SourceDir   string `json:"sourceDir"`
	WorktreeDir string `json:"worktreeDir"`
	LinkName    string `json:"linkName"`
	Branch      string `json:"branch"`
}

type codeSessionWorkspaceManifest struct {
	Version      int                             `json:"version"`
	Repositories []codeSessionRepositoryManifest `json:"repositories"`
}

func isAISessionWorkspaceDirectory(workDir string) bool {
	info, err := os.Stat(filepath.Join(workDir, codeSessionManifestName))
	return err == nil && info.Mode().IsRegular()
}

func loadCodeSessionRepositories(sessionID uint) ([]model.AIDevSessionRepository, error) {
	var repositories []model.AIDevSessionRepository
	err := global.DB.Where("session_id = ?", sessionID).Order("link_name asc").Find(&repositories).Error
	return repositories, err
}

func sessionRepositoryLinkNames(sourceDirs []string) []aiProjectWorkspaceSource {
	return buildAIProjectWorkspaceSources(sourceDirs, aiProjectWorkspaceManifest{}, nil)
}

func writeCodeSessionManifest(workDir string, repositories []model.AIDevSessionRepository) error {
	manifest := codeSessionWorkspaceManifest{Version: 1, Repositories: make([]codeSessionRepositoryManifest, 0, len(repositories))}
	for _, repository := range repositories {
		manifest.Repositories = append(manifest.Repositories, codeSessionRepositoryManifest{
			SourceDir: repository.SourceDir, WorktreeDir: repository.WorktreeDir,
			LinkName: repository.LinkName, Branch: repository.Branch,
		})
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(workDir, codeSessionManifestName), content, 0600)
}

func createCodeSessionRepositoryWorktrees(session *model.AIDevSession, project *model.AIGroup, sourceDirs []string) error {
	workspaceDir := aiSessionWorktreeDir(session.UserID, session.ID)
	if _, err := os.Lstat(workspaceDir); !errors.Is(err, os.ErrNotExist) {
		return errors.New("会话 Worktree 目录已存在")
	}
	if err := os.MkdirAll(workspaceDir, 0750); err != nil {
		return fmt.Errorf("创建会话工作区失败：%w", err)
	}
	created := make([]model.AIDevSessionRepository, 0, len(sourceDirs))
	rollback := func() {
		for index := len(created) - 1; index >= 0; index-- {
			repository := created[index]
			_, _ = runCodeGit(repository.SourceDir, "worktree", "remove", "--force", repository.WorktreeDir)
			_, _ = runCodeGit(repository.SourceDir, "branch", "-D", "--", repository.Branch)
		}
		_ = os.RemoveAll(workspaceDir)
	}
	stamp := time.Now().Unix()
	for index, source := range sessionRepositoryLinkNames(sourceDirs) {
		branch := fmt.Sprintf("gopanel/code-%d-%d-%d", session.ID, stamp, index+1)
		worktreeDir := filepath.Join(workspaceDir, source.LinkName)
		baseCommit, err := runCodeGit(source.Path, "rev-parse", "HEAD")
		if err != nil {
			rollback()
			return err
		}
		if _, err := runCodeGit(source.Path, "worktree", "add", "-b", branch, worktreeDir, "HEAD"); err != nil {
			rollback()
			return err
		}
		created = append(created, model.AIDevSessionRepository{
			SessionID: session.ID, ProjectID: project.ID, SourceDir: source.Path,
			WorktreeDir: worktreeDir, LinkName: source.LinkName, Branch: branch,
			BaseCommit: baseCommit, Status: "active",
		})
	}
	if err := writeCodeSessionManifest(workspaceDir, created); err != nil {
		rollback()
		return err
	}
	if err := global.DB.Create(&created).Error; err != nil {
		rollback()
		return err
	}
	session.WorkDir = workspaceDir
	session.SourceWorkDir = ""
	session.WorktreeBranch = ""
	session.IsolationMode = codeIsolationMultiWorktree
	return nil
}

func codeSessionRepositoryID(repositoryID uint) string {
	return fmt.Sprintf("session-repository-%d", repositoryID)
}

func codeSessionRepositoryByCodeID(sessionID uint, codeID string) (*model.AIDevSessionRepository, error) {
	repositories, err := loadCodeSessionRepositories(sessionID)
	if err != nil {
		return nil, err
	}
	for index := range repositories {
		if codeSessionRepositoryID(repositories[index].ID) == codeID {
			return &repositories[index], nil
		}
	}
	return nil, errors.New("Git 仓库不存在或不属于当前会话")
}

func validateCodeSessionRepositoryWorktree(session *model.AIDevSession, repository *model.AIDevSessionRepository) error {
	if session == nil || repository == nil || session.IsolationMode != codeIsolationMultiWorktree {
		return errors.New("会话仓库 Worktree 元数据无效")
	}
	workspaceDir := filepath.Clean(aiSessionWorktreeDir(session.UserID, session.ID))
	worktreeDir := filepath.Clean(repository.WorktreeDir)
	if filepath.Clean(session.WorkDir) != workspaceDir || filepath.Dir(worktreeDir) != workspaceDir || filepath.Base(worktreeDir) != repository.LinkName {
		return errors.New("会话仓库 Worktree 目录与会话记录不一致")
	}
	info, err := os.Lstat(worktreeDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("会话仓库 Worktree 目录无效")
	}
	if _, err := resolveCodexRepositoryWorktreeGitWritableDirs(repository.SourceDir, worktreeDir, repository.Branch); err != nil {
		return err
	}
	return nil
}

func rollbackCodeSessionRepositoryWorktrees(session *model.AIDevSession) {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		global.LOG.Errorf("Load Code repository worktrees %d failed: %v", session.ID, err)
		return
	}
	for index := len(repositories) - 1; index >= 0; index-- {
		repository := repositories[index]
		if _, err := runCodeGit(repository.SourceDir, "worktree", "remove", "--force", repository.WorktreeDir); err != nil {
			global.LOG.Errorf("Rollback Code repository worktree %d failed: %v", repository.ID, err)
			continue
		}
		_, _ = runCodeGit(repository.SourceDir, "branch", "-D", "--", repository.Branch)
	}
	_ = global.DB.Where("session_id = ?", session.ID).Delete(&model.AIDevSessionRepository{}).Error
	_ = os.RemoveAll(session.WorkDir)
}

func cleanupCodeSessionRepositoryWorktrees(session *model.AIDevSession) error {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		return err
	}
	for _, repository := range repositories {
		status, statusErr := runCodeGit(repository.WorktreeDir, "status", "--porcelain")
		if statusErr != nil || strings.TrimSpace(status) != "" {
			return fmt.Errorf("仓库 %s 仍有未提交修改，已保留会话工作区", repository.LinkName)
		}
		if _, ancestorErr := runCodeGit(repository.SourceDir, "merge-base", "--is-ancestor", repository.Branch, "HEAD"); ancestorErr != nil {
			return fmt.Errorf("仓库 %s 包含尚未合并的提交，已保留会话工作区", repository.LinkName)
		}
	}
	sort.SliceStable(repositories, func(i, j int) bool { return repositories[i].LinkName > repositories[j].LinkName })
	for _, repository := range repositories {
		if _, err := runCodeGit(repository.SourceDir, "worktree", "remove", repository.WorktreeDir); err != nil {
			return err
		}
		if _, err := runCodeGit(repository.SourceDir, "branch", "-d", "--", repository.Branch); err != nil {
			return err
		}
	}
	if err := global.DB.Where("session_id = ?", session.ID).Delete(&model.AIDevSessionRepository{}).Error; err != nil {
		return err
	}
	return os.RemoveAll(session.WorkDir)
}
