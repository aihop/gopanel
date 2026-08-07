package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestMultiRepositoryDeliveryRunnerUsesCapturedQualitySnapshot(t *testing.T) {
	session, project, _ := createMultiRepositorySession(t, 921)
	if err := global.DB.Model(project).Update("require_quality_gate", true).Error; err != nil {
		t.Fatal(err)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	for index := range repositories {
		repository := &repositories[index]
		if err := os.WriteFile(
			filepath.Join(repository.WorktreeDir, "package.json"),
			[]byte(`{"scripts":{"test":"true"}}`),
			0600,
		); err != nil {
			t.Fatal(err)
		}
		if _, _, err := saveCodeGitRepository(repository.WorktreeDir, "test: configure quality snapshot"); err != nil {
			t.Fatal(err)
		}
	}
	job, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	laterCommit := commitCodeTestFile(t, repositories[0].WorktreeDir, "later.txt", "later\n")

	previousCoordinator := codeExecutions
	codeExecutions = newCodeExecutionCoordinator(2)
	t.Cleanup(func() { codeExecutions = previousCoordinator })
	runner := &codeDeliveryRunner{
		queued: make(map[uint]struct{}), cancelled: make(map[uint]struct{}),
		owner: newCodeRepositoryLeaseOwner("multi-quality-test"),
	}
	runner.run(job.ID)
	if err := global.DB.First(job, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != codeDeliveryJobCompleted {
		t.Fatalf("captured multi-repository delivery failed: %#v", job)
	}
	var storedSession model.AIDevSession
	if err := global.DB.First(&storedSession, session.ID).Error; err != nil || storedSession.Status != codeSessionStatusActive {
		t.Fatalf("later commit should keep session active: %#v, %v", storedSession, err)
	}
	currentCommit, err := runCodeGit(repositories[0].WorktreeDir, "rev-parse", "HEAD")
	if err != nil || currentCommit != laterCommit {
		t.Fatalf("interactive worktree changed during snapshot delivery: got=%q want=%q err=%v", currentCommit, laterCommit, err)
	}
}
