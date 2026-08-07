package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const unresolvedCodeConflict = "<<<" + "<<<< HEAD\n" +
	"current\n" +
	"=======\n" +
	"incoming\n" +
	">>>" + ">>>> remote\n"

func writeCodeConflictMarkerFile(t *testing.T, workDir string, staged bool) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(workDir, "conflicted.txt"), []byte(unresolvedCodeConflict), 0600); err != nil {
		t.Fatal(err)
	}
	if staged {
		if _, err := runCodeGit(workDir, "add", "--", "conflicted.txt"); err != nil {
			t.Fatal(err)
		}
	}
}

func requireCodeConflictMarkerRejected(t *testing.T, workDir, before string, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "未解决的冲突标记") {
		t.Fatalf("unresolved conflict marker should be rejected: %v", err)
	}
	after, headErr := runCodeGit(workDir, "rev-parse", "HEAD")
	if headErr != nil || after != before {
		t.Fatalf("rejected commit moved HEAD: before=%q after=%q err=%v", before, after, headErr)
	}
}

func TestSaveCodeGitRepositoryRejectsConflictMarkers(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 918)
	before, err := runCodeGit(session.WorkDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	writeCodeConflictMarkerFile(t, session.WorkDir, false)

	_, changed, saveErr := saveCodeGitRepository(session.WorkDir, "test: reject conflict marker")
	if changed {
		t.Fatal("rejected save reported a committed change")
	}
	requireCodeConflictMarkerRejected(t, session.WorkDir, before, saveErr)
}

func TestCommitCodeSessionWorktreeRejectsConflictMarkers(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 919)
	before, err := runCodeGit(session.WorkDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	writeCodeConflictMarkerFile(t, session.WorkDir, true)

	_, commitErr := commitCodeSessionWorktree(session, "test: reject conflict marker")
	requireCodeConflictMarkerRejected(t, session.WorkDir, before, commitErr)
}

func TestCommitCodeSessionRepositoryRejectsConflictMarkers(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 920)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	repository := repositories[0]
	before, err := runCodeGit(repository.WorktreeDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	writeCodeConflictMarkerFile(t, repository.WorktreeDir, true)

	_, commitErr := commitCodeSessionRepository(
		session,
		codeSessionRepositoryID(repository.ID),
		"test: reject conflict marker",
	)
	requireCodeConflictMarkerRejected(t, repository.WorktreeDir, before, commitErr)
}
