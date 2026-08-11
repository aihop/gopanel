package api

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestMultiRepositoryDeliveredCleanupResumesAfterPartialRemoval(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 822)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) < 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	for index := range repositories {
		commitCodeTestFile(t, repositories[index].WorktreeDir, "delivery.txt", repositories[index].LinkName+"\n")
	}
	if result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID); err != nil || result.Status != codeDeliveryMerged {
		t.Fatalf("deliver repositories: %#v, %v", result, err)
	}
	repositories, err = loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	removed := &repositories[0]
	if err := removeCodeSessionRepositoryWorktree(removed, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(removed.WorktreeDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("simulated partial cleanup did not remove worktree: %v", err)
	}
	if err := cleanupDeliveredCodeSessionWorktrees(session); err != nil {
		t.Fatalf("partial cleanup did not resume: %v", err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(stored) != len(repositories) {
		t.Fatalf("delivery review metadata was not preserved: %#v, %v", stored, err)
	}
	if _, err := os.Stat(session.WorkDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cleanup workspace remained: %v", err)
	}
}

func TestFinalizedMultiRepositorySessionPreservesTaskBranches(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 823)
	if err := global.DB.Model(session).Update("status", codeSessionStatusDelivered).Error; err != nil {
		t.Fatal(err)
	}
	session.Status = codeSessionStatusDelivered
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) < 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	branches := make(map[string]string, len(repositories))
	for index := range repositories {
		branches[repositories[index].SourceDir] = repositories[index].Branch
	}
	cleanupFinalizedCodeSessionWorktrees(session.ID)

	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(stored) != len(repositories) {
		t.Fatalf("finalized repository review metadata was not preserved: %#v, %v", stored, err)
	}
	if _, err := os.Stat(session.WorkDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("finalized multi-repository workspace remained: %v", err)
	}
	for sourceDir, branch := range branches {
		listed, listErr := runCodeGit(sourceDir, "branch", "--list", branch)
		if listErr != nil || strings.TrimSpace(listed) == "" {
			t.Fatalf("finalized task branch %s was not preserved in %s: %q, %v", branch, sourceDir, listed, listErr)
		}
	}
}

func TestDeliveryRunnerFinalizationPreservesMultiRepositoryTaskBranches(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 824)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) < 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	branches := make(map[string]string, len(repositories))
	for index := range repositories {
		branches[repositories[index].SourceDir] = repositories[index].Branch
	}
	if err := global.DB.Model(session).Updates(map[string]any{
		"status": codeSessionStatusDelivering, "current_stage": codeDeliveryStageCleaning,
	}).Error; err != nil {
		t.Fatal(err)
	}
	runner := &codeDeliveryRunner{
		queued: make(map[uint]struct{}), cancelled: make(map[uint]struct{}), owner: newCodeRepositoryLeaseOwner("finish-cleanup-test"),
	}
	job := &model.AICodeDeliveryJob{
		SessionID: session.ID, ProjectID: session.ProjectID, UserID: session.UserID,
		Status: codeDeliveryJobRunning, Stage: codeDeliveryStageCleaning, Progress: 90,
		RepositoryKeys: "[\"repositories\"]", LeaseOwner: runner.owner,
	}
	if err := global.DB.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	runner.finish(job, codeGitDeliveryResult{Status: codeDeliveryMerged}, nil)

	if err := global.DB.First(session, session.ID).Error; err != nil || session.Status != codeSessionStatusDelivered {
		t.Fatalf("multi-repository session was not finalized: %#v, %v", session, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(stored) != len(repositories) {
		t.Fatalf("runner cleanup review metadata was not preserved: %#v, %v", stored, err)
	}
	for sourceDir, branch := range branches {
		listed, listErr := runCodeGit(sourceDir, "branch", "--list", branch)
		if listErr != nil || strings.TrimSpace(listed) == "" {
			t.Fatalf("runner cleanup removed task branch %s: %q, %v", branch, listed, listErr)
		}
	}
}
