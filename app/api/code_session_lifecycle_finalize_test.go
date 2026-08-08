package api

import (
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func createCodeLifecycleFinishFixture(t *testing.T, sessionID uint) (*model.AIDevSession, *model.AITask, *model.AICodeDeliveryJob, *codeDeliveryRunner) {
	t.Helper()
	database := withCodeGovernanceDB(t)
	workDir := createCodeGitRepository(t)
	head, err := runCodeGit(workDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{
		ID: sessionID, UserID: 1, ProjectID: 1, Title: "finish", WorkDir: workDir,
		Status: codeSessionStatusDelivering, CurrentStage: codeDeliveryStagePushing,
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{
		ID: sessionID + 1000, UserID: session.UserID, ProjectID: session.ProjectID,
		SessionID: session.ID, Title: "finish", WorkDir: workDir, Status: codeSessionStatusDelivering,
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(session).Update("last_task_id", task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AICodeDelivery{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryCompleted, WorkDir: workDir, WorktreeCommit: head,
	}).Error; err != nil {
		t.Fatal(err)
	}
	runner := &codeDeliveryRunner{queued: make(map[uint]struct{}), cancelled: make(map[uint]struct{}), owner: "finish-runner"}
	job := &model.AICodeDeliveryJob{
		SessionID: session.ID, TaskID: task.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryJobRunning, Stage: codeDeliveryStagePushing, Progress: 80,
		RepositoryKeys: "[\"repository\"]", LeaseOwner: runner.owner,
	}
	if err := database.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().Add(time.Minute)
	if err := database.Create(&model.AICodeDeliveryLease{
		RepositoryKey: "repository", JobID: job.ID, LeaseOwner: runner.owner, LeaseExpiresAt: &expiresAt,
	}).Error; err != nil {
		t.Fatal(err)
	}
	return session, task, job, runner
}

func TestCodeDeliveryRunnerFinishFinalizesLifecycleAtomically(t *testing.T) {
	session, task, job, runner := createCodeLifecycleFinishFixture(t, 940)
	done := make(chan struct{})
	go func() {
		runner.finish(job, codeGitDeliveryResult{Status: "merged", Commit: "result"}, nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runner finish deadlocked while finalizing lifecycle")
	}
	if err := global.DB.First(job, job.ID).Error; err != nil || job.Status != codeDeliveryJobCompleted || job.CompletedAt == nil {
		t.Fatalf("delivery job was not completed: %#v, %v", job, err)
	}
	if err := global.DB.First(session, session.ID).Error; err != nil || session.Status != codeSessionStatusDelivered || session.CurrentStage != codeDeliveryStageCompleted {
		t.Fatalf("session lifecycle was not finalized: %#v, %v", session, err)
	}
	if err := global.DB.First(task, task.ID).Error; err != nil || task.Status != "completed" {
		t.Fatalf("task lifecycle was not finalized: %#v, %v", task, err)
	}
	var leases int64
	if err := global.DB.Model(&model.AICodeDeliveryLease{}).Where("job_id = ?", job.ID).Count(&leases).Error; err != nil || leases != 0 {
		t.Fatalf("delivery lease was not released: %d, %v", leases, err)
	}
}

func TestCodeDeliveryRunnerFinishContinuesPostSnapshotDevelopment(t *testing.T) {
	session, _, job, runner := createCodeLifecycleFinishFixture(t, 941)
	commitCodeTestFile(t, session.WorkDir, "later.txt", "later\n")
	runner.finish(job, codeGitDeliveryResult{Status: "merged", Commit: "result"}, nil)
	if err := global.DB.First(session, session.ID).Error; err != nil || session.Status != codeSessionStatusActive || session.CurrentStage != codeDeliveryStageCompleted {
		t.Fatalf("post-snapshot development was not preserved: %#v, %v", session, err)
	}
}

func TestRestoreCodeDeliveryLifecycleOnlyUpdatesCurrentTask(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{ID: 942, UserID: 1, Status: codeSessionStatusActive, WorkDir: t.TempDir()}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	historical := &model.AITask{ID: 1942, UserID: 1, SessionID: session.ID, Title: "old", WorkDir: session.WorkDir, Status: "completed"}
	current := &model.AITask{ID: 1943, UserID: 1, SessionID: session.ID, Title: "current", WorkDir: session.WorkDir, Status: "completed"}
	if err := database.Create([]*model.AITask{historical, current}).Error; err != nil {
		t.Fatal(err)
	}
	job := &model.AICodeDeliveryJob{
		SessionID: session.ID, TaskID: current.ID, ProjectID: 1, UserID: 1,
		Status: codeDeliveryJobQueued, Stage: codeDeliveryStageQueued, RepositoryKeys: "[\"repository\"]",
	}
	if err := database.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	restoreCodeDeliverySessionLifecycles()
	if err := database.First(session, session.ID).Error; err != nil || session.Status != codeSessionStatusDelivering {
		t.Fatalf("active delivery lifecycle was not restored: %#v, %v", session, err)
	}
	if err := database.First(current, current.ID).Error; err != nil || current.Status != codeSessionStatusDelivering {
		t.Fatalf("current task lifecycle was not restored: %#v, %v", current, err)
	}
	if err := database.First(historical, historical.ID).Error; err != nil || historical.Status != "completed" {
		t.Fatalf("historical task lifecycle was mutated: %#v, %v", historical, err)
	}
}
