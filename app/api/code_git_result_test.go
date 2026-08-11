package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCodeGitResultStatusIncludesSavedChanges(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 931)
	if err := os.WriteFile(filepath.Join(session.WorkDir, "result.txt"), []byte("saved result\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "test: save reviewed result"); err != nil {
		t.Fatal(err)
	}

	status, err := loadCodeGitResultStatus(session, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Available || !status.ReviewReady || status.ReviewRevision == "" || status.Scope != "result" {
		t.Fatalf("result review was not ready: %#v", status)
	}
	if len(status.Repositories) != 1 || len(status.Repositories[0].Files) != 1 {
		t.Fatalf("unexpected result repositories: %#v", status.Repositories)
	}
	file := status.Repositories[0].Files[0]
	if file.Path != "result.txt" || file.ResultStatus != "A" {
		t.Fatalf("unexpected result file: %#v", file)
	}
	diff, truncated, err := loadCodeGitResultFileDiff(session, nil, "session", "result.txt")
	if err != nil || truncated || !strings.Contains(diff, "+saved result") {
		t.Fatalf("unexpected result diff: truncated=%v err=%v output=%q", truncated, err, diff)
	}
}

func TestLoadCodeGitResultStatusExcludesUnsavedWorkspace(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 932)
	if err := os.WriteFile(filepath.Join(session.WorkDir, "pending.txt"), []byte("pending\n"), 0600); err != nil {
		t.Fatal(err)
	}

	status, err := loadCodeGitResultStatus(session, nil)
	if err != nil {
		t.Fatal(err)
	}
	if status.ReviewReady || status.ReviewRevision != "" || status.Files != 0 {
		t.Fatalf("unsaved workspace leaked into task changes: %#v", status)
	}
	if _, _, err := loadCodeGitResultFileDiff(session, nil, "session", "pending.txt"); err == nil {
		t.Fatal("unsaved file should only be available in commit and merge")
	}
}

func TestLoadCodeGitResultStatusKeepsUnsavedEditsOutOfCommittedDiff(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 934)
	filePath := filepath.Join(session.WorkDir, "result.txt")
	if err := os.WriteFile(filePath, []byte("saved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "test: save result before later edits"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("saved\nunsaved\n"), 0600); err != nil {
		t.Fatal(err)
	}

	status, err := loadCodeGitResultStatus(session, nil)
	if err != nil {
		t.Fatal(err)
	}
	if status.ReviewReady || status.Files != 1 || status.Additions != 1 {
		t.Fatalf("unexpected committed task changes: %#v", status)
	}
	diff, truncated, err := loadCodeGitResultFileDiff(session, nil, "session", "result.txt")
	if err != nil || truncated || !strings.Contains(diff, "+saved") || strings.Contains(diff, "+unsaved") {
		t.Fatalf("unsaved edit leaked into committed diff: truncated=%v err=%v output=%q", truncated, err, diff)
	}
}

func TestValidateCodeGitReviewRevisionRejectsChangedCommit(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 933)
	if err := os.WriteFile(filepath.Join(session.WorkDir, "first.txt"), []byte("first\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "test: first review"); err != nil {
		t.Fatal(err)
	}
	status, err := loadCodeGitResultStatus(session, nil)
	if err != nil || !status.ReviewReady {
		t.Fatalf("first review unavailable: %#v, %v", status, err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "second.txt"), []byte("second\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "test: second review"); err != nil {
		t.Fatal(err)
	}
	if err := validateCodeGitReviewRevision(session, status.ReviewRevision); err == nil {
		t.Fatal("stale review revision should be rejected")
	}
}

func TestLoadCodeGitHistoryStaysInsideTaskRange(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 935)
	if err := os.WriteFile(filepath.Join(session.WorkDir, "first.txt"), []byte("first\n"), 0600); err != nil {
		t.Fatal(err)
	}
	firstResult, err := saveCodeSessionWorktree(session, "test: first history commit")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "second.txt"), []byte("second\n"), 0600); err != nil {
		t.Fatal(err)
	}
	secondResult, err := saveCodeSessionWorktree(session, "test: second history commit")
	if err != nil {
		t.Fatal(err)
	}

	history, err := loadCodeGitHistory(session, nil)
	if err != nil || history.Commits != 2 || len(history.Repositories) != 1 {
		t.Fatalf("unexpected task history: %#v, %v", history, err)
	}
	commits := history.Repositories[0].Commits
	firstCommit, secondCommit := firstResult.Commit, secondResult.Commit
	if commits[0].Commit != secondCommit || commits[1].Commit != firstCommit {
		t.Fatalf("unexpected commit order: %#v", commits)
	}
	diff, truncated, err := loadCodeGitHistoryDiff(session, nil, "session", firstCommit)
	if err != nil || truncated || !strings.Contains(diff, "+first") {
		t.Fatalf("unexpected history diff: truncated=%v err=%v output=%q", truncated, err, diff)
	}
	if _, _, err := loadCodeGitHistoryDiff(session, nil, "session", session.BaseCommit); err == nil {
		t.Fatal("task base commit should not be readable as task history")
	}
	if _, _, err := loadCodeGitHistoryDiff(session, nil, "session", firstCommit[:8]); err == nil {
		t.Fatal("abbreviated revisions should not be accepted")
	}
}
