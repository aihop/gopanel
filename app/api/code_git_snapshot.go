package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

const (
	maxCodeSnapshotFileBytes  = 64 << 20
	maxCodeSnapshotPatchBytes = 128 << 20
	maxCodeSnapshotTotalBytes = 512 << 20
)

type codeRepositoryCandidate struct {
	SourceDir       string
	ParentSourceDir string
	GitlinkPath     string
	Dirty           bool
}

func discoverCodeRepositoryCandidates(sourceDirs []string) ([]codeRepositoryCandidate, error) {
	roots, err := discoverCodeRepositoryRoots(sourceDirs)
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, nil
	}
	candidates := make([]codeRepositoryCandidate, len(roots))
	for index, root := range roots {
		candidates[index].SourceDir = root
		for _, parent := range roots {
			if parent == root || !isPathInside(root, parent) {
				continue
			}
			relative, relativeErr := filepath.Rel(parent, root)
			if relativeErr != nil || !codeRepositoryPathIsGitlink(parent, relative) {
				continue
			}
			if candidates[index].ParentSourceDir == "" || len(parent) > len(candidates[index].ParentSourceDir) {
				candidates[index].ParentSourceDir = parent
				candidates[index].GitlinkPath = filepath.ToSlash(relative)
			}
		}
	}
	parents := make(map[string]bool)
	for _, candidate := range candidates {
		if candidate.ParentSourceDir != "" {
			parents[candidate.ParentSourceDir] = true
		}
	}
	for index := range candidates {
		status, statusErr := codeRepositoryStatus(candidates[index].SourceDir, parents[candidates[index].SourceDir])
		if statusErr != nil {
			return nil, statusErr
		}
		candidates[index].Dirty = strings.TrimSpace(status) != ""
	}
	return candidates, nil
}

func codeRepositoryStatus(sourceDir string, ignoreNestedDirty bool) (string, error) {
	args := []string{"status", "--porcelain"}
	if ignoreNestedDirty {
		args = append(args, "--ignore-submodules=dirty")
	}
	return runCodeGit(sourceDir, args...)
}

func codeRepositoryPathIsGitlink(sourceDir, relativePath string) bool {
	entry, err := runCodeGit(sourceDir, "ls-files", "-s", "--", filepath.ToSlash(relativePath))
	if err != nil || strings.TrimSpace(entry) == "" {
		return false
	}
	fields := strings.Fields(strings.Split(entry, "\n")[0])
	return len(fields) >= 4 && fields[0] == "160000"
}

