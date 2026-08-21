package api

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

const (
	codeGitStatusFileLimit = 500
	codeGitDiffOutputLimit = 1024 * 1024
)

type codeGitFile struct {
	Path           string `json:"path"`
	OldPath        string `json:"oldPath,omitempty"`
	WorkspacePath  string `json:"workspacePath"`
	ResultStatus   string `json:"resultStatus,omitempty"`
	IndexStatus    string `json:"indexStatus"`
	WorktreeStatus string `json:"worktreeStatus"`
	Staged         bool   `json:"staged"`
	Changed        bool   `json:"changed"`
	Untracked      bool   `json:"untracked"`
}

type codeGitRepository struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Branch            string        `json:"branch"`
	Files             []codeGitFile `json:"files"`
	StagedCount       int           `json:"stagedCount"`
	ChangedCount      int           `json:"changedCount"`
	UntrackedCount    int           `json:"untrackedCount"`
	Additions         int           `json:"additions"`
	Deletions         int           `json:"deletions"`
	StagedAdditions   int           `json:"stagedAdditions"`
	StagedDeletions   int           `json:"stagedDeletions"`
	Truncated         bool          `json:"truncated"`
	Isolated          bool          `json:"isolated"`
	DeliveryStatus    string        `json:"deliveryStatus,omitempty"`
	SavedCommits      int           `json:"savedCommits,omitempty"`
	HeadCommit        string        `json:"headCommit,omitempty"`
	HeadCommitMessage string        `json:"headCommitMessage,omitempty"`
	BaseCommit        string        `json:"baseCommit,omitempty"`
	ResultCommit      string        `json:"resultCommit,omitempty"`
	ReviewState       string        `json:"reviewState,omitempty"`
	root              string
	workspacePrefix   string
	resultLive        bool
	targetBranch      string
}

type codeGitStatus struct {
	Available       bool                `json:"available"`
	Reason          string              `json:"reason,omitempty"`
	Repositories    []codeGitRepository `json:"repositories"`
	Files           int                 `json:"files"`
	Staged          int                 `json:"staged"`
	Changed         int                 `json:"changed"`
	Untracked       int                 `json:"untracked"`
	Additions       int                 `json:"additions"`
	Deletions       int                 `json:"deletions"`
	StagedAdditions int                 `json:"stagedAdditions"`
	StagedDeletions int                 `json:"stagedDeletions"`
	Scope           string              `json:"scope"`
	ReviewReady     bool                `json:"reviewReady"`
	ReviewRevision  string              `json:"reviewRevision,omitempty"`
}

func codeGitWorkspacePrefixes(session *model.AIDevSession) map[string]string {
	prefixes := make(map[string]string)
	if session == nil || !isAIProjectWorkspaceDirectory(session.WorkDir) {
		return prefixes
	}
	manifest, err := readAIProjectWorkspaceManifest(session.WorkDir)
	if err != nil {
		return prefixes
	}
	for _, source := range manifest.Sources {
		prefixes[filepath.Clean(source.Path)] = source.LinkName
	}
	return prefixes
}

func inspectCodeGitRepository(id, name, root, workspacePrefix string) (codeGitRepository, bool) {
	resolvedRoot, err := filepath.EvalSymlinks(filepath.Clean(root))
	if err != nil {
		return codeGitRepository{}, false
	}
	gitRoot, err := runCodeGit(resolvedRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return codeGitRepository{}, false
	}
	resolvedGitRoot, err := filepath.EvalSymlinks(filepath.Clean(gitRoot))
	if err != nil || resolvedGitRoot != resolvedRoot {
		return codeGitRepository{}, false
	}
	branch, _ := runCodeGit(resolvedRoot, "branch", "--show-current")
	return codeGitRepository{ID: id, Name: name, Branch: branch, root: resolvedRoot, workspacePrefix: workspacePrefix}, true
}

func loadCodeGitSavedState(repository *codeGitRepository, baseCommit string) {
	baseCommit = strings.TrimSpace(baseCommit)
	if repository == nil || baseCommit == "" {
		return
	}
	count, err := runCodeGit(repository.root, "rev-list", "--count", baseCommit+"..HEAD")
	if err != nil {
		return
	}
	repository.SavedCommits, err = strconv.Atoi(strings.TrimSpace(count))
	if err != nil || repository.SavedCommits == 0 {
		repository.SavedCommits = 0
		return
	}
	repository.HeadCommit, _ = runCodeGit(repository.root, "rev-parse", "--short=8", "HEAD")
	repository.HeadCommitMessage, _ = runCodeGit(repository.root, "log", "-1", "--pretty=format:%s", "HEAD")
}

