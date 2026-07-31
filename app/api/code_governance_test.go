package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func withCodeGovernanceDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "governance.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&model.AIProject{}, &model.AIDevSession{}, &model.AITask{}, &model.AIExecutionRun{},
		&model.AITimelineEvent{}, &model.AICodeDelivery{}, &model.AICodeAuditEvent{}, &model.AIDevSessionRepository{},
	); err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = oldDB })
	return database
}

func TestCodeDeliveryResumesAfterWorktreeCleanup(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, sourceDir := createDeliveryWorktree(t, 61)
	session.ProjectID = 8
	if err := database.Create(&model.AIProject{ID: 8, Name: "project", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: session.UserID, SessionID: session.ID, ProjectID: session.ProjectID, Title: "task", WorkDir: session.WorkDir}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	commit, err := runCodeGit(session.WorkDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	delivery := &model.AICodeDelivery{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryWorktreeCleaned, SourceWorkDir: sourceDir, WorkDir: session.WorkDir,
		WorktreeBranch: session.WorktreeBranch, WorktreeCommit: commit, MergeCommit: commit,
	}
	if err := database.Create(delivery).Error; err != nil {
		t.Fatal(err)
	}
	result, err := resumeCodeSessionDelivery(session, session.UserID)
	if err != nil || result.Commit != commit {
		t.Fatalf("unexpected recovery result: %#v, %v", result, err)
	}
	var storedSession model.AIDevSession
	if err := database.First(&storedSession, session.ID).Error; err != nil || storedSession.WorkDir != sourceDir || storedSession.WorktreeBranch != "" {
		t.Fatalf("session metadata not recovered: %#v, %v", storedSession, err)
	}
	var storedTask model.AITask
	if err := database.First(&storedTask, task.ID).Error; err != nil || storedTask.WorkDir != sourceDir {
		t.Fatalf("task metadata not recovered: %#v, %v", storedTask, err)
	}
}

func TestCodeDeliveryCompletesAndIsIdempotent(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, sourceDir := createDeliveryWorktree(t, 62)
	session.ProjectID = 9
	if err := database.Create(&model.AIProject{ID: 9, Name: "project", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: session.UserID, SessionID: session.ID, ProjectID: session.ProjectID, Title: "task", WorkDir: session.WorkDir}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "delivery.txt"), []byte("complete\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "delivery.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := commitCodeSessionWorktree(session, "feat: complete delivery"); err != nil {
		t.Fatal(err)
	}

	first, err := resumeCodeSessionDelivery(session, session.UserID)
	if err != nil || first.Status != "merged" || first.Commit == "" {
		t.Fatalf("unexpected delivery result: %#v, %v", first, err)
	}
	if content, err := os.ReadFile(filepath.Join(sourceDir, "delivery.txt")); err != nil || string(content) != "complete\n" {
		t.Fatalf("merged content unavailable: %q, %v", content, err)
	}
	if _, err := os.Stat(session.WorkDir); err != nil {
		t.Fatalf("delivery removed a potentially active worktree: %v", err)
	}
	if err := cleanupDeliveredCodeSessionWorktrees(session); err != nil {
		t.Fatalf("delivered worktree cleanup failed after execution stopped: %v", err)
	}
	if _, err := os.Stat(session.WorkDir); !os.IsNotExist(err) {
		t.Fatalf("delivered worktree remained after deferred cleanup: %v", err)
	}
	var delivery model.AICodeDelivery
	if err := database.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil || delivery.Status != codeDeliveryCompleted || delivery.CompletedAt == nil {
		t.Fatalf("delivery was not completed: %#v, %v", delivery, err)
	}
	second, err := resumeCodeSessionDelivery(session, session.UserID)
	if err != nil || second.Commit != first.Commit {
		t.Fatalf("repeated delivery was not idempotent: %#v, %v", second, err)
	}
}

func TestCodeDeliveryResumesFromMergedState(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, sourceDir := createDeliveryWorktree(t, 63)
	session.ProjectID = 10
	if err := database.Create(&model.AIProject{ID: 10, Name: "project", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: session.UserID, SessionID: session.ID, ProjectID: session.ProjectID, Title: "task", WorkDir: session.WorkDir}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "merged.txt"), []byte("merged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "merged.txt"); err != nil {
		t.Fatal(err)
	}
	committed, err := commitCodeSessionWorktree(session, "feat: merged recovery")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(sourceDir, "merge", "--no-ff", "--no-edit", session.WorktreeBranch); err != nil {
		t.Fatal(err)
	}
	mergeCommit, err := runCodeGit(sourceDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	delivery := &model.AICodeDelivery{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryMerged, SourceWorkDir: sourceDir, WorkDir: session.WorkDir,
		WorktreeBranch: session.WorktreeBranch, WorktreeCommit: committed.Commit,
		MergeCommit: mergeCommit, MergedAt: &now,
	}
	if err := database.Create(delivery).Error; err != nil {
		t.Fatal(err)
	}
	result, err := resumeCodeSessionDelivery(session, session.UserID)
	if err != nil || result.Commit != mergeCommit {
		t.Fatalf("unexpected merged recovery: %#v, %v", result, err)
	}
	if _, err := os.Stat(session.WorkDir); err != nil {
		t.Fatalf("merged delivery removed a potentially active worktree: %v", err)
	}
	if err := database.First(delivery, delivery.ID).Error; err != nil || delivery.Status != codeDeliveryCompleted {
		t.Fatalf("delivery did not complete: %#v, %v", delivery, err)
	}
}

func TestCodeQualityGateRejectsStaleRevision(t *testing.T) {
	database := withCodeGovernanceDB(t)
	workDir := createCodeGitRepository(t)
	project := &model.AIProject{Name: "quality", CreatorID: 1, RequireQualityGate: true}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{UserID: 1, ProjectID: project.ID, Title: "session", WorkDir: workDir}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "package.json"), []byte(`{"scripts":{"test":"true"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(workDir, "add", "package.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(workDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "add quality check"); err != nil {
		t.Fatal(err)
	}
	checks, err := detectCodeQualityChecks(session)
	if err != nil || len(checks) != 1 {
		t.Fatalf("unexpected quality checks: %#v, %v", checks, err)
	}
	revision, err := codeQualityRevision(workDir)
	if err != nil {
		t.Fatal(err)
	}
	result := codeQualityCheckResult{CheckID: checks[0].ID, Status: "passed", Revision: revision, Current: true, StartedAt: time.Now(), CompletedAt: time.Now()}
	if err := persistCodeQualityResult(session, session.UserID, checks[0], result); err != nil {
		t.Fatal(err)
	}
	if err := validateCodeQualityGate(session); err != nil {
		t.Fatalf("current passing revision should satisfy gate: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(workDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(workDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "change revision"); err != nil {
		t.Fatal(err)
	}
	if err := validateCodeQualityGate(session); err == nil || !strings.Contains(err.Error(), "已过期") {
		t.Fatalf("stale quality result should be rejected: %v", err)
	}
}

func TestCodeAuditPersistsSafeMetadata(t *testing.T) {
	database := withCodeGovernanceDB(t)
	startedAt := time.Now().Add(-10 * time.Millisecond)
	recordCodeAudit(3, 4, 5, "database_query", "success", "reporting", "done", "127.0.0.1", startedAt, codeAuditMeta{
		"sqlFingerprint": codeDatabaseSQLFingerprint(" SELECT  *  FROM users "),
	})
	var event model.AICodeAuditEvent
	if err := database.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.UserID != 3 || event.ProjectID != 4 || event.SessionID != 5 || event.DurationMS < 0 {
		t.Fatalf("unexpected audit event: %#v", event)
	}
	if strings.Contains(event.Meta, "SELECT") || strings.Contains(event.Meta, "users") {
		t.Fatalf("audit metadata leaked SQL: %s", event.Meta)
	}
	first := codeDatabaseSQLFingerprint(" SELECT  *  FROM users ")
	second := codeDatabaseSQLFingerprint("select * from USERS")
	if first != second || len(first) != 64 {
		t.Fatalf("SQL fingerprint is not stable: %q, %q", first, second)
	}
}

func TestCodeTokenBudgetBlocksExceededProject(t *testing.T) {
	database := withCodeGovernanceDB(t)
	project := &model.AIProject{Name: "budget", CreatorID: 1, MonthlyTokenBudget: 100}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{UserID: 1, ProjectID: project.ID, Title: "session", WorkDir: t.TempDir()}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	run := &model.AIExecutionRun{SessionID: session.ID, ExecutorID: "codex", Prompt: "run", Status: "completed", StartedAt: time.Now(), TotalTokens: 100}
	if err := database.Create(run).Error; err != nil {
		t.Fatal(err)
	}
	if err := validateCodeTokenBudget(session); err == nil {
		t.Fatal("exceeded project budget should block execution")
	}
	budget, err := loadCodeTokenBudget(project.ID, time.Now())
	if err != nil || !budget.Exceeded || budget.RemainingTokens != 0 {
		t.Fatalf("unexpected budget: %#v, %v", budget, err)
	}
}

func TestCodeDatabaseRateWindowExpiresEvents(t *testing.T) {
	now := time.Now()
	window := &codeRateWindow{events: make(map[string][]time.Time), now: func() time.Time { return now }}
	if !window.allow("user:1", 1) || window.allow("user:1", 1) {
		t.Fatal("rate window should reject the second event")
	}
	now = now.Add(codeDatabaseRateWindow + time.Second)
	if !window.allow("user:1", 1) {
		t.Fatal("expired rate event should not block the next query")
	}
}
