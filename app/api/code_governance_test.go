package api

import (
	"errors"
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
		&model.AITimelineEvent{}, &model.AICodeDelivery{}, &model.AICodeDeliveryJob{}, &model.AICodeDeliveryLease{},
		&model.AICodeAuditEvent{}, &model.AIDevSessionRepository{}, &model.AIInstruction{},
		&model.HostTerminalSession{},
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
	if _, err := os.Stat(session.WorkDir); !os.IsNotExist(err) {
		t.Fatalf("delivered worktree remained after successful cleanup: %v", err)
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

func TestCodeDeliveryMergesLocallyAndPushesRemote(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	sourceDir, remoteDir := createCodeRemoteRepository(t)
	session := &model.AIDevSession{ID: 64, UserID: 7, ProjectID: 11, WorkDir: sourceDir}
	if err := database.Create(&model.AIProject{ID: 11, Name: "remote", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, &model.AIProject{SourceDirs: []string{sourceDir}}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "delivered.txt"), []byte("delivered\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "delivered.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := commitCodeSessionWorktree(session, "feat: deliver remote"); err != nil {
		t.Fatal(err)
	}
	result, err := resumeCodeSessionDelivery(session, session.UserID)
	if err != nil || result.Status != "merged" || result.Commit == "" {
		t.Fatalf("unexpected atomic delivery: %#v, %v", result, err)
	}
	branch, _ := runCodeGit(sourceDir, "branch", "--show-current")
	localHead, localErr := runCodeGit(sourceDir, "rev-parse", "refs/heads/"+branch)
	remoteHead, remoteErr := runCodeGit(remoteDir, "rev-parse", "refs/heads/"+branch)
	if localErr != nil || remoteErr != nil || localHead != result.Commit || remoteHead != result.Commit {
		t.Fatalf("delivery commits differ: local=%q remote=%q want=%q errors=%v/%v", localHead, remoteHead, result.Commit, localErr, remoteErr)
	}
	if _, err := os.Stat(session.WorkDir); !os.IsNotExist(err) {
		t.Fatalf("delivered worktree remained after remote verification: %v", err)
	}
	var delivery model.AICodeDelivery
	if err := database.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil || delivery.PushStatus != codePushPushed || delivery.PushedCommit != result.Commit {
		t.Fatalf("push state was not persisted: %#v, %v", delivery, err)
	}
}

func TestCodeDeliveryUsesCapturedCommitWhileTerminalContinues(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, sourceDir := createDeliveryWorktree(t, 67)
	session.ProjectID, session.Status = 14, codeSessionStatusActive
	if err := database.Create(&model.AIProject{ID: session.ProjectID, Name: "snapshot", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	firstCommit := commitCodeTestFile(t, session.WorkDir, "captured.txt", "captured\n")
	_, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	secondCommit := commitCodeTestFile(t, session.WorkDir, "later.txt", "later\n")
	if firstCommit == secondCommit {
		t.Fatal("terminal did not advance after delivery snapshot")
	}
	result, err := resumeCodeSessionDelivery(session, session.UserID)
	if err != nil || result.Status != "merged" {
		t.Fatalf("snapshot delivery failed: %#v, %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "captured.txt")); err != nil {
		t.Fatalf("captured commit was not delivered: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "later.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("later terminal commit leaked into snapshot delivery: %v", err)
	}
	if _, err := os.Stat(session.WorkDir); err != nil {
		t.Fatalf("active worktree was removed: %v", err)
	}
	if _, err := os.Stat(session.WorkDir); err != nil {
		t.Fatalf("worktree with later commit should be preserved: %v", err)
	}
	if err := completeCodeSessionLifecycle(database, session.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1"); err == nil {
		t.Fatal("delivered session should require a new task")
	}
}

func TestCodeDeliveryKeepsWorktreeWhenPushFails(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	sourceDir, remoteDir := createCodeRemoteRepository(t)
	session := &model.AIDevSession{ID: 65, UserID: 7, ProjectID: 12, WorkDir: sourceDir}
	if err := database.Create(&model.AIProject{ID: 12, Name: "push-failure", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, &model.AIProject{SourceDirs: []string{sourceDir}}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "kept.txt"), []byte("kept\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "kept.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := commitCodeSessionWorktree(session, "feat: keep failed delivery"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(remoteDir, "config", "receive.denyNonFastForwards", "true"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(remoteDir, "hooks", "pre-receive"), []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := resumeCodeSessionDelivery(session, session.UserID); err == nil {
		t.Fatal("rejected push should fail delivery")
	}
	if _, err := os.Stat(session.WorkDir); err != nil {
		t.Fatalf("failed delivery removed worktree: %v", err)
	}
	var delivery model.AICodeDelivery
	if err := database.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil || delivery.PushStatus != codePushFailed || delivery.Status != codeDeliveryMerged {
		t.Fatalf("failed delivery state unavailable: %#v, %v", delivery, err)
	}
}

func TestCodeDeliveryRetriesWhenRemoteAdvancesAfterLocalMerge(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	sourceDir, remoteDir := createCodeRemoteRepository(t)
	session := &model.AIDevSession{ID: 66, UserID: 7, ProjectID: 13, WorkDir: sourceDir}
	if err := database.Create(&model.AIProject{ID: 13, Name: "concurrent", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, &model.AIProject{SourceDirs: []string{sourceDir}}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "session.txt"), []byte("session\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "session.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := commitCodeSessionWorktree(session, "feat: concurrent session"); err != nil {
		t.Fatal(err)
	}
	delivery, err := loadOrCreateCodeDelivery(session, session.UserID)
	if err != nil {
		t.Fatal(err)
	}
	firstMerge, err := mergePreparedCodeDelivery(delivery)
	if err != nil || firstMerge.Commit == "" {
		t.Fatalf("local merge failed: %#v, %v", firstMerge, err)
	}
	updater := cloneCodeRepository(t, remoteDir)
	remoteCommit := commitCodeTestFile(t, updater, "remote.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	result, err := integrateAndPushCodeDelivery(delivery)
	if err != nil || result.Commit == "" || result.Commit == firstMerge.Commit {
		t.Fatalf("delivery was not rebuilt on latest remote: %#v, %v", result, err)
	}
	branch, _ := runCodeGit(sourceDir, "branch", "--show-current")
	localHead, _ := runCodeGit(sourceDir, "rev-parse", "refs/heads/"+branch)
	remoteHead, _ := runCodeGit(remoteDir, "rev-parse", "refs/heads/"+branch)
	if localHead != result.Commit || remoteHead != result.Commit {
		t.Fatalf("retried delivery differs: local=%q remote=%q want=%q", localHead, remoteHead, result.Commit)
	}
	if _, err := runCodeGit(sourceDir, "merge-base", "--is-ancestor", remoteCommit, result.Commit); err != nil {
		t.Fatalf("remote update missing from retried delivery: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "session.txt")); err != nil {
		t.Fatalf("session update missing from local project: %v", err)
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
	if _, err := os.Stat(session.WorkDir); !os.IsNotExist(err) {
		t.Fatalf("merged delivery worktree was not cleaned: %v", err)
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