func discoverCodeGitRepositories(
	session *model.AIDevSession,
	sourceDirs []string,
	excludedRepositories []string,
) []codeGitRepository {
	if session == nil {
		return nil
	}
	excludedRepositories = normalizeCodeExcludedRepositories(excludedRepositories)
	if session.IsolationMode == codeIsolationMultiWorktree {
		sessionRepositories, err := loadCodeSessionRepositories(session.ID)
		if err != nil {
			return nil
		}
		repositories := make([]codeGitRepository, 0, len(sessionRepositories))
		for _, sessionRepository := range sessionRepositories {
			if isCodeRepositoryExcluded(sessionRepository.SourceDir, excludedRepositories) {
				continue
			}
			repository, ok := inspectCodeGitRepository(
				codeSessionRepositoryID(sessionRepository.ID), sessionRepository.LinkName,
				sessionRepository.WorktreeDir, sessionRepository.LinkName,
			)
			if !ok {
				continue
			}
			repository.Isolated = true
			repository.DeliveryStatus = sessionRepository.Status
			repository.BaseCommit = strings.TrimSpace(sessionRepository.BaseCommit)
			loadCodeGitSavedState(&repository, sessionRepository.BaseCommit)
			repositories = append(repositories, repository)
		}
		return repositories
	}
	if session.IsolationMode != codeIsolationDirect {
		if repository, ok := inspectCodeGitRepository("session", filepath.Base(session.WorkDir), session.WorkDir, ""); ok {
			repository.Isolated = session.WorktreeBranch != ""
			if repository.Isolated {
				repository.BaseCommit = strings.TrimSpace(session.BaseCommit)
				loadCodeGitSavedState(&repository, session.BaseCommit)
			}
			return []codeGitRepository{repository}
		}
	}
	prefixes := codeGitWorkspacePrefixes(session)
	repositoryRoots := sourceDirs
	if session.IsolationMode == codeIsolationDirect {
		if discovered, err := discoverCodeRepositoryRoots(sourceDirs); err == nil {
			repositoryRoots = discovered
		}
	}
	repositories := make([]codeGitRepository, 0, len(repositoryRoots))
	seen := make(map[string]struct{})
	for index, sourceDir := range repositoryRoots {
		cleanSource := filepath.Clean(sourceDir)
		if isCodeRepositoryExcluded(cleanSource, excludedRepositories) {
			continue
		}
		if _, exists := seen[cleanSource]; exists {
			continue
		}
		workspacePrefix := codeGitRepositoryWorkspacePrefix(cleanSource, sourceDirs, prefixes)
		repository, ok := inspectCodeGitRepository(
			fmt.Sprintf("source-%d", index), filepath.Base(cleanSource), cleanSource, workspacePrefix,
		)
		if !ok {
			continue
		}
		seen[cleanSource] = struct{}{}
		repositories = append(repositories, repository)
	}
	return repositories
}

func codeGitRepositoryWorkspacePrefix(repositoryRoot string, sourceDirs []string, prefixes map[string]string) string {
	for _, sourceDir := range sourceDirs {
		cleanSource := filepath.Clean(sourceDir)
		prefix := prefixes[cleanSource]
		if prefix == "" || (repositoryRoot != cleanSource && !isPathInside(repositoryRoot, cleanSource)) {
			continue
		}
		relative, err := filepath.Rel(cleanSource, repositoryRoot)
		if err != nil || relative == "." {
			return prefix
		}
		return filepath.ToSlash(filepath.Join(prefix, relative))
	}
	return ""
}

