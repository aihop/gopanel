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
	result, err := resumeCodeSessionDelivery(session, session.UserID)
	if err != nil || result.Status != "conflict" || len(result.ConflictFiles) != 1 {
		t.Fatalf("unexpected conflict result: %#v, %v", result, err)
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
		delivery.DeliveryWorkDir, "-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.local",
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
