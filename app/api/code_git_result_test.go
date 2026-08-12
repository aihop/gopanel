package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
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

func TestDiscoverCodeGitResultRepositoriesExcludesConfiguredRepository(t *testing.T) {
	database := withCodeGovernanceDB(t)
	includedDir := createCodeGitRepository(t)
	excludedDir := createCodeGitRepository(t)
	includedBase, _ := runCodeGit(includedDir, "rev-parse", "HEAD")
	excludedBase, _ := runCodeGit(excludedDir, "rev-parse", "HEAD")
	includedHead := commitCodeSummaryFiles(t, includedDir, map[string]string{"included.txt": "included\n"})
	excludedHead := commitCodeSummaryFiles(t, excludedDir, map[string]string{"excluded.txt": "excluded\n"})
	if err := os.WriteFile(filepath.Join(excludedDir, "stale-dirty.txt"), []byte("dirty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{
		UserID: 1, ProjectID: 1, Title: "result", WorkDir: t.TempDir(), IsolationMode: codeIsolationMultiWorktree,
	}
	project := &model.AIProject{
		ID: 1, Name: "project", CreatorID: 1, ExcludedRepositories: []string{excludedDir},
	}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	rows := []model.AIDevSessionRepository{
		{SessionID: session.ID, ProjectID: 1, SourceDir: includedDir, WorktreeDir: includedDir, LinkName: "included", Branch: "task-included", BaseCommit: strings.TrimSpace(includedBase), WorktreeCommit: includedHead, Status: "working"},
		{SessionID: session.ID, ProjectID: 1, SourceDir: excludedDir, WorktreeDir: excludedDir, LinkName: "excluded", Branch: "task-excluded", BaseCommit: strings.TrimSpace(excludedBase), WorktreeCommit: excludedHead, Status: "working"},
	}
	if err := database.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	repositories := discoverCodeGitResultRepositories(session, []string{excludedDir})
	if len(repositories) != 1 || repositories[0].Name != "included" {
		t.Fatalf("excluded repository leaked into task result: %#v", repositories)
	}
	status, err := loadCodeGitResultStatus(session, []string{excludedDir})
	if err != nil || len(status.Repositories) != 1 || status.Repositories[0].Name != "included" || status.ReviewRevision == "" {
		t.Fatalf("unexpected filtered result review: %#v, %v", status, err)
	}
	if err := validateCodeGitReviewRevision(session, status.ReviewRevision); err != nil {
		t.Fatalf("filtered review revision was rejected during delivery: %v", err)
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
	if history.Repositories[0].Branch != session.WorktreeBranch ||
		history.Repositories[0].TargetBranch != session.TargetBranch {
		t.Fatalf("unexpected task branch merge state: %#v", history.Repositories[0])
	}
	commits := history.Repositories[0].Commits
	firstCommit, secondCommit := firstResult.Commit, secondResult.Commit
	if commits[0].Commit != secondCommit || commits[1].Commit != firstCommit {
		t.Fatalf("unexpected commit order: %#v", commits)
	}
	if commits[0].Merged || commits[1].Merged {
		t.Fatalf("unmerged commits were incorrectly identified: %#v", commits)
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
	if _, err := runCodeGit(session.SourceWorkDir, "merge", "--ff-only", firstCommit); err != nil {
		t.Fatal(err)
	}
	partiallyMergedHistory, err := loadCodeGitHistory(session, nil)
	if err != nil || partiallyMergedHistory.Repositories[0].Commits[0].Merged ||
		!partiallyMergedHistory.Repositories[0].Commits[1].Merged {
		t.Fatalf("per-commit merge state was not identified: %#v, %v", partiallyMergedHistory, err)
	}
}
