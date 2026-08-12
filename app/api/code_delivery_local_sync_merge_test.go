package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
)

func createCodeDeliverySyncCommit(t *testing.T, repository, baseCommit, file, content string) string {
	t.Helper()
	workDir := filepath.Join(t.TempDir(), "delivery")
	if _, err := runCodeGit(repository, "worktree", "add", "--detach", workDir, baseCommit); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = runCodeGit(repository, "worktree", "remove", "--force", workDir) })
	return commitCodeTestFile(t, workDir, file, content)
}

func TestSyncCodeDeliveryTargetOnDemandFastForwards(t *testing.T) {
	repository := createCodeGitRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	baseCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
	deliveryCommit := createCodeDeliverySyncCommit(t, repository, baseCommit, "delivery.txt", "delivery\n")

	if err := syncCodeDeliveryTargetOnDemand(repository, strings.TrimSpace(branch), deliveryCommit); err != nil {
		t.Fatal(err)
	}
	head, _ := runCodeGit(repository, "rev-parse", "HEAD")
	if strings.TrimSpace(head) != deliveryCommit {
		t.Fatalf("HEAD = %q, want delivery commit %q", head, deliveryCommit)
	}
}

func TestSyncCodeDeliveryTargetOnDemandUpdatesCheckedOutTargetWorktree(t *testing.T) {
	repository := createCodeGitRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	branch = strings.TrimSpace(branch)
	baseCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
	deliveryCommit := createCodeDeliverySyncCommit(t, repository, baseCommit, "delivery.txt", "delivery\n")
	secondary := filepath.Join(t.TempDir(), "target")
	if _, err := runCodeGit(repository, "switch", "--detach"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "worktree", "add", secondary, branch); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = runCodeGit(repository, "worktree", "remove", "--force", secondary) })

	if err := syncCodeDeliveryTargetOnDemand(repository, branch, deliveryCommit); err != nil {
		t.Fatal(err)
	}
	head, _ := runCodeGit(secondary, "rev-parse", "HEAD")
	if strings.TrimSpace(head) != deliveryCommit {
		t.Fatalf("checked out target worktree HEAD = %q, want %q", head, deliveryCommit)
	}
}

func TestSyncCodeDeliveryTargetOnDemandMergesAdvancedBranch(t *testing.T) {
	repository := createCodeGitRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	baseCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
	deliveryCommit := createCodeDeliverySyncCommit(t, repository, baseCommit, "delivery.txt", "delivery\n")
	localCommit := commitCodeTestFile(t, repository, "local.txt", "local\n")

	if err := syncCodeDeliveryTargetOnDemand(repository, strings.TrimSpace(branch), deliveryCommit); err != nil {
		t.Fatal(err)
	}
	head, _ := runCodeGit(repository, "rev-parse", "HEAD")
	for _, commit := range []string{deliveryCommit, localCommit} {
		if _, err := runCodeGit(repository, "merge-base", "--is-ancestor", commit, strings.TrimSpace(head)); err != nil {
			t.Fatalf("merged HEAD does not contain %s: %v", commit, err)
		}
	}
}

func TestSyncCodeDeliveryTargetOnDemandPreservesDirtyRepository(t *testing.T) {
	repository := createCodeGitRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	baseCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
	deliveryCommit := createCodeDeliverySyncCommit(t, repository, baseCommit, "delivery.txt", "delivery\n")
	if err := os.WriteFile(filepath.Join(repository, "README.md"), []byte("local edit\n"), 0600); err != nil {
		t.Fatal(err)
	}

	err := syncCodeDeliveryTargetOnDemand(repository, strings.TrimSpace(branch), deliveryCommit)
	if err == nil || !strings.Contains(err.Error(), "未提交改动") {
		t.Fatalf("dirty repository should block browser merge: %v", err)
	}
	content, readErr := os.ReadFile(filepath.Join(repository, "README.md"))
	if readErr != nil || string(content) != "local edit\n" {
		t.Fatalf("dirty file changed: %q, %v", content, readErr)
	}
	if conflicts := codeGitConflictFiles(repository); len(conflicts) != 0 {
		t.Fatalf("source repository retained conflicts: %#v", conflicts)
	}
}

