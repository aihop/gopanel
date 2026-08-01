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
	SourceDir       string
	ParentSourceDir string
	GitlinkPath     string
	TargetBranch    string
	BaseCommit      string
	RemoteName      string
	RemoteBranch    string
	RemoteCommit    string
	SyncStatus      string
	Snapshot        bool
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
	status, err := codeRepositoryStatus(sourceDir, false)
	if err != nil {
		return codePreparedRepository{}, err
	}
	return prepareCodeRepositoryCandidate(codeRepositoryCandidate{SourceDir: sourceDir, Dirty: strings.TrimSpace(status) != ""}, false)
}

func prepareCodeRepositoryCandidate(candidate codeRepositoryCandidate, includeUncommitted bool) (codePreparedRepository, error) {
	return prepareCodeRepositoryCandidateForBranch(candidate, includeUncommitted, "")
}

func prepareCodeRepositoryCandidateForBranch(candidate codeRepositoryCandidate, includeUncommitted bool, deliveryBranch string) (codePreparedRepository, error) {
	dirty := candidate.Dirty
	if dirty && !includeUncommitted {
		return codePreparedRepository{}, fmt.Errorf("源仓库 %s 存在未提交变更；如需保留这些修改，请启用未提交改动快照", filepath.Base(candidate.SourceDir))
	}
	targetBranch, err := resolveCodeRepositoryTargetBranch(candidate.SourceDir, strings.TrimSpace(deliveryBranch), dirty)
	if err != nil {
		return codePreparedRepository{}, err
	}
	prepared := codePreparedRepository{
		SourceDir: candidate.SourceDir, ParentSourceDir: candidate.ParentSourceDir,
		GitlinkPath: candidate.GitlinkPath, TargetBranch: targetBranch, SyncStatus: "local", Snapshot: dirty,
	}
	remoteName, remoteRef := codeRepositoryRemoteTracking(candidate.SourceDir, targetBranch)
	localOnly := false
	if remoteName != "" {
		if _, err := fetchCodeRepository(candidate.SourceDir, remoteName); err != nil {
			if !isCodeGitFetchUnavailable(err) {
				return codePreparedRepository{}, fmt.Errorf("同步仓库 %s 失败：%w", filepath.Base(candidate.SourceDir), err)
			}
			localOnly = true
			remoteRef = ""
		}
		if remoteRef == "" && !localOnly {
			candidate := "refs/remotes/" + remoteName + "/" + targetBranch
			if _, err := runCodeGit(prepared.SourceDir, "show-ref", "--verify", candidate); err == nil {
				remoteRef = candidate
			} else {
				remoteName = ""
			}
		}
	}
	baseCommit, remoteCommit, syncStatus, err := codeRepositoryBaseline(prepared.SourceDir, targetBranch, remoteRef, dirty)
	if err != nil {
		return codePreparedRepository{}, err
	}
	prepared.BaseCommit = baseCommit
	prepared.RemoteName = remoteName
	prepared.RemoteBranch = codeRemoteBranch(remoteName, remoteRef, targetBranch)
	prepared.RemoteCommit = remoteCommit
	prepared.SyncStatus = syncStatus
	if localOnly {
		prepared.SyncStatus = "local_only"
	}
	return prepared, nil
}

