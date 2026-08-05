package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const (
	codeGitStatusFileLimit = 500
	codeGitDiffOutputLimit = 1024 * 1024
	codeGitCommandTimeout  = 15 * time.Second
)

type codeGitFile struct {
	Path           string `json:"path"`
	OldPath        string `json:"oldPath,omitempty"`
	WorkspacePath  string `json:"workspacePath"`
	IndexStatus    string `json:"indexStatus"`
	WorktreeStatus string `json:"worktreeStatus"`
	Staged         bool   `json:"staged"`
	Changed        bool   `json:"changed"`
	Untracked      bool   `json:"untracked"`
}

type codeGitRepository struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Branch          string        `json:"branch"`
	Files           []codeGitFile `json:"files"`
	StagedCount     int           `json:"stagedCount"`
	ChangedCount    int           `json:"changedCount"`
	UntrackedCount  int           `json:"untrackedCount"`
	Additions       int           `json:"additions"`
	Deletions       int           `json:"deletions"`
	StagedAdditions int           `json:"stagedAdditions"`
	StagedDeletions int           `json:"stagedDeletions"`
	Truncated       bool          `json:"truncated"`
	Isolated        bool          `json:"isolated"`
	DeliveryStatus  string        `json:"deliveryStatus,omitempty"`
	SavedCommits    int           `json:"savedCommits,omitempty"`
	HeadCommit      string        `json:"headCommit,omitempty"`
	root            string
	workspacePrefix string
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
}

type codeGitCappedBuffer struct {
	data      bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *codeGitCappedBuffer) Write(content []byte) (int, error) {
	written := len(content)
	remaining := buffer.limit - buffer.data.Len()
	if remaining > 0 {
		if len(content) > remaining {
			content = content[:remaining]
			buffer.truncated = true
		}
		_, _ = buffer.data.Write(content)
	} else if len(content) > 0 {
		buffer.truncated = true
	}
	return written, nil
}

func runCodeGitReviewCommand(workDir string, allowDiffExit bool, outputLimit int, args ...string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), codeGitCommandTimeout)
	defer cancel()
	commandArgs := append([]string{"-C", workDir}, args...)
	command := exec.CommandContext(ctx, "git", commandArgs...)
	command.Env = codeGitEnvironment()
	output := &codeGitCappedBuffer{limit: outputLimit}
	command.Stdout = output
	command.Stderr = output
	err := command.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", output.truncated, errors.New("Git 操作超时")
	}
	if err != nil {
		var exitError *exec.ExitError
		if !(allowDiffExit && errors.As(err, &exitError) && exitError.ExitCode() == 1) {
			message := strings.TrimSpace(output.data.String())
			if message == "" {
				message = err.Error()
			}
			return "", output.truncated, fmt.Errorf("Git 操作失败：%s", message)
		}
	}
	return output.data.String(), output.truncated, nil
}

func parseCodeGitStatus(output string, workspacePrefix string) []codeGitFile {
	records := strings.Split(output, "\x00")
	files := make([]codeGitFile, 0, len(records))
	for index := 0; index < len(records); index++ {
		record := records[index]
		if len(record) < 4 || record[2] != ' ' {
			continue
		}
		indexStatus := string(record[0])
		worktreeStatus := string(record[1])
		filePath := record[3:]
		oldPath := ""
		if strings.Contains("RC", indexStatus) || strings.Contains("RC", worktreeStatus) {
			if index+1 < len(records) {
				index++
				oldPath = records[index]
			}
		}
		untracked := indexStatus == "?" && worktreeStatus == "?"
		workspacePath := path.Join(workspacePrefix, filepath.ToSlash(filePath))
		files = append(files, codeGitFile{
			Path: filePath, OldPath: oldPath, WorkspacePath: workspacePath,
			IndexStatus: indexStatus, WorktreeStatus: worktreeStatus,
			Staged: !untracked && indexStatus != " ", Changed: !untracked && worktreeStatus != " ", Untracked: untracked,
		})
	}
	return files
}

func parseCodeGitNumstat(output string) (int, int) {
	additions, deletions := 0, 0
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 3 {
			continue
		}
		added, addedErr := strconv.Atoi(fields[0])
		deleted, deletedErr := strconv.Atoi(fields[1])
		if addedErr == nil {
			additions += added
		}
		if deletedErr == nil {
			deletions += deleted
		}
	}
	return additions, deletions
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
}