func TestSyncCodeDeliveryTargetOnDemandKeepsConflictIsolated(t *testing.T) {
	repository := createCodeGitRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	baseCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
	deliveryCommit := createCodeDeliverySyncCommit(t, repository, baseCommit, "README.md", "delivery\n")
	localCommit := commitCodeTestFile(t, repository, "README.md", "local\n")

	err := syncCodeDeliveryTargetOnDemand(repository, strings.TrimSpace(branch), deliveryCommit)
	if err == nil || !strings.Contains(err.Error(), "存在冲突") {
		t.Fatalf("conflicting delivery should require fallback: %v", err)
	}
	head, _ := runCodeGit(repository, "rev-parse", "HEAD")
	if strings.TrimSpace(head) != localCommit {
		t.Fatalf("source HEAD changed after isolated conflict: %q", head)
	}
	if conflicts := codeGitConflictFiles(repository); len(conflicts) != 0 {
		t.Fatalf("source repository retained conflicts: %#v", conflicts)
	}
}

func TestCodeDeliveryLocalSyncCommandQuotesArguments(t *testing.T) {
	command := codeDeliveryLocalSyncCommand("/tmp/project's source", "feature/local sync", "abc123")
	for _, value := range []string{"'/tmp/project'\\''s source'", "'feature/local sync'", "'abc123'"} {
		if !strings.Contains(command, value) {
			t.Fatalf("command %q does not quote %q", command, value)
		}
	}
}

func TestSyncCodeSessionDeliveryLocalCompletesSingleRepository(t *testing.T) {
	database := withCodeGovernanceDB(t)
	repository := createCodeGitRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	baseCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
	deliveryCommit := createCodeDeliverySyncCommit(t, repository, baseCommit, "delivery.txt", "delivery\n")
	session := &model.AIDevSession{ID: 1201, UserID: 7}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AICodeDeliveryJob{
		SessionID: session.ID, UserID: session.UserID, Status: codeDeliveryJobCompleted, Stage: codeDeliveryStageCompleted,
	}).Error; err != nil {
		t.Fatal(err)
	}
	delivery := &model.AICodeDelivery{
		SessionID: session.ID, UserID: session.UserID, Status: codeDeliveryCompleted,
		SourceWorkDir: repository, TargetBranch: strings.TrimSpace(branch), MergeCommit: deliveryCommit,
	}
	if err := database.Create(delivery).Error; err != nil {
		t.Fatal(err)
	}

	result, err := syncCodeSessionDeliveryLocal(session)
	if err != nil || result.Status != "completed" || len(result.Repositories) != 1 || !result.Repositories[0].LocalSynced {
		t.Fatalf("unexpected local sync result: %#v, %v", result, err)
	}
	if err := database.First(delivery, delivery.ID).Error; err != nil || delivery.SourceAppliedAt == nil || delivery.LocalSyncError != "" {
		t.Fatalf("local sync state was not persisted: %#v, %v", delivery, err)
	}
}

