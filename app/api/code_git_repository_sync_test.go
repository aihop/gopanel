package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func createCodeRemoteRepository(t *testing.T) (string, string) {
	t.Helper()
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	if _, err := runCodeGit(filepath.Dir(remoteDir), "init", "--bare", remoteDir); err != nil {
		t.Fatal(err)
	}
	localDir := createCodeGitRepository(t)
	if _, err := runCodeGit(localDir, "remote", "add", "origin", remoteDir); err != nil {
		t.Fatal(err)
	}
	branch, err := runCodeGit(localDir, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(localDir, "push", "-u", "origin", branch); err != nil {
		t.Fatal(err)
	}
	return localDir, remoteDir
}

func cloneCodeRepository(t *testing.T, remoteDir string) string {
	t.Helper()
	cloneDir := filepath.Join(t.TempDir(), "clone")
	if _, err := runCodeGit(filepath.Dir(cloneDir), "clone", remoteDir, cloneDir); err != nil {
		t.Fatal(err)
	}
	return cloneDir
}

func commitCodeTestFile(t *testing.T, repository, name, content string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repository, name), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "add", name); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "test update"); err != nil {
		t.Fatal(err)
	}
	commit, err := runCodeGit(repository, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return commit
}

func TestPrepareCodeRepositoryFetchesAndFastForwards(t *testing.T) {
	localDir, remoteDir := createCodeRemoteRepository(t)
	updater := cloneCodeRepository(t, remoteDir)
	remoteCommit := commitCodeTestFile(t, updater, "remote.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}

	prepared, err := prepareCodeRepository(localDir)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.BaseCommit != remoteCommit || prepared.RemoteCommit != remoteCommit || prepared.SyncStatus != "fast_forwarded" {
		t.Fatalf("unexpected prepared repository: %#v", prepared)
	}
	if _, err := os.Stat(filepath.Join(localDir, "remote.txt")); err != nil {
		t.Fatalf("local repository was not fast-forwarded: %v", err)
	}
}

func TestPrepareCodeRepositoryRejectsLocalAhead(t *testing.T) {
	localDir, _ := createCodeRemoteRepository(t)
	commitCodeTestFile(t, localDir, "local.txt", "local\n")

	_, err := prepareCodeRepository(localDir)
	if err == nil || !strings.Contains(err.Error(), "领先") {
		t.Fatalf("local-ahead repository should be rejected: %v", err)
	}
}

func TestPrepareCodeRepositoryRejectsUnavailableRemote(t *testing.T) {
	localDir, _ := createCodeRemoteRepository(t)
	var err error
	if _, err := runCodeGit(localDir, "remote", "set-url", "origin", "https://127.0.0.1:1/gopanel.git"); err != nil {
		t.Fatal(err)
	}

	_, err = prepareCodeRepository(localDir)
	if err == nil || !strings.Contains(err.Error(), "同步仓库") {
		t.Fatalf("unavailable remote should block worktree creation: %v", err)
	}
}

func TestNormalizeCodeGitCommandErrorClassifiesAndRedactsAuthentication(t *testing.T) {
	err := normalizeCodeGitCommandError("fatal: unable to get password from user for 'https://secret-token@github.com/aihop/ainode.git'")
	if !errors.Is(err, errCodeGitAuthentication) {
		t.Fatalf("authentication failure was not classified: %v", err)
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("authentication failure leaked credentials: %v", err)
	}
	if !strings.Contains(err.Error(), "credential helper") {
		t.Fatalf("authentication failure did not include recovery guidance: %v", err)
	}
}

func TestNormalizeCodeGitCommandErrorPreservesNonAuthenticationFailure(t *testing.T) {
	err := normalizeCodeGitCommandError("fatal: unable to access remote: connection refused")
	if errors.Is(err, errCodeGitAuthentication) || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("network failure was misclassified: %v", err)
	}
}

func TestRunCodeGitRedactsAuthenticationFailure(t *testing.T) {
	repository := createCodeGitRepository(t)
	if _, err := runCodeGit(
		repository,
		"config", "alias.auth-failure", "!f() { echo \"fatal: authentication failed for https://user:secret@github.com/private/repo.git\" >&2; exit 1; }; f",
	); err != nil {
		t.Fatal(err)
	}
	_, err := runCodeGit(repository, "auth-failure")
	if !errors.Is(err, errCodeGitAuthentication) {
		t.Fatalf("Git authentication failure was not classified: %v", err)
	}
	if strings.Contains(err.Error(), "user:secret") {
		t.Fatalf("Git authentication failure leaked credentials: %v", err)
	}
}

