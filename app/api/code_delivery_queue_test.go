package api

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync/atomic"
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
	session, _ := createDeliveryWorktree(t, 907)
	session.ProjectID = 1
	if err := database.Create(&model.AIProject{ID: session.ProjectID, Name: "queue", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
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
	var storedSession model.AIDevSession
	if err := database.First(&storedSession, session.ID).Error; err != nil || storedSession.Status != codeSessionStatusDelivering {
		t.Fatalf("session was not sealed for delivery: %#v, %v", storedSession, err)
	}
}

func TestCodeDeliverySnapshotsWithInteractiveTerminalWithoutCancellingIt(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, _ := createDeliveryWorktree(t, 914)
	session.ProjectID, session.Status = 1, codeSessionStatusActive
	if err := database.Create(&model.AIProject{ID: session.ProjectID, Name: "interactive", CreatorID: session.UserID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	previousCoordinator := codeExecutions
	codeExecutions = newCodeExecutionCoordinator(2)
	t.Cleanup(func() { codeExecutions = previousCoordinator })
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionInteractive, false)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	var cancelled atomic.Bool
	lease.SetCancel(func() { cancelled.Store(true) })
	job, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1")
	if err != nil || job.Status != codeDeliveryJobQueued {
		t.Fatalf("delivery should snapshot with an active interactive terminal: %#v, %v", job, err)
	}
	if cancelled.Load() {
		t.Fatal("delivery cancelled the interactive terminal")
	}
}

func TestCodeDeliveryLifecycleCompletesAndReopens(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{ID: 908, UserID: 1, Status: codeSessionStatusDelivering, WorkDir: t.TempDir()}
	task := &model.AITask{ID: 909, UserID: 1, SessionID: session.ID, Title: "delivery", WorkDir: session.WorkDir, Status: "running"}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := completeCodeSessionLifecycle(database, session.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := database.First(session, session.ID).Error; err != nil || session.Status != codeSessionStatusActive || session.DeliveredAt == nil {
		t.Fatalf("session was not completed: %#v, %v", session, err)
	}
	if err := validateCodeSessionDevelopmentOpen(session); err != nil {
		t.Fatalf("delivered snapshot should keep session open: %v", err)
	}
	if err := database.Model(session).Updates(map[string]any{"status": codeSessionStatusDelivering, "delivered_at": nil}).Error; err != nil {
		t.Fatal(err)
	}
	if err := reopenCodeSessionLifecycle(database, session.ID); err != nil {
		t.Fatal(err)
	}
	if err := database.First(session, session.ID).Error; err != nil || session.Status != codeSessionStatusActive {
		t.Fatalf("failed delivery did not reopen session: %#v, %v", session, err)
	}
}

func TestCodeDeliveredSessionRejectsWorkspaceMutation(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{
		ID: 912, UserID: 1, Status: codeSessionStatusDelivered, WorkDir: t.TempDir(),
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	called := false
	err := runCodeSessionWorkspaceMutation(session, func(*model.AIDevSession) error {
		called = true
		return nil
	})
	if err == nil || called {
		t.Fatalf("delivered mutation was not rejected: called=%v err=%v", called, err)
	}
}

func TestCodeWorkspaceMutationCompletesBeforeDeliverySeal(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, _ := createDeliveryWorktree(t, 913)
	session.Status = codeSessionStatusActive
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	mutationStarted := make(chan struct{})
	releaseMutation := make(chan struct{})
	mutationDone := make(chan error, 1)
	go func() {
		mutationDone <- runCodeSessionWorkspaceMutation(session, func(*model.AIDevSession) error {
			close(mutationStarted)
			<-releaseMutation
			return nil
		})
	}()
	select {
	case <-mutationStarted:
	case <-time.After(time.Second):
		t.Fatal("workspace mutation did not start")
	}
	deliveryDone := make(chan error, 1)
	go func() {
		_, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1")
		deliveryDone <- err
	}()
	select {
	case err := <-deliveryDone:
		t.Fatalf("delivery sealed before mutation completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseMutation)
	if err := <-mutationDone; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-deliveryDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("delivery did not resume after mutation")
	}
	if err := database.First(session, session.ID).Error; err != nil || session.Status != codeSessionStatusDelivering {
		t.Fatalf("session was not sealed after mutation: %#v, %v", session, err)
	}
}

func TestCodeDeliveryRecoveryRestoresSessionLifecycle(t *testing.T) {
	database := withCodeGovernanceDB(t)
	queuedSession := &model.AIDevSession{ID: 910, UserID: 1, Status: codeSessionStatusActive, WorkDir: t.TempDir()}
	completedSession := &model.AIDevSession{ID: 911, UserID: 1, Status: codeSessionStatusActive, WorkDir: t.TempDir()}
	for _, session := range []*model.AIDevSession{queuedSession, completedSession} {
		if err := database.Create(session).Error; err != nil {
			t.Fatal(err)
		}
	}
	queued := createCodeDeliveryQueueJob(t, queuedSession.ID, "queued-repository")
	completed := createCodeDeliveryQueueJob(t, completedSession.ID, "completed-repository")
	completedAt := time.Now()
	if err := database.Model(completed).Updates(map[string]any{
		"status": codeDeliveryJobCompleted, "completed_at": completedAt,
	}).Error; err != nil {
		t.Fatal(err)
	}
	restoreCodeDeliverySessionLifecycles()
	if err := database.First(queuedSession, queuedSession.ID).Error; err != nil || queuedSession.Status != codeSessionStatusDelivering {
		t.Fatalf("queued session lifecycle was not restored: %#v, %v", queuedSession, err)
	}
	if err := database.First(completedSession, completedSession.ID).Error; err != nil || completedSession.Status != codeSessionStatusActive || completedSession.DeliveredAt == nil {
		t.Fatalf("completed session lifecycle was not restored: %#v, %v", completedSession, err)
	}
	if queued.Status != codeDeliveryJobQueued {
		t.Fatalf("unexpected queued job mutation: %#v", queued)
	}
}

func TestCodeMultiRepositoryResultType(t *testing.T) {
	results := []codeRepositoryDeliveryResult{
		{Status: codeDeliveryCompleted, PushStatus: codePushPushed},
		{Status: codeDeliveryMerged, PushStatus: "local"},
	}
	if resultType := codeMultiRepositoryResultType(results); resultType != "mixed" {
		t.Fatalf("unexpected mixed result type: %s", resultType)
	}
	partial := codeMultiRepositoryFailure(results[:1], errors.New("second repository failed"))
	if partial.Status != codeDeliveryJobPartial || partial.ResultType != "remote_verified" {
		t.Fatalf("unexpected partial result: %#v", partial)
	}
}

func TestCodeStoredRepositoryDeliveryResultDoesNotNeedWorktree(t *testing.T) {
	repository := &model.AIDevSessionRepository{
		ID: 12, LinkName: "child", Status: codeDeliveryCompleted, TargetBranch: "main",
		RemoteName: "origin", RemoteBranch: "main", MergeCommit: "merge", PushStatus: codePushPushed,
		PushedCommit: "merge", WorktreeDir: filepath.Join(t.TempDir(), "missing"),
	}
	result := codeStoredRepositoryDeliveryResult(repository)
	if result.Status != codeDeliveryCompleted || result.PushStatus != codePushPushed || result.PushedCommit != "merge" {
		t.Fatalf("stored repository result was not reconstructed: %#v", result)
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
