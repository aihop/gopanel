package api

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
)

var errCodeWorktreeBranchMismatch = buserr.New(constant.ErrCodeWorktreeBranchMismatch)

func codexWritableDirsForSession(session *model.AIDevSession) ([]string, error) {
	if session == nil {
		return nil, nil
	}
	if session.IsolationMode == codeIsolationMultiWorktree {
		return resolveCodexMultiWorktreeGitWritableDirs(session)
	}
	if session.SourceWorkDir != "" || session.WorktreeBranch != "" {
		if session.SourceWorkDir == "" || session.WorktreeBranch == "" {
			return nil, buserr.New(constant.ErrCodeWorktreeMetadataIncomplete)
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
			return nil, buserr.New(constant.ErrCodeProjectSourceDirNotAbs)
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
			return nil, buserr.New(constant.ErrCodeProjectSourcePathNotDir)
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
		return nil, buserr.New(constant.ErrCodeWorktreeOutsideManaged)
	}
	if filepath.Clean(session.WorkDir) != filepath.Clean(aiSessionWorktreeDir(session.UserID, session.ID)) {
		return nil, buserr.New(constant.ErrCodeWorktreeDirIDMismatch)
	}
	workDir, err := resolveCodexGitDirectory(session.WorkDir)
	if err != nil {
		return nil, err
	}
	return resolveCodexRepositoryWorktreeGitWritableDirs(session.SourceWorkDir, workDir, session.WorktreeBranch)
}

func resolveCodexMultiWorktreeGitWritableDirs(session *model.AIDevSession) ([]string, error) {
	if !isManagedAISessionWorkDir(session.WorkDir, session.UserID) || !isAISessionWorkspaceDirectory(session.WorkDir) {
		return nil, buserr.New(constant.ErrCodeMultiWorktreeOutsideManaged)
	}
	if filepath.Clean(session.WorkDir) != filepath.Clean(aiSessionWorktreeDir(session.UserID, session.ID)) {
		return nil, buserr.New(constant.ErrCodeMultiWorktreeDirIDMismatch)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return nil, buserr.New(constant.ErrCodeMultiWorktreeMetadataUnavailable)
	}
	seen := make(map[string]struct{})
	writableDirs := make([]string, 0, len(repositories)*4)
	for _, repository := range repositories {
		if !isPathInside(filepath.Clean(repository.WorktreeDir), filepath.Clean(session.WorkDir)) {
			return nil, buserr.New(constant.ErrCodeRepoWorktreeOutsideManaged)
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
	writableDirs, currentBranch, err := inspectCodexRepositoryWorktreeGitWritableDirs(sourcePath, worktreePath)
	if err != nil {
		return nil, err
	}
	if currentBranch != branch {
		return nil, errCodeWorktreeBranchMismatch
	}
	return writableDirs, nil
}

func inspectCodexRepositoryWorktreeGitWritableDirs(sourcePath, worktreePath string) ([]string, string, error) {
	workDir, err := resolveCodexGitDirectory(worktreePath)
	if err != nil {
		return nil, "", err
	}
	sourceDir, err := resolveCodexGitDirectory(sourcePath)
	if err != nil {
		return nil, "", err
	}
	worktreeRoot, err := resolveCodeGitPath(workDir, "--show-toplevel")
	if err != nil || worktreeRoot != workDir {
		return nil, "", buserr.New(constant.ErrCodeSessionDirNotWorktreeRoot)
	}
	sourceRoot, err := resolveCodeGitPath(sourceDir, "--show-toplevel")
	if err != nil || sourceRoot != sourceDir {
		return nil, "", buserr.New(constant.ErrCodeSessionSourceDirNotRepoRoot)
	}
	worktreeGitDir, err := resolveCodeGitPath(workDir, "--git-dir")
	if err != nil {
		return nil, "", err
	}
	worktreeCommonDir, err := resolveCodeGitPath(workDir, "--git-common-dir")
	if err != nil {
		return nil, "", err
	}
	sourceCommonDir, err := resolveCodeGitPath(sourceDir, "--git-common-dir")
	if err != nil || sourceCommonDir != worktreeCommonDir {
		return nil, "", buserr.New(constant.ErrCodeWorktreeGitCommonDirMismatch)
	}
	if worktreeGitDir == worktreeCommonDir || !isPathInside(worktreeGitDir, filepath.Join(worktreeCommonDir, "worktrees")) {
		return nil, "", buserr.New(constant.ErrCodeWorktreeGitPrivateDirInvalid)
	}
	currentBranch := ""
	if headRef, headErr := runCodeGit(workDir, "symbolic-ref", "--quiet", "HEAD"); headErr == nil {
		currentBranch = strings.TrimPrefix(strings.TrimSpace(headRef), "refs/heads/")
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
			return nil, "", resolveErr
		}
		container := worktreeCommonDir
		if index == 0 {
			container = filepath.Join(worktreeCommonDir, "worktrees")
		}
		if !isPathInside(resolvedDir, container) {
			return nil, "", buserr.New(constant.ErrCodeGitWritableDirOutOfMetadata)
		}
		writableDirs[index] = resolvedDir
	}
	return writableDirs, currentBranch, nil
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
		return "", buserr.New(constant.ErrCodeGitMetadataPathNotAbs)
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
		return "", buserr.WithMap(constant.ErrCodeGitMetadataPathNotDir, map[string]interface{}{"path": resolvedPath})
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
		if arg == "exec" || arg == "resume" || arg == "app-server" {
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