func resolveCodeRepositoryTargetBranch(sourceDir, deliveryBranch string, dirty bool) (string, error) {
	currentBranch, err := runCodeGit(sourceDir, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	currentBranch = strings.TrimSpace(currentBranch)
	if deliveryBranch == "" {
		if currentBranch == "" {
			return "", fmt.Errorf("源仓库 %s 当前处于 detached HEAD，无法确定目标分支", filepath.Base(sourceDir))
		}
		return currentBranch, nil
	}
	if currentBranch == deliveryBranch {
		return deliveryBranch, nil
	}
	if currentBranch != "" {
		return "", fmt.Errorf("主交付仓库当前分支为 %s，请先切换到交付分支 %s", currentBranch, deliveryBranch)
	}
	if dirty {
		return "", fmt.Errorf("主交付仓库处于 detached HEAD 且存在未提交变更，无法安全恢复到交付分支 %s", deliveryBranch)
	}
	headCommit, headErr := runCodeGit(sourceDir, "rev-parse", "HEAD")
	targetRef := "refs/heads/" + deliveryBranch
	targetCommit, targetErr := runCodeGit(sourceDir, "rev-parse", targetRef)
	createLocalBranch := false
	if targetErr != nil {
		targetRef = "refs/remotes/origin/" + deliveryBranch
		targetCommit, targetErr = runCodeGit(sourceDir, "rev-parse", targetRef)
		createLocalBranch = targetErr == nil
	}
	if headErr != nil || targetErr != nil || strings.TrimSpace(headCommit) != strings.TrimSpace(targetCommit) {
		return "", fmt.Errorf("主交付仓库处于 detached HEAD，且当前提交不等于交付分支 %s，请先确认目标分支", deliveryBranch)
	}
	switchArgs := []string{"switch", "--", deliveryBranch}
	if createLocalBranch {
		switchArgs = []string{"switch", "-c", deliveryBranch, "--track", "origin/" + deliveryBranch}
	}
	if _, err := runCodeGit(sourceDir, switchArgs...); err != nil {
		return "", fmt.Errorf("主交付仓库无法自动恢复到交付分支 %s：%w", deliveryBranch, err)
	}
	return deliveryBranch, nil
}

func isCodeGitFetchUnavailable(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"git 操作超时", "authentication failed", "unable to get password",
		"could not read username", "terminal prompts disabled", "permission denied (publickey)",
		"could not resolve host", "temporary failure in name resolution", "network is unreachable",
		"no route to host", "failed to connect", "connection refused", "connection reset",
		"connection timed out", "operation timed out", "connection closed by remote host",
		"ssl_error_syscall", "tls handshake timeout",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func codeRepositoryBaseline(sourceDir, targetBranch, remoteRef string, dirty bool) (string, string, string, error) {
	if dirty {
		localCommit, err := runCodeGit(sourceDir, "rev-parse", targetBranch)
		if err != nil {
			return "", "", "", err
		}
		if remoteRef == "" {
			return localCommit, "", "snapshot", nil
		}
		remoteCommit, err := runCodeGit(sourceDir, "rev-parse", remoteRef)
		if err != nil {
			return "", "", "", fmt.Errorf("远端目标分支不可用：%s", remoteRef)
		}
		if localCommit != remoteCommit {
			return "", "", "", fmt.Errorf("源仓库 %s 存在未提交变更且本地分支未与远端同步，请先处理分支差异", filepath.Base(sourceDir))
		}
		return localCommit, remoteCommit, "snapshot", nil
	}
	return fastForwardCodeRepository(sourceDir, targetBranch, remoteRef)
}

func codeRemoteBranch(remoteName, remoteRef, fallback string) string {
	remoteRef = strings.TrimSpace(remoteRef)
	for _, prefix := range []string{"refs/remotes/" + remoteName + "/", remoteName + "/"} {
		if remoteName != "" && strings.HasPrefix(remoteRef, prefix) {
			return strings.TrimPrefix(remoteRef, prefix)
		}
	}
	return fallback
}

func codeRepositoryRemoteTracking(sourceDir, branch string) (string, string) {
	remoteName, _ := runCodeGit(sourceDir, "config", "--get", "branch."+branch+".remote")
	remoteName = strings.TrimSpace(remoteName)
	if remoteName == "." {
		return "", ""
	}
	mergeRef, _ := runCodeGit(sourceDir, "config", "--get", "branch."+branch+".merge")
	upstream := ""
	if remoteName != "" && strings.HasPrefix(strings.TrimSpace(mergeRef), "refs/heads/") {
		upstream = remoteName + "/" + strings.TrimPrefix(strings.TrimSpace(mergeRef), "refs/heads/")
	}
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
		"-c", "credential.interactive=never", "fetch", "--prune", "--", remoteName,
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

func prepareDiscoveredCodeRepositories(sourceDirs []string, includeUncommitted ...bool) ([]codePreparedRepository, error) {
	return prepareDiscoveredCodeRepositoriesWithPolicy(sourceDirs, codeDeliveryPolicy{}, includeUncommitted...)
}

func prepareDiscoveredCodeRepositoriesWithPolicy(sourceDirs []string, policy codeDeliveryPolicy, includeUncommitted ...bool) ([]codePreparedRepository, error) {
	candidates, err := discoverCodeRepositoryCandidates(sourceDirs)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, errors.New("项目目录中未发现 Git 仓库")
	}
	allowSnapshot := len(includeUncommitted) > 0 && includeUncommitted[0]
	prepared := make([]codePreparedRepository, 0, len(candidates))
	for _, candidate := range candidates {
		targetBranch := ""
		if candidate.SourceDir == policy.PrimaryRepository {
			targetBranch = policy.DeliveryBranch
		}
		repository, err := prepareCodeRepositoryCandidateForBranch(candidate, allowSnapshot, targetBranch)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, repository)
	}
	return prepared, nil
}
