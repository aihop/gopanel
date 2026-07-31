package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
)

func codexWritableDirsForSession(session *model.AIDevSession) ([]string, error) {
	if session == nil {
		return nil, nil
	}
	if session.IsolationMode == codeIsolationMultiWorktree {
		return resolveCodexMultiWorktreeGitWritableDirs(session)
	}
	if session.SourceWorkDir != "" || session.WorktreeBranch != "" {
		if session.SourceWorkDir == "" || session.WorktreeBranch == "" {
			return nil, errors.New("会话 Worktree 元数据不完整")
		}
		return resolveCodexWorktreeGitWritableDirs(session)
	}
	if session.ProjectID == 0 {
		return nil, nil
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(session.ProjectID)
	if err != nil {
		return nil, err
	}
	return resolveCodexWritableDirs(project.SourceDirs)
}

func resolveCodexWritableDirs(sourceDirs []string) ([]string, error) {
	seen := make(map[string]struct{}, len(sourceDirs))
	resolvedDirs := make([]string, 0, len(sourceDirs))
	for _, sourceDir := range sourceDirs {
		sourceDir = strings.TrimSpace(sourceDir)
		if sourceDir == "" || !filepath.IsAbs(sourceDir) {
			return nil, errors.New("项目源目录必须是有效的绝对目录")
		}
		resolvedDir, err := filepath.EvalSymlinks(filepath.Clean(sourceDir))
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(resolvedDir)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, errors.New("项目源路径不是目录")
		}
		if _, exists := seen[resolvedDir]; exists {
			continue
		}
		seen[resolvedDir] = struct{}{}
		resolvedDirs = append(resolvedDirs, resolvedDir)
	}
	return resolvedDirs, nil
}

func resolveCodexWorktreeGitWritableDirs(session *model.AIDevSession) ([]string, error) {
	if !isManagedAISessionWorkDir(session.WorkDir, session.UserID) {
		return nil, errors.New("会话 Worktree 不在 GoPanel 管理目录中")
	}
	if filepath.Clean(session.WorkDir) != filepath.Clean(aiSessionWorktreeDir(session.UserID, session.ID)) {
		return nil, errors.New("会话 Worktree 目录与会话编号不一致")
	}
	workDir, err := resolveCodexGitDirectory(session.WorkDir)
	if err != nil {
		return nil, err
	}
	return resolveCodexRepositoryWorktreeGitWritableDirs(session.SourceWorkDir, workDir, session.WorktreeBranch)
}

func resolveCodexMultiWorktreeGitWritableDirs(session *model.AIDevSession) ([]string, error) {
	if !isManagedAISessionWorkDir(session.WorkDir, session.UserID) || !isAISessionWorkspaceDirectory(session.WorkDir) {
		return nil, errors.New("会话多仓库 Worktree 不在 GoPanel 管理目录中")
	}
	if filepath.Clean(session.WorkDir) != filepath.Clean(aiSessionWorktreeDir(session.UserID, session.ID)) {
		return nil, errors.New("会话多仓库 Worktree 目录与会话编号不一致")
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return nil, errors.New("会话多仓库 Worktree 元数据不可用")
	}
	seen := make(map[string]struct{})
	writableDirs := make([]string, 0, len(repositories)*4)
	for _, repository := range repositories {
		if !isPathInside(filepath.Clean(repository.WorktreeDir), filepath.Clean(session.WorkDir)) {
			return nil, errors.New("会话仓库 Worktree 超出管理目录")
		}
		resolved, resolveErr := resolveCodexRepositoryWorktreeGitWritableDirs(repository.SourceDir, repository.WorktreeDir, repository.Branch)
		if resolveErr != nil {
			return nil, resolveErr
		}
		for _, writableDir := range resolved {
			if _, exists := seen[writableDir]; exists {
				continue
			}
			seen[writableDir] = struct{}{}
			writableDirs = append(writableDirs, writableDir)
		}
	}
	return writableDirs, nil
}

