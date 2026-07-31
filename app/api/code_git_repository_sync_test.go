package api

import (
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
	if err := createCodeSessionWorktree(session, &model.AIGroup{SourceDirs: []string{repository}}); err != nil {
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