func TestCodeDeliveryPolicyDefaultsToMain(t *testing.T) {
	repository := createCodeGitRepository(t)
	currentBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if currentBranch != "main" {
		if _, err := runCodeGit(repository, "branch", "main"); err != nil {
			t.Fatal(err)
		}
	}
	policy, err := normalizeCodeDeliveryPolicy([]string{repository}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if policy.PrimaryRepository != repository || policy.DeliveryBranch != "main" {
		t.Fatalf("unexpected default delivery policy: %#v", policy)
	}
}

func TestCodeDeliveryPolicyRejectsMissingConfiguredBranch(t *testing.T) {
	repository := createCodeGitRepository(t)
	_, err := normalizeCodeDeliveryPolicy([]string{repository}, repository, "release")
	if err == nil || !strings.Contains(err.Error(), "不存在配置的交付分支") {
		t.Fatalf("missing configured branch should be rejected: %v", err)
	}
}

func TestPrepareCodeRepositoryRequiresConfiguredBranchCheckout(t *testing.T) {
	repository := createCodeGitRepository(t)
	currentBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "branch", "main"); err != nil && currentBranch != "main" {
		t.Fatal(err)
	}
	if currentBranch == "main" {
		if _, err := runCodeGit(repository, "switch", "-c", "develop"); err != nil {
			t.Fatal(err)
		}
	}
	candidate := codeRepositoryCandidate{SourceDir: repository}
	_, err = prepareCodeRepositoryCandidateForBranch(candidate, false, "main")
	if err == nil || !strings.Contains(err.Error(), "请先切换到交付分支 main") {
		t.Fatalf("unchecked delivery branch should be rejected: %v", err)
	}
}

func TestPrepareCodeRepositoryRestoresDetachedConfiguredBranch(t *testing.T) {
	repository := createCodeGitRepository(t)
	targetBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "checkout", "--detach", "HEAD"); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareCodeRepositoryCandidateForBranch(codeRepositoryCandidate{SourceDir: repository}, false, targetBranch)
	if err != nil {
		t.Fatal(err)
	}
	currentBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil || currentBranch != targetBranch || prepared.TargetBranch != targetBranch {
		t.Fatalf("detached repository was not restored: branch=%q prepared=%#v err=%v", currentBranch, prepared, err)
	}
}

func TestPrepareCodeRepositoryRejectsDetachedDifferentCommit(t *testing.T) {
	repository := createCodeGitRepository(t)
	targetBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	previousCommit, err := runCodeGit(repository, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	commitCodeTestFile(t, repository, "next.txt", "next\n")
	if _, err := runCodeGit(repository, "checkout", "--detach", previousCommit); err != nil {
		t.Fatal(err)
	}
	_, err = prepareCodeRepositoryCandidateForBranch(codeRepositoryCandidate{SourceDir: repository}, false, targetBranch)
	if err == nil || !strings.Contains(err.Error(), "当前提交不等于交付分支") {
		t.Fatalf("unexpected detached commit should be rejected: %v", err)
	}
}

func TestPrepareCodeRepositoryRestoresDetachedRemoteOnlyBranch(t *testing.T) {
	repository, _ := createCodeRemoteRepository(t)
	targetBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "checkout", "--detach", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "branch", "-D", targetBranch); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareCodeRepositoryCandidateForBranch(codeRepositoryCandidate{SourceDir: repository}, false, targetBranch)
	if err != nil {
		t.Fatal(err)
	}
	currentBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil || currentBranch != targetBranch || prepared.TargetBranch != targetBranch {
		t.Fatalf("remote-only branch was not restored: branch=%q prepared=%#v err=%v", currentBranch, prepared, err)
	}
}

func TestCreateCodeSessionWorktreeRestoresDetachedPrimaryRepository(t *testing.T) {
	withAIProjectBaseDir(t)
	repository := createCodeGitRepository(t)
	targetBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "checkout", "--detach", "HEAD"); err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: 92, UserID: 7, WorkDir: repository}
	project := &model.AIProject{
		SourceDirs: []string{repository}, PrimaryRepository: repository, DeliveryBranch: targetBranch,
	}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	currentBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil || currentBranch != targetBranch || session.TargetBranch != targetBranch {
		t.Fatalf("task worktree did not restore primary repository: branch=%q session=%#v err=%v", currentBranch, session, err)
	}
	if _, err := os.Stat(session.WorkDir); err != nil {
		t.Fatalf("task worktree was not created: %v", err)
	}
}