func resolveCodexRepositoryWorktreeGitWritableDirs(sourcePath, worktreePath, branch string) ([]string, error) {
	workDir, err := resolveCodexGitDirectory(worktreePath)
	if err != nil {
		return nil, err
	}
	sourceDir, err := resolveCodexGitDirectory(sourcePath)
	if err != nil {
		return nil, err
	}
	worktreeRoot, err := resolveCodeGitPath(workDir, "--show-toplevel")
	if err != nil || worktreeRoot != workDir {
		return nil, errors.New("会话目录不是预期的 Git Worktree 根目录")
	}
	sourceRoot, err := resolveCodeGitPath(sourceDir, "--show-toplevel")
	if err != nil || sourceRoot != sourceDir {
		return nil, errors.New("会话源目录不是预期的 Git 仓库根目录")
	}
	worktreeGitDir, err := resolveCodeGitPath(workDir, "--git-dir")
	if err != nil {
		return nil, err
	}
	worktreeCommonDir, err := resolveCodeGitPath(workDir, "--git-common-dir")
	if err != nil {
		return nil, err
	}
	sourceCommonDir, err := resolveCodeGitPath(sourceDir, "--git-common-dir")
	if err != nil || sourceCommonDir != worktreeCommonDir {
		return nil, errors.New("会话 Worktree 与源仓库的 Git 公共目录不一致")
	}
	if worktreeGitDir == worktreeCommonDir || !isPathInside(worktreeGitDir, filepath.Join(worktreeCommonDir, "worktrees")) {
		return nil, errors.New("会话 Worktree 的 Git 私有目录无效")
	}
	headRef, err := runCodeGit(workDir, "symbolic-ref", "--quiet", "HEAD")
	if err != nil || strings.TrimSpace(headRef) != "refs/heads/"+branch {
		return nil, errors.New("会话 Worktree 当前分支与会话记录不一致")
	}
	writableDirs := []string{
		worktreeGitDir,
		filepath.Join(worktreeCommonDir, "objects"),
		filepath.Join(worktreeCommonDir, "refs"),
		filepath.Join(worktreeCommonDir, "logs"),
	}
	for index, writableDir := range writableDirs {
		resolvedDir, resolveErr := resolveCodexGitDirectory(writableDir)
		if resolveErr != nil {
			return nil, resolveErr
		}
		container := worktreeCommonDir
		if index == 0 {
			container = filepath.Join(worktreeCommonDir, "worktrees")
		}
		if !isPathInside(resolvedDir, container) {
			return nil, errors.New("Git 可写目录超出已验证的元数据范围")
		}
		writableDirs[index] = resolvedDir
	}
	return writableDirs, nil
}

func resolveCodeGitPath(workDir, pathName string) (string, error) {
	value, err := runCodeGit(workDir, "rev-parse", "--path-format=absolute", pathName)
	if err != nil {
		return "", err
	}
	return resolveCodexGitDirectory(value)
}

func resolveCodexGitDirectory(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "", errors.New("Git 元数据路径必须是有效的绝对目录")
	}
	resolvedPath, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("Git 元数据路径不是目录：%s", resolvedPath)
	}
	return resolvedPath, nil
}

func isPathInside(path, parent string) bool {
	relativePath, err := filepath.Rel(parent, path)
	return err == nil && relativePath != "." && relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))
}

func addCodexWritableDirArgs(args, writableDirs []string) []string {
	if len(writableDirs) == 0 {
		return args
	}
	insertionIndex := len(args)
	for index, arg := range args {
		if arg == "exec" || arg == "resume" {
			insertionIndex = index
			break
		}
	}
	extraArgs := make([]string, 0, len(writableDirs)*2)
	for _, writableDir := range writableDirs {
		extraArgs = append(extraArgs, "--add-dir", writableDir)
	}
	result := make([]string, 0, len(args)+len(extraArgs))
	result = append(result, args[:insertionIndex]...)
	result = append(result, extraArgs...)
	result = append(result, args[insertionIndex:]...)
	return result
}