func TestSyncCodeSessionDeliveryLocalConflictCanBeResolvedInBrowser(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	repository := createCodeGitRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	branch = strings.TrimSpace(branch)
	baseCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
	deliveryCommit := createCodeDeliverySyncCommit(t, repository, baseCommit, "README.md", "delivery\n")
	mainCommit := commitCodeTestFile(t, repository, "README.md", "main\n")
	session := &model.AIDevSession{ID: 1205, UserID: 7, ProjectID: 15}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	job := &model.AICodeDeliveryJob{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryJobCompleted, Stage: codeDeliveryStageCompleted,
	}
	if err := database.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	delivery := &model.AICodeDelivery{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryCompleted, SourceWorkDir: repository, TargetBranch: branch,
		WorktreeBranch: "task", WorktreeCommit: deliveryCommit, MergeCommit: deliveryCommit,
	}
	if err := database.Create(delivery).Error; err != nil {
		t.Fatal(err)
	}

	result, err := syncCodeSessionDeliveryLocal(session)
	if err != nil || result.Status != "conflict" || len(result.Repositories) != 1 ||
		len(result.Repositories[0].ConflictFiles) != 1 || result.Repositories[0].ConflictFiles[0] != "README.md" {
		t.Fatalf("unexpected conflict result: %#v, %v", result, err)
	}
	if err := database.First(job, job.ID).Error; err != nil || !isCodeDeliveryLocalSyncConflict(job) {
		t.Fatalf("local conflict job was not persisted: %#v, %v", job, err)
	}
	loadedJob, contexts, err := loadCodeDeliveryConflictContexts(session.ID)
	if err != nil || len(contexts) != 1 {
		t.Fatalf("browser conflict context unavailable: %#v, %v", contexts, err)
	}
	file, err := readCodeDeliveryConflictFile(&contexts[0], "README.md")
	if err != nil || file.MainContent != "main\n" || file.TaskContent != "delivery\n" {
		t.Fatalf("unexpected browser conflict file: %#v, %v", file, err)
	}
	if _, err := saveCodeDeliveryConflictFile(&contexts[0], file.Path, "content", "resolved\n", file.Version); err != nil {
		t.Fatal(err)
	}
	if err := completeCodeDeliveryConflict(loadedJob, contexts); err != nil {
		t.Fatal(err)
	}

	head, _ := runCodeGit(repository, "rev-parse", "refs/heads/"+branch)
	for _, commit := range []string{mainCommit, deliveryCommit} {
		if _, err := runCodeGit(repository, "merge-base", "--is-ancestor", commit, strings.TrimSpace(head)); err != nil {
			t.Fatalf("resolved main does not contain %s: %v", commit, err)
		}
	}
	content, readErr := os.ReadFile(filepath.Join(repository, "README.md"))
	if readErr != nil || string(content) != "resolved\n" {
		t.Fatalf("resolved main content = %q, %v", content, readErr)
	}
	if err := database.First(delivery, delivery.ID).Error; err != nil || delivery.SourceAppliedAt == nil ||
		delivery.LocalSyncError != "" || delivery.DeliveryWorkDir != "" {
		t.Fatalf("resolved delivery state is incomplete: %#v, %v", delivery, err)
	}
	if _, err := os.Stat(contexts[0].WorkDir); !os.IsNotExist(err) {
		t.Fatalf("conflict worktree was not cleaned: %v", err)
	}
	view, err := loadCodeDeliveryJobView(session.ID)
	if err != nil || view.Status != codeDeliveryJobCompleted || view.FailureCode != "" ||
		len(view.ConflictFiles) != 0 || len(view.Repositories) != 1 || !view.Repositories[0].LocalSynced {
		t.Fatalf("resolved delivery view is incomplete: %#v, %v", view, err)
	}
}