func codeCandidateNames(candidates []codeRepositoryCandidate) []string {
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Dirty {
			name := filepath.Base(candidate.SourceDir)
			if candidate.ParentSourceDir != "" && candidate.GitlinkPath != "" {
				name = filepath.ToSlash(filepath.Join(filepath.Base(candidate.ParentSourceDir), candidate.GitlinkPath))
			}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func applyCodeRepositorySnapshot(repository codePreparedRepository, worktreeDir string, prepared []codePreparedRepository) error {
	excluded := directCodeRepositoryChildren(repository.SourceDir, prepared)
	if repository.Snapshot {
		if err := applyCodeRepositoryDiff(repository, worktreeDir, excluded, true); err != nil {
			return err
		}
		if err := applyCodeRepositoryDiff(repository, worktreeDir, excluded, false); err != nil {
			return err
		}
		if err := copyCodeRepositoryUntrackedFiles(repository.SourceDir, worktreeDir, excluded); err != nil {
			return err
		}
	}
	for _, child := range excluded {
		commit, err := runCodeGit(child.SourceDir, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		if err := updateCodeGitlink(worktreeDir, child.GitlinkPath, commit); err != nil {
			return err
		}
	}
	return nil
}

func applyCodeRepositoryDiff(repository codePreparedRepository, worktreeDir string, excluded []codePreparedRepository, staged bool) error {
	args := []string{"diff", "--binary"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", ".")
	for _, child := range excluded {
		args = append(args, ":(exclude)"+child.GitlinkPath)
	}
	patch, err := runCodeGitBytes(repository.SourceDir, nil, args...)
	if err != nil || len(bytes.TrimSpace(patch)) == 0 {
		return err
	}
	if len(patch) > maxCodeSnapshotPatchBytes {
		return fmt.Errorf("仓库 %s 的修改补丁超过 %d MiB 上限", filepath.Base(repository.SourceDir), maxCodeSnapshotPatchBytes>>20)
	}
	applyArgs := []string{"apply", "--whitespace=nowarn"}
	if staged {
		applyArgs = append(applyArgs, "--index")
	}
	applyArgs = append(applyArgs, "-")
	if _, err := runCodeGitBytes(worktreeDir, patch, applyArgs...); err != nil {
		state := "未暂存"
		if staged {
			state = "已暂存"
		}
		return fmt.Errorf("重放仓库 %s 的%s修改失败：%w", filepath.Base(repository.SourceDir), state, err)
	}
	return nil
}

func directCodeRepositoryChildren(parent string, prepared []codePreparedRepository) []codePreparedRepository {
	children := make([]codePreparedRepository, 0)
	for _, repository := range prepared {
		if repository.ParentSourceDir == parent {
			children = append(children, repository)
		}
	}
	return children
}

func copyCodeRepositoryUntrackedFiles(sourceDir, worktreeDir string, excluded []codePreparedRepository) error {
	output, err := runCodeGitBytes(sourceDir, nil, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return err
	}
	var totalBytes int64
	for _, rawPath := range bytes.Split(output, []byte{0}) {
		if len(rawPath) == 0 {
			continue
		}
		relative := filepath.Clean(string(rawPath))
		if filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || codePathInsideExcludedRepository(relative, excluded) {
			continue
		}
		copiedBytes, copyErr := copyCodeSnapshotFile(worktreeDir, filepath.Join(sourceDir, relative), filepath.Join(worktreeDir, relative))
		if copyErr != nil {
			return fmt.Errorf("复制仓库 %s 的未跟踪文件失败：%w", filepath.Base(sourceDir), copyErr)
		}
		totalBytes += copiedBytes
		if totalBytes > maxCodeSnapshotTotalBytes {
			return fmt.Errorf("仓库 %s 的未跟踪快照超过 %d MiB 上限", filepath.Base(sourceDir), maxCodeSnapshotTotalBytes>>20)
		}
	}
	return nil
}

func codePathInsideExcludedRepository(relative string, excluded []codePreparedRepository) bool {
	relative = filepath.ToSlash(relative)
	for _, repository := range excluded {
		path := strings.TrimSuffix(repository.GitlinkPath, "/")
		if relative == path || strings.HasPrefix(relative, path+"/") {
			return true
		}
	}
	return false
}

func copyCodeSnapshotFile(worktreeDir, source, destination string) (int64, error) {
	info, err := os.Lstat(source)
	if err != nil {
		return 0, err
	}
	if err := ensureCodeSnapshotDestination(worktreeDir, filepath.Dir(destination)); err != nil {
		return 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 0, errors.New("未跟踪快照不支持符号链接")
	}
	if !info.Mode().IsRegular() {
		return 0, errors.New("快照包含不支持的特殊文件")
	}
	if info.Size() > maxCodeSnapshotFileBytes {
		return 0, fmt.Errorf("文件 %s 超过 %d MiB 上限", filepath.Base(source), maxCodeSnapshotFileBytes>>20)
	}
	input, err := os.Open(source)
	if err != nil {
		return 0, err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return 0, err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return 0, copyErr
	}
	return info.Size(), closeErr
}

func ensureCodeSnapshotDestination(worktreeDir, destinationDir string) error {
	relative, err := filepath.Rel(worktreeDir, destinationDir)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("快照目标目录超出 Worktree")
	}
	current := filepath.Clean(worktreeDir)
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			if mkdirErr := os.Mkdir(current, 0750); mkdirErr != nil {
				return mkdirErr
			}
			continue
		}
		if statErr != nil {
			return statErr
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("快照目标路径包含非目录或符号链接：%s", current)
		}
	}
	return nil
}

func runCodeGitBytes(workDir string, input []byte, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), codeWorktreeCommandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", append([]string{"-C", workDir}, args...)...)
	command.Env = codeGitEnvironment()
	if input != nil {
		command.Stdin = bytes.NewReader(input)
	}
	output, err := command.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, errors.New("Git 操作超时")
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("Git 操作失败：%s", message)
	}
	return output, nil
}

