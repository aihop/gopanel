package api

import (
	"errors"
	"os"
	"testing"
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
	if err != nil || len(stored) != 0 {
		t.Fatalf("cleanup metadata remained: %#v, %v", stored, err)
	}
	if _, err := os.Stat(session.WorkDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cleanup workspace remained: %v", err)
	}
}
