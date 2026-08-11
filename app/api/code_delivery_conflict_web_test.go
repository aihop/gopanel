package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestCodeDeliveryConflictFileCanBeResolvedAndCommitted(t *testing.T) {
	context := createCodeDeliveryConflictFixture(t)
	file, err := readCodeDeliveryConflictFile(context, "conflict.txt")
	if err != nil {
		t.Fatal(err)
	}
	if file.Resolved || file.Binary || file.BaseContent != "base\n" || file.MainContent != "main\n" || file.TaskContent != "task\n" {
		t.Fatalf("unexpected conflict file: %#v", file)
	}
	resolved, err := saveCodeDeliveryConflictFile(context, file.Path, "content", "resolved\n", file.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Resolved || resolved.ResultContent != "resolved\n" || len(codeGitConflictFiles(context.WorkDir)) != 0 {
		t.Fatalf("conflict was not resolved: %#v", resolved)
	}
	commit, err := finalizeCodeDeliveryConflictCommit(context)
	if err != nil {
		t.Fatal(err)
	}
	parents, err := runCodeGit(context.WorkDir, "show", "-s", "--format=%P", commit)
	if err != nil || !strings.Contains(parents, context.SourceCommit) || !strings.Contains(parents, context.TaskCommit) {
		t.Fatalf("merge parents are incomplete: %q, %v", parents, err)
	}
	content, err := os.ReadFile(filepath.Join(context.WorkDir, "conflict.txt"))
	if err != nil || string(content) != "resolved\n" {
		t.Fatalf("resolved content mismatch: %q, %v", content, err)
	}
}

func TestCodeDeliveryConflictRejectsUnsafeOrStaleWrites(t *testing.T) {
	context := createCodeDeliveryConflictFixture(t)
	if _, _, err := findCodeDeliveryConflictContext([]codeDeliveryConflictContext{*context}, "session", "../secret"); err == nil {
		t.Fatal("path traversal was accepted")
	}
	file, err := readCodeDeliveryConflictFile(context, "conflict.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeDeliveryConflictFile(context, file.Path, "main", "", "stale-version"); err == nil {
		t.Fatal("stale version was accepted")
	}
	if _, err := finalizeCodeDeliveryConflictCommit(context); err == nil {
		t.Fatal("unresolved conflict was committed")
	}
}

func TestCodeDeliveryConflictRejectsRemainingConflictMarkers(t *testing.T) {
	context := createCodeDeliveryConflictFixture(t)
	file, err := readCodeDeliveryConflictFile(context, "conflict.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeDeliveryConflictFile(context, file.Path, "content", unresolvedCodeConflict, file.Version); err == nil ||
		!strings.Contains(err.Error(), "仍包含 Git 冲突标记") {
		t.Fatalf("remaining conflict markers were accepted: %v", err)
	}
	current, readErr := readCodeDeliveryConflictFile(context, file.Path)
	if readErr != nil || current.Version != file.Version {
		t.Fatalf("rejected conflict marker changed the file: %#v, %v", current, readErr)
	}
}

func TestCodeDeliveryConflictSupportsDeletingTaskVersion(t *testing.T) {
	context := createCodeDeliveryConflictFixture(t)
	file, err := readCodeDeliveryConflictFile(context, "conflict.txt")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := saveCodeDeliveryConflictFile(context, file.Path, "delete", "", file.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Resolved || resolved.ResultExists {
		t.Fatalf("deleted conflict remains unresolved: %#v", resolved)
	}
	if _, err := finalizeCodeDeliveryConflictCommit(context); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyManualCodeDeliveryConflictRequiresTaskCommit(t *testing.T) {
	context := createCodeDeliveryConflictFixture(t)
	if err := verifyManualCodeDeliveryConflict(context); err == nil || !strings.Contains(err.Error(), "尚未包含任务分支") {
		t.Fatalf("unmerged task was accepted: %v", err)
	}
	if _, err := runCodeGit(context.SourceDir, "merge", "--no-ff", "--no-edit", context.TaskCommit); err == nil {
		t.Fatal("fixture manual merge should conflict")
	}
	if err := os.WriteFile(filepath.Join(context.SourceDir, "conflict.txt"), []byte("manual\n"), 0600); err != nil {
		t.Fatal(err)
	}
	commitCodeConflictFixture(t, context.SourceDir, "manual merge")
	if err := verifyManualCodeDeliveryConflict(context); err != nil {
		t.Fatalf("completed manual merge was rejected: %v", err)
	}
}

func TestEnrichCodeDeliveryConflictRepositoryRestoresMultiRepositoryState(t *testing.T) {
	database := withCodeGovernanceDB(t)
	repository := &model.AIDevSessionRepository{SessionID: 77, ProjectID: 2, LinkName: "api", Status: codeDeliveryJobConflict}
	if err := database.Create(repository).Error; err != nil {
		t.Fatal(err)
	}
	results := []codeRepositoryDeliveryResult{{RepositoryID: codeSessionRepositoryID(repository.ID), Status: codeDeliveryPrepared}}
	enrichCodeDeliveryConflictRepository(repository.SessionID, results, []string{"conflict.go"})
	if results[0].Status != codeDeliveryJobConflict || len(results[0].ConflictFiles) != 1 {
		t.Fatalf("multi repository conflict was not enriched: %#v", results)
	}
}

func createCodeDeliveryConflictFixture(t *testing.T) *codeDeliveryConflictContext {
	t.Helper()
	repository := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repository, "conflict.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	commitCodeConflictFixture(t, repository, "base")
	mainBranch, _ := runCodeGit(repository, "branch", "--show-current")
	if _, err := runCodeGit(repository, "checkout", "-b", "task"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "conflict.txt"), []byte("task\n"), 0600); err != nil {
		t.Fatal(err)
	}
	commitCodeConflictFixture(t, repository, "task")
	taskCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
	if _, err := runCodeGit(repository, "checkout", mainBranch); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "conflict.txt"), []byte("main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	commitCodeConflictFixture(t, repository, "main")
	mainCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
	workDir := filepath.Join(t.TempDir(), "delivery")
	if _, err := runCodeGit(repository, "worktree", "add", "--detach", workDir, mainCommit); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = runCodeGit(repository, "worktree", "remove", "--force", workDir) })
	if _, err := runCodeGit(workDir, "merge", "--no-ff", "--no-edit", taskCommit); err == nil || len(codeGitConflictFiles(workDir)) != 1 {
		t.Fatalf("fixture did not create a conflict: %v", err)
	}
	return &codeDeliveryConflictContext{
		RepositoryID: "session", Name: "repository", Branch: "task", TargetBranch: mainBranch,
		SourceDir: repository, WorkDir: workDir, SourceCommit: mainCommit, TaskCommit: taskCommit, Files: []string{"conflict.txt"},
		Delivery: nil,
	}
}

func commitCodeConflictFixture(t *testing.T, workDir, message string) {
	t.Helper()
	if _, err := runCodeGit(workDir, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(
		workDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local",
		"-c", "commit.gpgsign=false", "commit", "-m", message,
	); err != nil {
		t.Fatal(err)
	}
}