func loadCodeGitRepositoryStatus(repository codeGitRepository) (codeGitRepository, error) {
	output, truncated, err := runCodeGitReviewCommand(repository.root, false, 4*codeGitDiffOutputLimit, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return repository, err
	}
	if truncated {
		return repository, errors.New("Git 状态输出过大，请先减少工作区变更")
	}
	repository.Files = parseCodeGitStatus(output, repository.workspacePrefix)
	if len(repository.Files) > codeGitStatusFileLimit {
		repository.Files = repository.Files[:codeGitStatusFileLimit]
		repository.Truncated = true
	}
	for _, file := range repository.Files {
		if file.Staged {
			repository.StagedCount++
		}
		if file.Changed {
			repository.ChangedCount++
		}
		if file.Untracked {
			repository.UntrackedCount++
		}
	}
	workingStats, _, _ := runCodeGitReviewCommand(repository.root, false, codeGitDiffOutputLimit, "diff", "--numstat", "--no-ext-diff")
	stagedStats, _, _ := runCodeGitReviewCommand(repository.root, false, codeGitDiffOutputLimit, "diff", "--cached", "--numstat", "--no-ext-diff")
	repository.Additions, repository.Deletions = parseCodeGitNumstat(workingStats)
	repository.StagedAdditions, repository.StagedDeletions = parseCodeGitNumstat(stagedStats)
	// git diff 只认已跟踪文件，新建文件在 numstat 里一行都不算。
	// 不补这一笔，AI 刚写出来的整个新文件在 +/- 上就等于不存在。
	for _, file := range repository.Files {
		if file.Untracked {
			repository.Additions += countCodeTaskUntrackedLines(filepath.Join(repository.root, file.Path))
		}
	}
	return repository, nil
}

func loadCodeGitStatus(
	session *model.AIDevSession,
	sourceDirs []string,
	excludedRepositories []string,
) (codeGitStatus, error) {
	repositories := discoverCodeGitRepositories(session, sourceDirs, excludedRepositories)
	result := codeGitStatus{Available: len(repositories) > 0, Reason: "no_repository", Repositories: make([]codeGitRepository, 0, len(repositories))}
	if result.Available {
		result.Reason = ""
	}
	for _, repository := range repositories {
		loaded, err := loadCodeGitRepositoryStatus(repository)
		if err != nil {
			return codeGitStatus{}, err
		}
		result.Repositories = append(result.Repositories, loaded)
		result.Files += len(loaded.Files)
		result.Staged += loaded.StagedCount
		result.Changed += loaded.ChangedCount
		result.Untracked += loaded.UntrackedCount
		result.Additions += loaded.Additions
		result.Deletions += loaded.Deletions
		result.StagedAdditions += loaded.StagedAdditions
		result.StagedDeletions += loaded.StagedDeletions
	}
	return result, nil
}

func findCodeGitRepository(repositories []codeGitRepository, repositoryID string) (*codeGitRepository, error) {
	for index := range repositories {
		if repositories[index].ID == repositoryID {
			return &repositories[index], nil
		}
	}
	return nil, errors.New("Git 仓库不存在或不属于当前会话")
}

func findCodeGitFile(repository codeGitRepository, filePath, kind string) (*codeGitFile, error) {
	status, err := loadCodeGitRepositoryStatus(repository)
	if err != nil {
		return nil, err
	}
	cleanPath := filepath.ToSlash(path.Clean(strings.TrimSpace(filePath)))
	if cleanPath == "." || path.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "../") {
		return nil, errors.New("Git 文件路径无效")
	}
	for index := range status.Files {
		file := &status.Files[index]
		if filepath.ToSlash(file.Path) != cleanPath {
			continue
		}
		if kind == "staged" && file.Staged {
			return file, nil
		}
		if kind == "working" && (file.Changed || file.Untracked) {
			return file, nil
		}
	}
	return nil, errors.New("文件不在当前 Git 变更中")
}

func loadCodeGitDiff(repository codeGitRepository, file codeGitFile, kind string) (string, bool, error) {
	args := []string{"--literal-pathspecs", "diff", "--no-ext-diff", "--unified=3"}
	allowDiffExit := false
	if kind == "staged" {
		args = append(args, "--cached", "--", file.Path)
	} else if file.Untracked {
		allowDiffExit = true
		args = append(args, "--no-index", "--", "/dev/null", file.Path)
	} else {
		args = append(args, "--", file.Path)
	}
	return runCodeGitReviewCommand(repository.root, allowDiffExit, codeGitDiffOutputLimit, args...)
}

func updateCodeGitPathsStage(repository codeGitRepository, paths []string, staged bool) error {
	args := []string{"--literal-pathspecs", "add", "--"}
	if !staged {
		if _, err := runCodeGit(repository.root, "rev-parse", "--verify", "HEAD"); err == nil {
			args = []string{"--literal-pathspecs", "reset", "--quiet", "HEAD", "--"}
		} else {
			args = []string{"--literal-pathspecs", "rm", "--cached", "--quiet", "--"}
		}
	}
	args = append(args, paths...)
	_, err := runCodeGit(repository.root, args...)
	return err
}
