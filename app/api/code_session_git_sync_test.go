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

func TestCodeSessionGitSyncRejectsDirtyAndLocalCommits(t *testing.T) {
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

func TestCodeSessionGitSyncDirtyRepositoryDisablesWholeSession(t *testing.T) {
	cleanRepository := createCodeGitRepository(t)
	dirtyRepository := createCodeGitRepository(t)
	branch, err := runCodeGit(cleanRepository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	baseCommit, err := runCodeGit(cleanRepository, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirtyRepository, "dirty.txt"), []byte("dirty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	targets := []codeSessionGitSyncTarget{
		{ID: "clean", Name: "clean", WorktreeDir: cleanRepository, Branch: branch, BaseCommit: baseCommit},
		{ID: "dirty", Name: "dirty", WorktreeDir: dirtyRepository, Branch: branch, BaseCommit: baseCommit},
	}
	status := inspectCodeSessionGitSyncTargets(1, targets, nil)
	if status.CanSync || status.Repositories[0].CanSync {
		t.Fatalf("dirty repository should disable session sync: %#v", status)
	}
}
