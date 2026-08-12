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
	codeExecutions = newCodeExecutionCoordinator(2, 2)
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

func TestMultiRepositoryRepeatedDeliveryDoesNotRunQualityChecks(t *testing.T) {
	session, project, _ := createMultiRepositorySession(t, 948)
	if err := global.DB.Model(project).Update("require_quality_gate", true).Error; err != nil {
		t.Fatal(err)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) < 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	changed := &repositories[0]
	marker := filepath.Join(t.TempDir(), "quality-runs")
	writeAndCommitCodeTestFiles(t, changed.WorktreeDir, map[string]string{
		"package.json": `{"scripts":{"test":"node verify.js"}}`,
		"verify.js": "const fs=require('fs');fs.appendFileSync(" + quotedCodeTestJS(marker) +
			",'run\\n');if(fs.existsSync('fail-quality'))process.exit(1);\n",
	})
	first, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || first.Status != codeDeliveryMerged {
		t.Fatalf("first delivery failed: %#v, %v", first, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("first Git delivery executed quality script: %v", err)
	}
	firstSourceHead, err := runCodeGit(changed.SourceDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	commitCodeTestFile(t, changed.WorktreeDir, "fail-quality", "fail\n")
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	if prepared, err := prepareCodeMultiRepositoryDeliveryWithProgress(session, nil); err != nil || prepared.Status != codeDeliveryMerged {
		t.Fatalf("prepare second delivery: %#v, %v", prepared, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil || !codeMultiRepositoryDeliveryFrozen(stored) {
		t.Fatalf("second delivery was not frozen: %#v, %v", stored, err)
	}
	for index := range stored {
		repository := &stored[index]
		if repository.ID == changed.ID {
			if repository.Status != codeDeliveryMerged {
				t.Fatalf("changed repository was not merged: %#v", repository)
			}
			continue
		}
		if repository.Status != codeDeliveryCompleted || repository.SourceAppliedAt != nil {
			t.Fatalf("unchanged repository leaked previous delivery state: %#v", repository)
		}
	}
	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != codeDeliveryMerged {
		t.Fatalf("second Git delivery failed: %#v, %v", result, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("second Git delivery executed quality script: %v", err)
	}
	currentSourceHead, err := runCodeGit(changed.SourceDir, "rev-parse", "HEAD")
	if err != nil || currentSourceHead == firstSourceHead {
		t.Fatalf("second Git delivery did not advance source: got=%q previous=%q err=%v", currentSourceHead, firstSourceHead, err)
	}
}

func TestMultiRepositorySecondDeliveryRerunsUnchangedRepositoryQuality(t *testing.T) {
	session, project, _ := createMultiRepositorySession(t, 949)
	if err := global.DB.Model(project).Update("require_quality_gate", true).Error; err != nil {
		t.Fatal(err)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) < 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	changed, checked := &repositories[0], &repositories[1]
	marker := filepath.Join(t.TempDir(), "quality-runs")
	writeAndCommitCodeTestFiles(t, checked.WorktreeDir, map[string]string{
		"package.json": `{"scripts":{"test":"node verify.js"}}`,
		"verify.js":    "require('fs').appendFileSync(" + quotedCodeTestJS(marker) + ",'run\\n');\n",
	})
	if result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID); err != nil || result.Status != codeDeliveryMerged {
		t.Fatalf("first delivery failed: %#v, %v", result, err)
	}
	commitCodeTestFile(t, changed.WorktreeDir, "second-delivery.txt", "changed\n")
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	if result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID); err != nil || result.Status != codeDeliveryMerged {
		t.Fatalf("second delivery failed: %#v, %v", result, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("repeated Git delivery executed quality script: %v", err)
	}
}