func updateCodeGitlink(worktreeDir, gitlinkPath, commit string) error {
	current, _ := runCodeGit(worktreeDir, "ls-files", "-s", "--", gitlinkPath)
	fields := strings.Fields(strings.Split(current, "\n")[0])
	if len(fields) >= 2 && fields[0] == "160000" && fields[1] == commit {
		return nil
	}
	_, err := runCodeGit(worktreeDir, "update-index", "--add", "--cacheinfo", "160000,"+commit+","+gitlinkPath)
	return err
}

func syncCodeSessionRepositoryGitlinks(repositories []model.AIDevSessionRepository) error {
	parents := make(map[string]*model.AIDevSessionRepository, len(repositories))
	for index := range repositories {
		parents[repositories[index].SourceDir] = &repositories[index]
	}
	for index := range repositories {
		child := &repositories[index]
		parent := parents[child.ParentSourceDir]
		if parent == nil || child.GitlinkPath == "" {
			continue
		}
		commit := child.MergeCommit
		if commit == "" {
			var err error
			commit, err = runCodeGit(child.WorktreeDir, "rev-parse", "HEAD")
			if err != nil {
				return err
			}
		}
		if err := updateCodeGitlink(parent.WorktreeDir, child.GitlinkPath, commit); err != nil {
			return fmt.Errorf("同步父仓库 %s 的子仓库指针失败：%w", parent.LinkName, err)
		}
	}
	return nil
}

func commitCodeRepositoryGitlinkUpdates(repository *model.AIDevSessionRepository, repositories []model.AIDevSessionRepository) error {
	expected := make(map[string]struct{})
	for _, child := range repositories {
		if child.ParentSourceDir == repository.SourceDir && child.GitlinkPath != "" && child.MergeCommit != "" {
			expected[filepath.ToSlash(child.GitlinkPath)] = struct{}{}
		}
	}
	if len(expected) == 0 {
		return nil
	}
	staged, err := runCodeGitBytes(repository.WorktreeDir, nil, "diff", "--cached", "--name-only", "-z")
	if err != nil || len(staged) == 0 {
		return err
	}
	for _, rawPath := range bytes.Split(staged, []byte{0}) {
		if len(rawPath) == 0 {
			continue
		}
		path := filepath.ToSlash(string(rawPath))
		if _, allowed := expected[path]; !allowed {
			return fmt.Errorf("仓库 %s 包含非预期的暂存修改：%s", repository.LinkName, path)
		}
	}
	if _, err := runCodeGit(
		repository.WorktreeDir,
		codeGitAuthoredArgs(
			"-c", "commit.gpgsign=false", "commit", "-m", "chore: update nested repository pointers",
		)...,
	); err != nil {
		return err
	}
	commit, err := runCodeGit(repository.WorktreeDir, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	repository.WorktreeCommit = commit
	return nil
}

func validateCodeRepositoryMergeStatus(repository *model.AIDevSessionRepository, repositories []model.AIDevSessionRepository) error {
	status, err := runCodeGitBytes(repository.SourceDir, nil, "status", "--porcelain=v1", "-z", "--ignore-submodules=dirty")
	if err != nil || len(status) == 0 {
		return err
	}
	expected := make(map[string]string)
	for _, child := range repositories {
		if child.ParentSourceDir == repository.SourceDir && child.GitlinkPath != "" && (child.Status == codeDeliveryMerged || child.Status == codeDeliveryCompleted) {
			expected[filepath.ToSlash(child.GitlinkPath)] = child.MergeCommit
		}
	}
	for _, entry := range bytes.Split(status, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		if len(entry) < 4 {
			return errors.New("源仓库状态无法安全识别")
		}
		path := string(entry[3:])
		commit := expected[filepath.ToSlash(path)]
		if commit == "" || entry[0] != ' ' || entry[1] != 'M' {
			return fmt.Errorf("源仓库 %s 存在未提交变更，无法安全合并：%s %s", repository.LinkName, string(entry[:2]), path)
		}
		childDir := filepath.Join(repository.SourceDir, filepath.FromSlash(path))
		childHead, headErr := runCodeGit(childDir, "rev-parse", "HEAD")
		if headErr != nil || childHead != commit {
			return fmt.Errorf("源仓库 %s 的子仓库指针与交付结果不一致", repository.LinkName)
		}
		delete(expected, filepath.ToSlash(path))
	}
	return nil
}
