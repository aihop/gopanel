package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func createCodeDeliveryQueueJob(t *testing.T, sessionID uint, repositoryKeys ...string) *model.AICodeDeliveryJob {
	t.Helper()
	encoded, err := json.Marshal(repositoryKeys)
	if err != nil {
		t.Fatal(err)
	}
	job := &model.AICodeDeliveryJob{
		SessionID: sessionID, ProjectID: 1, UserID: 1, Status: codeDeliveryJobQueued,
		Stage: codeDeliveryStageQueued, RepositoryKeys: string(encoded),
	}
	if err := global.DB.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	return job
}

func TestCodeDeliveryClaimIsExclusive(t *testing.T) {
	withCodeGovernanceDB(t)
	job := createCodeDeliveryQueueJob(t, 901, "repository-a")
	first := &codeDeliveryRunner{queued: make(map[uint]struct{}), owner: "runner-a"}
	second := &codeDeliveryRunner{queued: make(map[uint]struct{}), owner: "runner-b"}
	claimedJob, _, claimed, err := first.claim(job.ID)
	if err != nil || !claimed || claimedJob.LeaseOwner != first.owner {
		t.Fatalf("first claim failed: %#v, %v, %v", claimedJob, claimed, err)
	}
	if _, _, claimed, err := second.claim(job.ID); err != nil || claimed {
		t.Fatalf("active lease was claimed twice: %v, %v", claimed, err)
	}
}

func TestCodeDeliveryEnqueueIsIdempotent(t *testing.T) {
	database := withCodeGovernanceDB(t)
	sourceDir := t.TempDir()
	session := &model.AIDevSession{ID: 907, UserID: 1, ProjectID: 1, SourceWorkDir: sourceDir, TargetBranch: "main"}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	first, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.Model(&model.AICodeDeliveryJob{}).Where("session_id = ?", session.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || count != 1 {
		t.Fatalf("duplicate enqueue created multiple jobs: %d, %d, %d", first.ID, second.ID, count)
	}
}

func TestCodeDeliveryClaimRecoversExpiredLease(t *testing.T) {
	database := withCodeGovernanceDB(t)
	job := createCodeDeliveryQueueJob(t, 902, "repository-a")
	expired := time.Now().Add(-time.Minute)
	if err := database.Model(job).Updates(map[string]any{
		"status": codeDeliveryJobRunning, "lease_owner": "old-runner", "lease_expires_at": expired,
	}).Error; err != nil {
		t.Fatal(err)
	}
	runner := &codeDeliveryRunner{queued: make(map[uint]struct{}), owner: "new-runner"}
	recovered, _, claimed, err := runner.claim(job.ID)
	if err != nil || !claimed || recovered.LeaseOwner != runner.owner {
		t.Fatalf("expired job was not recovered: %#v, %v, %v", recovered, claimed, err)
	}
}

func TestCodeDeliveryRepositoryLeaseSerializesJobs(t *testing.T) {
	withCodeGovernanceDB(t)
	firstJob := createCodeDeliveryQueueJob(t, 903, "shared-repository")
	secondJob := createCodeDeliveryQueueJob(t, 904, "shared-repository")
	first := &codeDeliveryRunner{queued: make(map[uint]struct{}), owner: "runner-a"}
	second := &codeDeliveryRunner{queued: make(map[uint]struct{}), owner: "runner-b"}
	acquired, err := first.acquireRepositoryLeases(firstJob, []string{"shared-repository"})
	if err != nil || !acquired {
		t.Fatalf("first repository lease failed: %v, %v", acquired, err)
	}
	acquired, err = second.acquireRepositoryLeases(secondJob, []string{"shared-repository"})
	if err != nil || acquired {
		t.Fatalf("repository lease was not exclusive: %v, %v", acquired, err)
	}
	if err := global.DB.Model(&model.AICodeDeliveryLease{}).Where("job_id = ?", firstJob.ID).
		Update("lease_expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	acquired, err = second.acquireRepositoryLeases(secondJob, []string{"shared-repository"})
	if err != nil || !acquired {
		t.Fatalf("expired repository lease was not recovered: %v, %v", acquired, err)
	}
}

func TestCodeTaskSummaryIncludesDeliveryProgress(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{ID: 905, UserID: 1, ProjectID: 1, WorkDir: t.TempDir()}
	task := &model.AITask{ID: 906, UserID: 1, ProjectID: 1, SessionID: session.ID, Title: "delivery"}
	job := createCodeDeliveryQueueJob(t, session.ID, "repository-a")
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(job).Updates(map[string]any{
		"status": codeDeliveryJobRunning, "stage": codeDeliveryStagePushing, "progress": 70, "attempt": 2,
	}).Error; err != nil {
		t.Fatal(err)
	}
	items, err := buildCodeTaskListItems([]*model.AITask{task}, false)
	if err != nil || len(items) != 1 {
		t.Fatalf("task summary failed: %#v, %v", items, err)
	}
	summary := items[0].Summary
	if summary.DeliveryStatus != codeDeliveryJobRunning || summary.DeliveryStage != codeDeliveryStagePushing || summary.DeliveryProgress != 70 || summary.DeliveryAttempt != 2 {
		t.Fatalf("delivery summary mismatch: %#v", summary)
	}
}