func discoverCodeGitRepositories(session *model.AIDevSession, sourceDirs []string) []codeGitRepository {
	if session == nil {
		return nil
	}
	if session.IsolationMode == codeIsolationMultiWorktree {
		sessionRepositories, err := loadCodeSessionRepositories(session.ID)
		if err != nil {
			return nil
		}
		repositories := make([]codeGitRepository, 0, len(sessionRepositories))
		for _, sessionRepository := range sessionRepositories {
			repository, ok := inspectCodeGitRepository(
				codeSessionRepositoryID(sessionRepository.ID), sessionRepository.LinkName,
				sessionRepository.WorktreeDir, sessionRepository.LinkName,
			)
			if !ok {
				continue
			}
			repository.Isolated = true
			repository.DeliveryStatus = sessionRepository.Status
			loadCodeGitSavedState(&repository, sessionRepository.BaseCommit)
			repositories = append(repositories, repository)
		}
		return repositories
	}
	if repository, ok := inspectCodeGitRepository("session", filepath.Base(session.WorkDir), session.WorkDir, ""); ok {
		repository.Isolated = session.WorktreeBranch != ""
		if repository.Isolated {
			loadCodeGitSavedState(&repository, session.BaseCommit)
		}
		return []codeGitRepository{repository}
	}
	prefixes := codeGitWorkspacePrefixes(session)
	repositories := make([]codeGitRepository, 0, len(sourceDirs))
	seen := make(map[string]struct{})
	for index, sourceDir := range sourceDirs {
		cleanSource := filepath.Clean(sourceDir)
		if _, exists := seen[cleanSource]; exists {
			continue
		}
		repository, ok := inspectCodeGitRepository(
			fmt.Sprintf("source-%d", index), filepath.Base(cleanSource), cleanSource, prefixes[cleanSource],
		)
		if !ok {
			continue
		}
		seen[cleanSource] = struct{}{}
		repositories = append(repositories, repository)
	}
	return repositories
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
	return repository, nil
}

func loadCodeGitStatus(session *model.AIDevSession, sourceDirs []string) (codeGitStatus, error) {
	repositories := discoverCodeGitRepositories(session, sourceDirs)
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

func getCodeGitSessionContext(c fiber.Ctx) (*model.AIDevSession, []string, error) {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return nil, nil, errors.New("会话 ID 无效")
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return nil, nil, err
	}
	if err := validateAIProjectWorkDirForClaims(session.WorkDir, claims); err != nil {
		return nil, nil, err
	}
	sourceDirs, err := getAISessionSourceDirs(session.ProjectID, claims)
	return session, sourceDirs, err
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

func GetCodeGitStatus(c fiber.Ctx) error {
	session, sourceDirs, err := getCodeGitSessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	status, err := loadCodeGitStatus(session, sourceDirs)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(status))
}

func GetCodeGitDiff(c fiber.Ctx) error {
	session, sourceDirs, err := getCodeGitSessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	repository, err := findCodeGitRepository(discoverCodeGitRepositories(session, sourceDirs), c.Query("repositoryId"))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	kind := c.Query("kind", "working")
	if kind != "working" && kind != "staged" {
		return c.JSON(e.Fail(errors.New("Git 差异类型无效")))
	}
	file, err := findCodeGitFile(*repository, c.Query("path"), kind)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	content, truncated, err := loadCodeGitDiff(*repository, *file, kind)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"repositoryId": repository.ID, "path": file.Path, "kind": kind, "content": content, "truncated": truncated}))
}

func UpdateCodeGitStage(c fiber.Ctx) error {
	var req struct {
		RepositoryID string   `json:"repositoryId"`
		Paths        []string `json:"paths"`
		Staged       bool     `json:"staged"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if len(req.Paths) == 0 || len(req.Paths) > 200 {
		return c.JSON(e.Fail(errors.New("请选择 1 到 200 个 Git 文件")))
	}
	session, sourceDirs, err := getCodeGitSessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var updated codeGitStatus
	err = runCodeSessionGitMutation(session, func(current *model.AIDevSession) error {
		repository, mutationErr := findCodeGitRepository(discoverCodeGitRepositories(current, sourceDirs), req.RepositoryID)
		if mutationErr != nil {
			return mutationErr
		}
		status, mutationErr := loadCodeGitRepositoryStatus(*repository)
		if mutationErr != nil {
			return mutationErr
		}
		validPaths := make(map[string]codeGitFile, len(status.Files))
		for _, file := range status.Files {
			validPaths[filepath.ToSlash(file.Path)] = file
		}
		paths := make([]string, 0, len(req.Paths))
		seen := make(map[string]struct{})
		for _, requestedPath := range req.Paths {
			cleanPath := filepath.ToSlash(path.Clean(strings.TrimSpace(requestedPath)))
			file, exists := validPaths[cleanPath]
			if !exists || (req.Staged && !file.Changed && !file.Untracked) || (!req.Staged && !file.Staged) {
				return fmt.Errorf("文件不允许执行当前暂存操作：%s", cleanPath)
			}
			if _, exists := seen[cleanPath]; exists {
				continue
			}
			seen[cleanPath] = struct{}{}
			paths = append(paths, cleanPath)
		}
		if mutationErr = updateCodeGitPathsStage(*repository, paths, req.Staged); mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = loadCodeGitStatus(current, sourceDirs)
		return mutationErr
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(updated))
}