func TestRefreshCodeRepositoryTargetRejectsBranchSwitch(t *testing.T) {
	repository := createCodeGitRepository(t)
	targetBranch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "switch", "-c", "other"); err != nil {
		t.Fatal(err)
	}
	_, err = refreshCodeRepositoryTarget(repository, targetBranch, "")
	if err == nil || !strings.Contains(err.Error(), "交付目标分支") {
		t.Fatalf("branch switch should be rejected: %v", err)
	}
}

func TestSyncCodeWorktreeWithUpdatedTarget(t *testing.T) {
	withAIProjectBaseDir(t)
	repository := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 91, UserID: 7, WorkDir: repository}
	if err := createCodeSessionWorktree(session, &model.AIProject{SourceDirs: []string{repository}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	commitCodeTestFile(t, session.WorkDir, "worktree.txt", "worktree\n")
	targetCommit := commitCodeTestFile(t, repository, "target.txt", "target\n")

	if err := syncCodeWorktreeWithTarget(session.WorkDir, session.TargetBranch); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "merge-base", "--is-ancestor", targetCommit, "HEAD"); err != nil {
		t.Fatalf("updated target was not merged into worktree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(session.WorkDir, "target.txt")); err != nil {
		t.Fatalf("updated target content unavailable in worktree: %v", err)
	}
}

func createGitlinkRepositoryTree(t *testing.T) (string, string) {
	t.Helper()
	parent := createCodeGitRepository(t)
	child := filepath.Join(parent, "themes", "custom")
	if err := os.MkdirAll(filepath.Dir(child), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(filepath.Dir(child), "init", child); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "staged.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "working.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(child, "add", "staged.txt", "working.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(child, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "initial child"); err != nil {
		t.Fatal(err)
	}
	childCommit, err := runCodeGit(child, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(parent, "update-index", "--add", "--cacheinfo", "160000,"+childCommit+",themes/custom"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(parent, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "track child gitlink"); err != nil {
		t.Fatal(err)
	}
	return parent, child
}

func TestDiscoverCodeRepositoriesIncludesGitlinkWithoutGitmodules(t *testing.T) {
	parent, child := createGitlinkRepositoryTree(t)
	candidates, err := discoverCodeRepositoryCandidates([]string{parent})
	if err != nil || len(candidates) != 2 {
		t.Fatalf("unexpected candidates: %#v, %v", candidates, err)
	}
	var parentCandidate, childCandidate *codeRepositoryCandidate
	for index := range candidates {
		candidate := &candidates[index]
		if candidate.SourceDir == parent {
			parentCandidate = candidate
		}
		if candidate.SourceDir == child {
			childCandidate = candidate
		}
	}
	if parentCandidate == nil || parentCandidate.Dirty {
		t.Fatalf("parent repository incorrectly dirty: %#v", parentCandidate)
	}
	if childCandidate == nil || childCandidate.ParentSourceDir != parent || childCandidate.GitlinkPath != "themes/custom" {
		t.Fatalf("gitlink relationship unavailable: %#v", childCandidate)
	}
}

func TestPrepareGitlinkRepositoriesRejectsDirtyChildWhenSnapshotDisabled(t *testing.T) {
	parent, child := createGitlinkRepositoryTree(t)
	if err := os.WriteFile(filepath.Join(child, "working.txt"), []byte("dirty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := prepareDiscoveredCodeRepositories([]string{parent}, false)
	if err == nil || !strings.Contains(err.Error(), "源仓库 custom 存在未提交变更") {
		t.Fatalf("dirty child should be rejected explicitly: %v", err)
	}
}

func TestGitlinkPointerChangeKeepsParentDirty(t *testing.T) {
	parent, child := createGitlinkRepositoryTree(t)
	commitCodeTestFile(t, child, "next.txt", "next\n")
	candidates, err := discoverCodeRepositoryCandidates([]string{parent})
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if candidate.SourceDir == parent {
			if !candidate.Dirty {
				t.Fatal("parent gitlink pointer change must remain visible")
			}
			return
		}
	}
	t.Fatal("parent repository unavailable")
}

func TestCodeSnapshotRejectsSymlinkDestinationDirectory(t *testing.T) {
	worktree := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(worktree, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := ensureCodeSnapshotDestination(worktree, filepath.Join(worktree, "escape", "nested")); err == nil {
		t.Fatal("snapshot destination escaped through symlink")
	}
}

func TestCodeSnapshotRejectsOversizedFile(t *testing.T) {
	worktree := t.TempDir()
	source := filepath.Join(t.TempDir(), "large.bin")
	file, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxCodeSnapshotFileBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := copyCodeSnapshotFile(worktree, source, filepath.Join(worktree, "large.bin")); err == nil {
		t.Fatal("oversized snapshot file should be rejected")
	}
}
