package api

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxCodeDiscoveredRepositories = 50
	codeGitFetchTimeout           = 60 * time.Second
)

var codeGitScanExcludedDirectories = map[string]struct{}{
	".git": {}, ".cache": {}, ".next": {}, ".nuxt": {}, ".output": {},
	".pnpm-store": {}, ".turbo": {}, ".venv": {}, "build": {}, "coverage": {},
	"dist": {}, "node_modules": {}, "target": {}, "vendor": {},
}

type codePreparedRepository struct {
	SourceDir    string
	TargetBranch string
	BaseCommit   string
	RemoteName   string
	RemoteCommit string
	SyncStatus   string
}

func discoverCodeRepositoryRoots(sourceDirs []string) ([]string, error) {
	seen := make(map[string]struct{})
	repositories := make([]string, 0, len(sourceDirs))
	for _, sourceDir := range sourceDirs {
		boundary, err := filepath.EvalSymlinks(filepath.Clean(sourceDir))
		if err != nil {
			return nil, fmt.Errorf("项目目录不可访问：%s", sourceDir)
		}
		err = filepath.WalkDir(boundary, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() {
				return nil
			}
			if path != boundary {
				if _, excluded := codeGitScanExcludedDirectories[entry.Name()]; excluded {
					return filepath.SkipDir
				}
			}
			if _, err := os.Lstat(filepath.Join(path, ".git")); err != nil {
				return nil
			}
			root, err := runCodeGit(path, "rev-parse", "--show-toplevel")
			if err != nil {
				return nil
			}
			root, err = filepath.EvalSymlinks(filepath.Clean(root))
			if err != nil || root != path {
				return nil
			}
			if superproject, _ := runCodeGit(path, "rev-parse", "--show-superproject-working-tree"); strings.TrimSpace(superproject) != "" {
				return filepath.SkipDir
			}
			if _, exists := seen[root]; !exists {
				seen[root] = struct{}{}
				repositories = append(repositories, root)
				if len(repositories) > maxCodeDiscoveredRepositories {
					return fmt.Errorf("项目目录中 Git 仓库超过 %d 个，请缩小目录范围", maxCodeDiscoveredRepositories)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(repositories)
	return repositories, nil
}

func prepareCodeRepository(sourceDir string) (codePreparedRepository, error) {
	status, err := runCodeGit(sourceDir, "status", "--porcelain")
	if err != nil {
		return codePreparedRepository{}, err
	}
	if strings.TrimSpace(status) != "" {
		return codePreparedRepository{}, fmt.Errorf("源仓库 %s 存在未提交变更", filepath.Base(sourceDir))
	}
	targetBranch, err := runCodeGit(sourceDir, "branch", "--show-current")
	if err != nil || strings.TrimSpace(targetBranch) == "" {
		return codePreparedRepository{}, fmt.Errorf("源仓库 %s 当前处于 detached HEAD，无法确定目标分支", filepath.Base(sourceDir))
	}
	prepared := codePreparedRepository{SourceDir: sourceDir, TargetBranch: targetBranch, SyncStatus: "local"}
	remoteName, remoteRef := codeRepositoryRemoteTracking(sourceDir, targetBranch)
	if remoteName != "" {
		if _, err := fetchCodeRepository(sourceDir, remoteName); err != nil {
			return codePreparedRepository{}, fmt.Errorf("同步仓库 %s 失败：%w", filepath.Base(sourceDir), err)
		}
		if remoteRef == "" {
			candidate := "refs/remotes/" + remoteName + "/" + targetBranch
			if _, err := runCodeGit(sourceDir, "show-ref", "--verify", candidate); err == nil {
				remoteRef = candidate
			} else {
				remoteName = ""
			}
		}
	}
	baseCommit, remoteCommit, syncStatus, err := fastForwardCodeRepository(sourceDir, targetBranch, remoteRef)
	if err != nil {
		return codePreparedRepository{}, err
	}
	prepared.BaseCommit = baseCommit
	prepared.RemoteName = remoteName
	prepared.RemoteCommit = remoteCommit
	prepared.SyncStatus = syncStatus
	return prepared, nil
}

func codeRepositoryRemoteTracking(sourceDir, branch string) (string, string) {
	remoteName, _ := runCodeGit(sourceDir, "config", "--get", "branch."+branch+".remote")
	remoteName = strings.TrimSpace(remoteName)
	if remoteName == "." {
		return "", ""
	}
	upstream, _ := runCodeGit(sourceDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if remoteName != "" {
		return remoteName, strings.TrimSpace(upstream)
	}
	remotes, _ := runCodeGit(sourceDir, "remote")
	for _, candidate := range strings.Fields(remotes) {
		if candidate == "origin" {
			return candidate, ""
		}
	}
	return "", ""
}

func fastForwardCodeRepository(sourceDir, targetBranch, remoteRef string) (string, string, string, error) {
	localCommit, err := runCodeGit(sourceDir, "rev-parse", targetBranch)
	if err != nil {
		return "", "", "", err
	}
	if remoteRef == "" {
		return localCommit, "", "local", nil
	}
	remoteCommit, err := runCodeGit(sourceDir, "rev-parse", remoteRef)
	if err != nil {
		return "", "", "", fmt.Errorf("远端目标分支不可用：%s", remoteRef)
	}
	if localCommit == remoteCommit {
		return localCommit, remoteCommit, "synced", nil
	}
	if _, err := runCodeGit(sourceDir, "merge-base", "--is-ancestor", localCommit, remoteCommit); err == nil {
		if _, err := runCodeGit(sourceDir, "merge", "--ff-only", remoteRef); err != nil {
			return "", "", "", err
		}
		return remoteCommit, remoteCommit, "fast_forwarded", nil
	}
	if _, err := runCodeGit(sourceDir, "merge-base", "--is-ancestor", remoteCommit, localCommit); err == nil {
		return "", "", "", fmt.Errorf("本地目标分支 %s 领先于 %s，请先推送或选择本地模式", targetBranch, remoteRef)
	}
	return "", "", "", fmt.Errorf("本地目标分支 %s 与 %s 已分叉，请先人工处理", targetBranch, remoteRef)
}

func refreshCodeRepositoryTarget(sourceDir, targetBranch, remoteName string) (string, error) {
	status, err := runCodeGit(sourceDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		return "", fmt.Errorf("源仓库 %s 存在未提交变更，无法安全合并", filepath.Base(sourceDir))
	}
	currentBranch, err := runCodeGit(sourceDir, "branch", "--show-current")
	if err != nil || currentBranch != targetBranch {
		return "", fmt.Errorf("源仓库 %s 当前分支为 %s，交付目标分支为 %s", filepath.Base(sourceDir), currentBranch, targetBranch)
	}
	if remoteName == "" {
		commit, err := runCodeGit(sourceDir, "rev-parse", targetBranch)
		return commit, err
	}
	if _, err := fetchCodeRepository(sourceDir, remoteName); err != nil {
		return "", fmt.Errorf("同步仓库 %s 失败：%w", filepath.Base(sourceDir), err)
	}
	_, remoteRef := codeRepositoryRemoteTracking(sourceDir, targetBranch)
	if remoteRef == "" {
		remoteRef = "refs/remotes/" + remoteName + "/" + targetBranch
	}
	commit, _, _, err := fastForwardCodeRepository(sourceDir, targetBranch, remoteRef)
	return commit, err
}

func syncCodeWorktreeWithTarget(worktreeDir, targetBranch string) error {
	if strings.TrimSpace(targetBranch) == "" {
		return errors.New("交付目标分支不可用")
	}
	if _, err := runCodeGit(worktreeDir, "merge-base", "--is-ancestor", targetBranch, "HEAD"); err == nil {
		return nil
	}
	status, err := runCodeGit(worktreeDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		return errors.New("Worktree 仍有未提交变更，请先提交")
	}
	if _, err := runCodeGit(
		worktreeDir,
		"-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.local",
		"-c", "commit.gpgsign=false", "merge", "--no-edit", targetBranch,
	); err != nil {
		conflicts := codeGitConflictFiles(worktreeDir)
		if len(conflicts) > 0 {
			return fmt.Errorf("目标分支更新与 Worktree 冲突，请在隔离工作区解决：%s", strings.Join(conflicts, ", "))
		}
		_, _ = runCodeGit(worktreeDir, "merge", "--abort")
		return err
	}
	return nil
}

func fetchCodeRepository(sourceDir, remoteName string) (string, error) {
	return runCodeGitWithTimeout(
		sourceDir, codeGitFetchTimeout,
		"-c", "credential.interactive=never", "fetch", "--prune", remoteName,
	)
}

func repositoryWithinSourceDirs(repository string, sourceDirs []string) bool {
	for _, sourceDir := range sourceDirs {
		resolved, err := filepath.EvalSymlinks(filepath.Clean(sourceDir))
		if err == nil && (repository == resolved || isPathInside(repository, resolved)) {
			return true
		}
	}
	return false
}

func prepareDiscoveredCodeRepositories(sourceDirs []string) ([]codePreparedRepository, error) {
	roots, err := discoverCodeRepositoryRoots(sourceDirs)
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, errors.New("项目目录中未发现 Git 仓库")
	}
	prepared := make([]codePreparedRepository, 0, len(roots))
	for _, root := range roots {
		repository, err := prepareCodeRepository(root)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, repository)
	}
	return prepared, nil
}
