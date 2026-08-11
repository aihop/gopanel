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

// 回归：本地快进失败被刻意降级为非阻断，仓库仍会标 completed。
// 此时「本地主仓」这条事实必须显示未完成，否则界面会谎报「已交付主仓」。
func TestCodeMultiRepositoryDeliveryFactsDoNotTreatCompletedAsLocallyApplied(t *testing.T) {
	now := time.Now()
	facts := codeMultiRepositoryDeliveryFacts([]codeRepositoryDeliveryResult{
		{
			Status: codeDeliveryCompleted, Commit: "landed", SnapshotReady: true, MergeReady: true,
			SourceAppliedAt: &now, LocalSynced: true, PushStatus: codePushPushed,
			Remote: "origin", RemoteBranch: "main",
		},
		{
			// 交付提交产出了、仓库也标了 completed，但快进失败：SourceAppliedAt 为空。
			Status: codeDeliveryCompleted, Commit: "not-landed", SnapshotReady: true, MergeReady: true,
			LocalSyncError: "源仓库 qingpu-ai 无法安全快进", PushStatus: codePushPushed,
			Remote: "origin", RemoteBranch: "main",
		},
	})
	local := codeDeliveryFactByKey(facts, "local")
	if local.Count != 1 || local.Total != 2 || local.Status != "partial" {
		t.Fatalf("本地主仓事实应为 1/2 partial，实际 %#v", local)
	}
	// 交付提交本身是产出了的，merge 这一层不受影响。
	if merge := codeDeliveryFactByKey(facts, "merge"); merge.Status != "completed" {
		t.Fatalf("merge 事实不应受本地同步失败影响: %#v", merge)
	}
}

func codeDeliveryFactByKey(facts []codeDeliveryFact, key string) codeDeliveryFact {
	for _, fact := range facts {
		if fact.Key == key {
			return fact
		}
	}
	return codeDeliveryFact{}
}
