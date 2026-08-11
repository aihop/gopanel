package api

import (
	"fmt"
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

func TestDiscoverCodeDeliveryConflictFilesIncludesStagedMarkers(t *testing.T) {
	context := createCodeDeliveryConflictFixture(t)
	if _, err := runCodeGit(context.WorkDir, "add", "--", "conflict.txt"); err != nil {
		t.Fatal(err)
	}
	if conflicts := codeGitConflictFiles(context.WorkDir); len(conflicts) != 0 {
		t.Fatalf("fixture should no longer have unmerged index entries: %#v", conflicts)
	}
	files := discoverCodeDeliveryConflictFiles(context.WorkDir)
	if len(files) != 1 || files[0] != "conflict.txt" {
		t.Fatalf("staged conflict markers were not discovered: %#v", files)
	}
	view := codeDeliveryConflictRepositoryViews([]codeDeliveryConflictContext{*context})
	if len(view) != 1 || view[0].Resolved != 0 || len(view[0].UnresolvedFiles) != 1 {
		t.Fatalf("staged conflict markers were shown as resolved: %#v", view)
	}
}

func TestMultiRepositoryDeliveryCollectsAllConflicts(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 950)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	for index := range repositories {
		commitCodeTestFile(t, repositories[index].SourceDir, "README.md", fmt.Sprintf("main-%d\n", index))
		commitCodeTestFile(t, repositories[index].WorktreeDir, "README.md", fmt.Sprintf("task-%d\n", index))
	}
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	result, err := prepareCodeMultiRepositoryDeliveryWithProgress(session, nil)
	if err != nil || result.Status != codeDeliveryJobConflict {
		t.Fatalf("prepare multi repository conflict: %#v, %v", result, err)
	}
	conflicted := 0
	for index := range result.Repositories {
		if result.Repositories[index].Status != codeDeliveryJobConflict {
			continue
		}
		conflicted++
		if len(result.Repositories[index].ConflictFiles) != 1 || result.Repositories[index].ConflictFiles[0] != "README.md" {
			t.Fatalf("repository conflict files are incomplete: %#v", result.Repositories[index])
		}
	}
	if conflicted != 2 {
		t.Fatalf("collected %d conflicted repositories, want 2: %#v", conflicted, result.Repositories)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		if files := discoverCodeDeliveryConflictFiles(stored[index].IntegrationWorkDir); len(files) != 1 || files[0] != "README.md" {
			t.Fatalf("integration conflict missing for %s: %#v", stored[index].LinkName, files)
		}
		status, statusErr := runCodeGit(stored[index].WorktreeDir, "status", "--porcelain")
		if statusErr != nil || strings.TrimSpace(status) != "" {
			t.Fatalf("task worktree %s was polluted: %q, %v", stored[index].LinkName, status, statusErr)
		}
	}
}

func TestCodeRepositoryConflictBlocksAllAncestors(t *testing.T) {
	root := model.AIDevSessionRepository{SourceDir: "/workspace/root"}
	child := model.AIDevSessionRepository{SourceDir: "/workspace/root/child", ParentSourceDir: root.SourceDir}
	leaf := model.AIDevSessionRepository{
		SourceDir: "/workspace/root/child/leaf", ParentSourceDir: child.SourceDir, Status: codeDeliveryJobConflict,
	}
	repositories := []model.AIDevSessionRepository{root, child, leaf}
	if !codeRepositoryHasConflictedDescendant(&repositories[0], repositories) ||
		!codeRepositoryHasConflictedDescendant(&repositories[1], repositories) ||
		codeRepositoryHasConflictedDescendant(&repositories[2], repositories) {
		t.Fatalf("nested conflict dependency was not propagated: %#v", repositories)
	}
}

func TestCompleteCodeDeliveryConflictFinalizesMultipleRepositories(t *testing.T) {
	contexts := []codeDeliveryConflictContext{*createCodeDeliveryConflictFixture(t), *createCodeDeliveryConflictFixture(t)}
	for index := range contexts {
		contexts[index].Name = fmt.Sprintf("repository-%d", index)
		file, err := readCodeDeliveryConflictFile(&contexts[index], "conflict.txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := saveCodeDeliveryConflictFile(&contexts[index], file.Path, "content", fmt.Sprintf("resolved-%d\n", index), file.Version); err != nil {
			t.Fatal(err)
		}
	}
	for index := range contexts {
		commit, err := finalizeCodeDeliveryConflictCommit(&contexts[index])
		if err != nil {
			t.Fatalf("finalize repository %d: %v", index, err)
		}
		parents, err := runCodeGit(contexts[index].WorkDir, "show", "-s", "--format=%P", commit)
		if err != nil || !strings.Contains(parents, contexts[index].SourceCommit) || !strings.Contains(parents, contexts[index].TaskCommit) {
			t.Fatalf("repository %d merge parents are incomplete: %q, %v", index, parents, err)
		}
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
