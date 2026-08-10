package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestCodeMultiRepositoryDeliveryFactsExposePartialProgress(t *testing.T) {
	now := time.Now()
	facts := codeMultiRepositoryDeliveryFacts([]codeRepositoryDeliveryResult{
		{Status: codeDeliveryCompleted, Commit: "first", SnapshotReady: true, MergeReady: true, PushStatus: codePushPushed, SourceAppliedAt: &now, Remote: "origin", RemoteBranch: "main"},
		{Status: codeDeliveryMerged, Commit: "second", SnapshotReady: true, MergeReady: true, PushStatus: codePushPending, Remote: "origin", RemoteBranch: "main"},
	})
	if len(facts) != 4 || facts[0].Status != "completed" || facts[1].Status != "completed" ||
		facts[2].Status != "partial" || facts[3].Status != "partial" {
		t.Fatalf("unexpected delivery facts: %#v", facts)
	}
}

func TestLoadCodeDeliveryFactsDistinguishesIsolatedMergeFromLocalMain(t *testing.T) {
	database := withCodeGovernanceDB(t)
	sourceDir := createCodeGitRepository(t)
	baseCommit, err := runCodeGit(sourceDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	deliveryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(deliveryDir, "merged.txt"), []byte("merged"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(deliveryDir, "add", "merged.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(deliveryDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "merge result"); err != nil {
		t.Fatal(err)
	}
	mergeCommit, err := runCodeGit(deliveryDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	delivery := &model.AICodeDelivery{
		SessionID: 992, ProjectID: 1, UserID: 1, Status: codeDeliveryMerged,
		SourceWorkDir: sourceDir, WorkDir: deliveryDir, DeliveryWorkDir: deliveryDir,
		WorktreeBranch: "session", TargetBranch: "main", WorktreeCommit: mergeCommit,
		MergeCommit: mergeCommit, RemoteCommit: baseCommit, PushStatus: codePushPending,
	}
	if err := database.Create(delivery).Error; err != nil {
		t.Fatal(err)
	}
	global.DB = database
	facts := loadCodeDeliveryFacts(delivery.SessionID, nil)
	if len(facts) != 4 || facts[0].Status != "completed" || facts[1].Status != "completed" ||
		facts[2].Status != "pending" || facts[3].Status != "skipped" {
		t.Fatalf("unexpected single-repository facts: %#v", facts)
	}
}
