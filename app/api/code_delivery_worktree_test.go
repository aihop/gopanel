package api

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestCodeDeliveryConflictCanResumeFromIntegrationWorktree(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, sourceDir := createDeliveryWorktree(t, 68)
	session.ProjectID = 15
	if err := database.Create(&model.AIProject{ID: session.ProjectID, Name: "conflict", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "README.md"), []byte("session\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "feat: session conflict"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("source\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(sourceDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(sourceDir, "commit", "-m", "feat: source conflict"); err != nil {
		t.Fatal(err)
	}
	_, result, err := prepareCodeSessionDeliveryWithProgress(session, session.UserID, nil)
	if err != nil || result.Status != "conflict" || len(result.ConflictFiles) != 1 {
		t.Fatalf("unexpected conflict result: %#v, %v", result, err)
	}
	if len(result.Repositories) != 1 || result.Repositories[0].Branch != session.WorktreeBranch ||
		result.Repositories[0].TargetBranch != session.TargetBranch || result.Repositories[0].Status != "conflict" {
		t.Fatalf("conflict result did not preserve the local task branch: %#v", result.Repositories)
	}
	if _, err := runCodeGit(sourceDir, "show-ref", "--verify", "refs/heads/"+session.WorktreeBranch); err != nil {
		t.Fatalf("local task branch was not retained in the source repository: %v", err)
	}
	var delivery model.AICodeDelivery
	if err := database.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		t.Fatal(err)
	}
	conflicts := codeGitConflictFiles(delivery.DeliveryWorkDir)
	if delivery.DeliveryWorkDir == "" || len(conflicts) != 1 || conflicts[0] != "README.md" {
		t.Fatalf("conflict was not preserved in delivery worktree: %#v", delivery)
	}
	status, err := runCodeGit(sourceDir, "status", "--porcelain")
	if err != nil || status != "" {
		t.Fatalf("source repository entered conflict state: %q, %v", status, err)
	}
	if err := os.WriteFile(filepath.Join(delivery.DeliveryWorkDir, "README.md"), []byte("resolved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(delivery.DeliveryWorkDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(
		delivery.DeliveryWorkDir, "-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.cn",
		"-c", "commit.gpgsign=false", "commit", "-m", "fix: resolve delivery conflict",
	); err != nil {
		t.Fatal(err)
	}
	result, err = resumeCodeSessionDelivery(session, session.UserID)
	if err != nil || result.Status != "merged" {
		t.Fatalf("resolved delivery did not resume: %#v, %v", result, err)
	}
	content, err := os.ReadFile(filepath.Join(sourceDir, "README.md"))
	if err != nil || string(content) != "resolved\n" {
		t.Fatalf("resolved content was not delivered: %q, %v", content, err)
	}
}

func TestCodeDeliveryConflictRecognizesManualSourceMerge(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, sourceDir := createDeliveryWorktree(t, 72)
	session.ProjectID = 16
	if err := database.Create(&model.AIProject{ID: session.ProjectID, Name: "manual-conflict", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "README.md"), []byte("session\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "feat: session conflict"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("source\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(sourceDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(sourceDir, "commit", "-m", "feat: source conflict"); err != nil {
		t.Fatal(err)
	}
	if _, result, err := prepareCodeSessionDeliveryWithProgress(session, session.UserID, nil); err != nil || result.Status != "conflict" {
		t.Fatalf("unexpected conflict result: %#v, %v", result, err)
	}
	if _, err := runCodeGit(sourceDir, "merge", "--no-ff", "--no-edit", session.WorktreeBranch); err == nil {
		t.Fatal("manual source merge should require conflict resolution")
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("manual resolution\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(sourceDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(sourceDir, "commit", "--no-edit"); err != nil {
		t.Fatal(err)
	}
	result, err := resumeCodeSessionDelivery(session, session.UserID)
	if err != nil || result.Status != "merged" {
		t.Fatalf("manual source merge was not recognized: %#v, %v", result, err)
	}
	if result.Commit == "" {
		t.Fatalf("manual source merge did not produce a delivery commit: %#v", result)
	}
	if conflicts := codeGitConflictFiles(sourceDir); len(conflicts) != 0 {
		t.Fatalf("source repository retained conflicts after manual resolution: %#v", conflicts)
	}
}

func TestCleanupCodeDeliveryWorktreesWhenSourceDirectoryIsMissing(t *testing.T) {
	session, sourceDir := createDeliveryWorktree(t, 69)
	deliveryDir := aiSessionDeliveryWorktreeDir(session.UserID, session.ID)
	if err := os.MkdirAll(deliveryDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deliveryDir, "snapshot"), []byte("ready\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(sourceDir); err != nil {
		t.Fatal(err)
	}
	delivery := &model.AICodeDelivery{
		SessionID: session.ID, UserID: session.UserID, Status: codeDeliveryCompleted,
		SourceWorkDir: sourceDir, WorkDir: session.WorkDir, WorktreeBranch: session.WorktreeBranch,
		DeliveryWorkDir: deliveryDir,
	}
	if err := cleanupCodeDeliveryWorktree(delivery); err != nil {
		t.Fatal(err)
	}
	for _, workDir := range []string{deliveryDir, session.WorkDir} {
		if _, err := os.Stat(workDir); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing source cleanup retained %s: %v", workDir, err)
		}
	}
}

func TestCleanupCodeDeliveryWorktreesWhenSourceIsNoLongerGitRepository(t *testing.T) {
	session, sourceDir := createDeliveryWorktree(t, 70)
	deliveryDir := aiSessionDeliveryWorktreeDir(session.UserID, session.ID)
	if err := os.MkdirAll(deliveryDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(sourceDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sourceDir, 0750); err != nil {
		t.Fatal(err)
	}
	delivery := &model.AICodeDelivery{
		SessionID: session.ID, UserID: session.UserID, Status: codeDeliveryCompleted,
		SourceWorkDir: sourceDir, WorkDir: session.WorkDir, WorktreeBranch: session.WorktreeBranch,
		DeliveryWorkDir: deliveryDir,
	}
	if err := cleanupCodeDeliveryWorktree(delivery); err != nil {
		t.Fatal(err)
	}
	for _, workDir := range []string{deliveryDir, session.WorkDir} {
		if _, err := os.Stat(workDir); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale cleanup retained %s: %v", workDir, err)
		}
	}
}

func TestCleanupCodeDeliveryWorktreesWhenManagedDirectoriesAreAlreadyMissing(t *testing.T) {
	withAIProjectBaseDir(t)
	delivery := &model.AICodeDelivery{
		SessionID: 71, UserID: 7, Status: codeDeliveryCompleted,
		SourceWorkDir: filepath.Join(t.TempDir(), "missing-source"),
		WorkDir: aiSessionWorktreeDir(7, 71), WorktreeBranch: "gopanel/code-71-stale",
		DeliveryWorkDir: aiSessionDeliveryWorktreeDir(7, 71),
	}
	if err := cleanupCodeDeliveryWorktree(delivery); err != nil {
		t.Fatalf("already removed worktrees should be treated as cleaned: %v", err)
	}
}