func TestSyncCodeMultiRepositoryLocalConflictPreservesCompletedResults(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	session := &model.AIDevSession{ID: 1206, UserID: 7, ProjectID: 16}
	if err := database.Create(&model.AIProject{ID: session.ProjectID, Name: "multi", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	job := &model.AICodeDeliveryJob{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryJobCompleted, Stage: codeDeliveryStageCompleted,
	}
	if err := database.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 2; index++ {
		repository := createCodeGitRepository(t)
		branch, _ := runCodeGit(repository, "branch", "--show-current")
		baseCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
		file, content := "delivery.txt", "delivery\n"
		if index == 1 {
			file, content = "README.md", "delivery-conflict\n"
		}
		deliveryCommit := createCodeDeliverySyncCommit(t, repository, baseCommit, file, content)
		if index == 1 {
			commitCodeTestFile(t, repository, "README.md", "main-conflict\n")
		}
		row := &model.AIDevSessionRepository{
			SessionID: session.ID, ProjectID: session.ProjectID, SourceDir: repository,
			WorktreeDir: filepath.Join(t.TempDir(), "removed"), LinkName: string(rune('a' + index)), Branch: "task",
			TargetBranch: strings.TrimSpace(branch), BaseCommit: strings.TrimSpace(baseCommit),
			WorktreeCommit: deliveryCommit, MergeCommit: deliveryCommit, Status: codeDeliveryCompleted,
		}
		if err := database.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	result, err := syncCodeSessionDeliveryLocal(session)
	if err != nil || result.Status != "conflict" || len(result.Repositories) != 2 ||
		!result.Repositories[0].LocalSynced || len(result.Repositories[1].ConflictFiles) != 1 {
		t.Fatalf("unexpected multi repository conflict: %#v, %v", result, err)
	}
	loadedJob, contexts, err := loadCodeDeliveryConflictContexts(session.ID)
	if err != nil || len(contexts) != 1 || contexts[0].RepositoryID != result.Repositories[1].RepositoryID {
		t.Fatalf("unexpected multi repository conflict context: %#v, %v", contexts, err)
	}
	file, err := readCodeDeliveryConflictFile(&contexts[0], "README.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeDeliveryConflictFile(&contexts[0], file.Path, "content", "resolved-multi\n", file.Version); err != nil {
		t.Fatal(err)
	}
	if err := completeCodeDeliveryConflict(loadedJob, contexts); err != nil {
		t.Fatal(err)
	}

	var repositories []model.AIDevSessionRepository
	if err := database.Where("session_id = ?", session.ID).Order("id asc").Find(&repositories).Error; err != nil {
		t.Fatal(err)
	}
	if len(repositories) != 2 || repositories[0].SourceAppliedAt == nil || repositories[1].SourceAppliedAt == nil {
		t.Fatalf("multi repository local sync state was lost: %#v", repositories)
	}
	view, err := loadCodeDeliveryJobView(session.ID)
	if err != nil || view.Status != codeDeliveryJobCompleted || len(view.Repositories) != 2 ||
		!view.Repositories[0].LocalSynced || !view.Repositories[1].LocalSynced {
		t.Fatalf("multi repository results were not preserved: %#v, %v", view, err)
	}
}

func TestSyncCodeSessionDeliveryLocalRejectsIncompleteJob(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{ID: 1202, UserID: 7}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AICodeDeliveryJob{
		SessionID: session.ID, UserID: session.UserID, Status: codeDeliveryJobRunning, Stage: codeDeliveryStageMerging,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := syncCodeSessionDeliveryLocal(session); err == nil || !strings.Contains(err.Error(), "尚未完成") {
		t.Fatalf("incomplete delivery job should be rejected: %v", err)
	}
}

func TestSyncCodeSessionDeliveryLocalReturnsPartialAndPersistsBlocker(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{ID: 1203, UserID: 7}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AICodeDeliveryJob{
		SessionID: session.ID, UserID: session.UserID, Status: codeDeliveryJobCompleted, Stage: codeDeliveryStageCompleted,
	}).Error; err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 2; index++ {
		repository := createCodeGitRepository(t)
		branch, _ := runCodeGit(repository, "branch", "--show-current")
		baseCommit, _ := runCodeGit(repository, "rev-parse", "HEAD")
		deliveryCommit := createCodeDeliverySyncCommit(t, repository, baseCommit, "delivery.txt", "delivery\n")
		if index == 1 {
			if err := os.WriteFile(filepath.Join(repository, "README.md"), []byte("dirty\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		row := &model.AIDevSessionRepository{
			SessionID: session.ID, SourceDir: repository, WorktreeDir: filepath.Join(t.TempDir(), "removed"),
			LinkName: string(rune('a' + index)), Branch: "task", TargetBranch: strings.TrimSpace(branch),
			BaseCommit: strings.TrimSpace(baseCommit), MergeCommit: deliveryCommit, Status: codeDeliveryCompleted,
		}
		if err := database.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	result, err := syncCodeSessionDeliveryLocal(session)
	if err != nil || result.Status != "partial" || len(result.Repositories) != 2 {
		t.Fatalf("unexpected partial result: %#v, %v", result, err)
	}
	var rows []model.AIDevSessionRepository
	if err := database.Where("session_id = ?", session.ID).Order("link_name asc").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if rows[0].SourceAppliedAt == nil || rows[1].SourceAppliedAt != nil || !strings.Contains(rows[1].LocalSyncError, "未提交改动") {
		t.Fatalf("partial state was not persisted: %#v", rows)
	}
}

func TestGetCodeDeliveryLocalSyncSessionWorksAfterWorkspaceCleanup(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 1204)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range repositories {
		if err := global.DB.Model(&repositories[index]).Update("status", codeDeliveryCompleted).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := cleanupCodeSessionRepositoryWorktreesWithMetadata(session, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(session.WorkDir); !os.IsNotExist(err) {
		t.Fatalf("session workspace was not cleaned: %v", err)
	}
	loaded, err := getCodeDeliveryLocalSyncSession(session.ID, &token.CustomClaims{UserId: session.UserID})
	if err != nil || loaded.ID != session.ID {
		t.Fatalf("completed delivery should remain available after cleanup: %#v, %v", loaded, err)
	}
}
