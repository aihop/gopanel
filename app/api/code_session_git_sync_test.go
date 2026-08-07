package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func createCodeSessionSyncFixture(t *testing.T) (*model.AIDevSession, string) {
	t.Helper()
	withAIProjectBaseDir(t)
	sourceDir, remoteDir := createCodeRemoteRepository(t)
	session := &model.AIDevSession{ID: 301, UserID: 7}
	if err := createCodeSessionWorktree(session, &model.AIProject{SourceDirs: []string{sourceDir}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	return session, remoteDir
}

func TestCodeSessionGitSyncDetectsAndFastForwardsRemote(t *testing.T) {
	session, remoteDir := createCodeSessionSyncFixture(t)
	updater := cloneCodeRepository(t, remoteDir)
	remoteCommit := commitCodeTestFile(t, updater, "remote.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	targets, err := codeSessionGitSyncTargets(session)
	if err != nil {
		t.Fatal(err)
	}
	fetchErrors := fetchCodeSessionGitTargets(targets)
	status := inspectCodeSessionGitSyncTargets(session.ID, targets, fetchErrors)
	if len(status.Repositories) != 1 || !status.Repositories[0].CanSync || status.Repositories[0].Status != "behind" {
		t.Fatalf("unexpected sync status: %#v", status)
	}
	if _, err := runCodeGit(session.WorkDir, "merge", "--ff-only", codeSessionGitRemoteRef(targets[0])); err != nil {
		t.Fatal(err)
	}
	commit, err := runCodeGit(session.WorkDir, "rev-parse", "HEAD")
	if err != nil || commit != remoteCommit {
		t.Fatalf("worktree commit = %q, want %q: %v", commit, remoteCommit, err)
	}
	targets[0].BaseCommit = remoteCommit
	status = inspectCodeSessionGitSyncTargets(session.ID, targets, fetchErrors)
	if status.Status != "synced" || status.Repositories[0].CanSync {
		t.Fatalf("unexpected post-sync status: %#v", status)
	}
}

func TestCodeSessionGitSyncRejectsDirtyAndLocalOnlyCommits(t *testing.T) {
	session, _ := createCodeSessionSyncFixture(t)
	targets, err := codeSessionGitSyncTargets(session)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "dirty.txt"), []byte("dirty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status := inspectCodeSessionGitSyncTargets(session.ID, targets, nil)
	if status.Repositories[0].Status != "dirty" || status.Repositories[0].CanSync {
		t.Fatalf("dirty worktree should be blocked: %#v", status)
	}
	if err := os.Remove(filepath.Join(session.WorkDir, "dirty.txt")); err != nil {
		t.Fatal(err)
	}
	commitCodeTestFile(t, session.WorkDir, "local.txt", "local\n")
	status = inspectCodeSessionGitSyncTargets(session.ID, targets, nil)
	if status.Repositories[0].Status != "local_ahead" || status.Repositories[0].CanSync {
		t.Fatalf("local commits should be blocked: %#v", status)
	}
}

func TestCodeSessionGitSyncMergesDivergedRemote(t *testing.T) {
	session, remoteDir := createCodeSessionSyncFixture(t)
	localCommit := commitCodeTestFile(t, session.WorkDir, "local.txt", "local\n")
	updater := cloneCodeRepository(t, remoteDir)
	remoteCommit := commitCodeTestFile(t, updater, "remote.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	targets, err := codeSessionGitSyncTargets(session)
	if err != nil {
		t.Fatal(err)
	}
	status := inspectCodeSessionGitSyncTargets(session.ID, targets, fetchCodeSessionGitTargets(targets))
	state := status.Repositories[0]
	if state.Status != "diverged" || !state.CanSync || state.Reason != "merge_required" {
		t.Fatalf("diverged session should allow merge sync: %#v", state)
	}
	mergedCommit, err := syncCodeSessionGitTarget(targets[0], state)
	if err != nil || mergedCommit == localCommit || mergedCommit == remoteCommit {
		t.Fatalf("remote was not merged into the session: commit=%q err=%v", mergedCommit, err)
	}
	for _, ancestor := range []string{localCommit, remoteCommit} {
		if _, err := runCodeGit(session.WorkDir, "merge-base", "--is-ancestor", ancestor, "HEAD"); err != nil {
			t.Fatalf("commit %s is not in merged session history: %v", ancestor, err)
		}
	}
	status = inspectCodeSessionGitSyncTargets(session.ID, targets, nil)
	if status.Repositories[0].Status != "integrated" || !status.Repositories[0].CanSync {
		t.Fatalf("merged remote should be ready to update its baseline: %#v", status.Repositories[0])
	}
	targets[0].BaseCommit = remoteCommit
	status = inspectCodeSessionGitSyncTargets(session.ID, targets, nil)
	if status.Repositories[0].Status != "local_ahead" || status.Repositories[0].CanSync {
		t.Fatalf("session commits should remain ahead of the updated baseline: %#v", status.Repositories[0])
	}
}

func TestCodeSessionGitSyncLeavesConflictsInIsolatedWorktree(t *testing.T) {
	session, remoteDir := createCodeSessionSyncFixture(t)
	commitCodeTestFile(t, session.WorkDir, "README.md", "session\n")
	updater := cloneCodeRepository(t, remoteDir)
	commitCodeTestFile(t, updater, "README.md", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	targets, err := codeSessionGitSyncTargets(session)
	if err != nil {
		t.Fatal(err)
	}
	state := inspectCodeSessionGitSyncTargets(session.ID, targets, fetchCodeSessionGitTargets(targets)).Repositories[0]
	if state.Status != "diverged" || !state.CanSync {
		t.Fatalf("conflicting histories should be offered for merge: %#v", state)
	}
	if _, err := syncCodeSessionGitTarget(targets[0], state); err == nil || len(codeGitConflictFiles(session.WorkDir)) != 1 {
		t.Fatalf("merge conflict was not preserved for resolution: %v", err)
	}
	t.Cleanup(func() { _, _ = runCodeGit(session.WorkDir, "merge", "--abort") })
}

func TestCodeSessionGitSyncRejectsGitlinkRepositories(t *testing.T) {
	repository := createCodeGitRepository(t)
	branch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	commit, err := runCodeGit(repository, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	target := codeSessionGitSyncTarget{
		ID: "gitlink", Name: "gitlink", WorktreeDir: repository,
		Branch: branch, BaseCommit: commit, HasGitlink: true,
	}
	status := inspectCodeSessionGitSyncTargets(1, []codeSessionGitSyncTarget{target}, nil)
	if status.Repositories[0].Status != "blocked" || status.Repositories[0].Reason != "gitlink_coordination_required" {
		t.Fatalf("gitlink repository should be blocked: %#v", status)
	}
}

func TestCodeSessionGitSyncDirtyRepositoryDoesNotBlockIndependentRepository(t *testing.T) {
	session, remoteDir := createCodeSessionSyncFixture(t)
	updater := cloneCodeRepository(t, remoteDir)
	commitCodeTestFile(t, updater, "remote.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	cleanTargets, err := codeSessionGitSyncTargets(session)
	if err != nil {
		t.Fatal(err)
	}
	dirtyRepository := createCodeGitRepository(t)
	branch, err := runCodeGit(dirtyRepository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	baseCommit, err := runCodeGit(dirtyRepository, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirtyRepository, "dirty.txt"), []byte("dirty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	targets := []codeSessionGitSyncTarget{
		cleanTargets[0],
		{ID: "dirty", Name: "dirty", WorktreeDir: dirtyRepository, Branch: branch, BaseCommit: baseCommit},
	}
	status := inspectCodeSessionGitSyncTargets(session.ID, targets, fetchCodeSessionGitTargets(targets))
	if !status.CanSync || !status.Repositories[0].CanSync || status.Repositories[1].CanSync {
		t.Fatalf("dirty repository should not block an independent clean repository: %#v", status)
	}
}
