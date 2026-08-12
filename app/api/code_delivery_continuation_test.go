package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestCodeDeliveryPreservesActiveTerminalWorktree(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, sourceDir := createDeliveryWorktree(t, 68)
	session.ProjectID, session.Status = 15, codeSessionStatusActive
	if err := database.Create(&model.AIProject{ID: session.ProjectID, Name: "terminal", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	commitCodeTestFile(t, session.WorkDir, "terminal.txt", "terminal\n")
	previousCoordinator := codeExecutions
	codeExecutions = newCodeExecutionCoordinator(2, 2)
	t.Cleanup(func() { codeExecutions = previousCoordinator })
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionInteractive, false)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	if _, err := resumeCodeSessionDelivery(session, session.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(session.WorkDir); err != nil {
		t.Fatalf("delivery removed an active terminal worktree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "terminal.txt")); err != nil {
		t.Fatalf("active terminal snapshot was not delivered: %v", err)
	}
}

func TestCodeDeliveryJobViewDistinguishesPendingCommitsAndUncommittedChanges(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, _ := createDeliveryWorktree(t, 916)
	session.ProjectID, session.Status = 16, codeSessionStatusActive
	if err := database.Create(&model.AIProject{ID: session.ProjectID, Name: "pending", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	snapshotCommit, err := runCodeGit(session.WorkDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AICodeDelivery{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryCompleted, WorkDir: session.WorkDir, WorktreeCommit: snapshotCommit,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AICodeDeliveryJob{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryJobCompleted, Stage: codeDeliveryStageCompleted,
	}).Error; err != nil {
		t.Fatal(err)
	}

	view, err := loadCodeDeliveryJobView(session.ID)
	if err != nil || view.HasPendingChanges || view.HasPendingCommits || view.HasUncommittedChanges {
		t.Fatalf("clean snapshot reported pending work: %#v, %v", view, err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "pending.txt"), []byte("pending\n"), 0600); err != nil {
		t.Fatal(err)
	}
	view, err = loadCodeDeliveryJobView(session.ID)
	if err != nil || !view.HasPendingChanges || view.HasPendingCommits || !view.HasUncommittedChanges {
		t.Fatalf("uncommitted work was not distinguished: %#v, %v", view, err)
	}
	commitCodeTestFile(t, session.WorkDir, "pending.txt", "pending\n")
	view, err = loadCodeDeliveryJobView(session.ID)
	if err != nil || !view.HasPendingChanges || !view.HasPendingCommits || view.HasUncommittedChanges {
		t.Fatalf("pending commit was not distinguished: %#v, %v", view, err)
	}
}
