package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestCodeDeliveryExcludesRepositoryConfiguredAfterSessionCreation(t *testing.T) {
	session, project, _ := createMultiRepositorySession(t, 954)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	included, excluded := &repositories[0], &repositories[1]
	includedCommit := commitCodeTestFile(t, included.WorktreeDir, "included.txt", "included\n")
	if err := os.WriteFile(filepath.Join(excluded.WorktreeDir, "stale-dirty.txt"), []byte("dirty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	project.ExcludedRepositories = []string{excluded.SourceDir}
	if err := global.DB.Save(project).Error; err != nil {
		t.Fatal(err)
	}

	status, err := loadCodeGitResultStatus(session, project.ExcludedRepositories)
	if err != nil || !status.ReviewReady || status.ReviewRevision == "" || len(status.Repositories) != 1 {
		t.Fatalf("filtered review unavailable: %#v, %v", status, err)
	}
	job, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1", status.ReviewRevision)
	if err != nil {
		t.Fatalf("excluded dirty repository blocked enqueue: %v", err)
	}
	keys, err := decodeCodeDeliveryKeys(job)
	if err != nil || len(keys) != 1 {
		t.Fatalf("delivery repository keys = %#v, %v", keys, err)
	}
	wantKey := codeDeliveryRepositoryKey(included.SourceDir, included.RemoteName, included.TargetBranch)
	if keys[0] != wantKey {
		t.Fatalf("delivery repository key = %q, want %q", keys[0], wantKey)
	}

	prepared, err := prepareCodeMultiRepositoryDeliveryWithProgress(session, nil)
	if err != nil || prepared.Status != codeDeliveryMerged || len(prepared.Repositories) != 1 {
		t.Fatalf("filtered delivery preparation failed: %#v, %v", prepared, err)
	}
	roots, cleanup, err := prepareCodeMultiRepositoryQualityRoots(session)
	if err != nil || len(roots) != 1 || roots[0].RuntimeDir != included.SourceDir {
		t.Fatalf("delivery quality roots = %#v, %v", roots, err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	result, err := publishCodeMultiRepositoryDeliveryWithProgress(session, nil)
	if err != nil || result.Status != codeDeliveryMerged || len(result.Repositories) != 1 {
		t.Fatalf("filtered delivery failed: %#v, %v", result, err)
	}
	if result.Repositories[0].RepositoryPath != included.SourceDir {
		t.Fatalf("excluded repository leaked into delivery result: %#v", result.Repositories)
	}
	if _, err := runCodeGit(included.SourceDir, "merge-base", "--is-ancestor", includedCommit, "HEAD"); err != nil {
		t.Fatalf("included repository was not delivered: %v", err)
	}

	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(stored) != 2 {
		t.Fatalf("excluded repository history was removed: %#v, %v", stored, err)
	}
	var storedExcluded *model.AIDevSessionRepository
	for index := range stored {
		if stored[index].ID == excluded.ID {
			storedExcluded = &stored[index]
		}
	}
	if storedExcluded == nil || storedExcluded.Status == codeDeliveryCompleted {
		t.Fatalf("excluded repository delivery state was mutated: %#v", storedExcluded)
	}
	if status, err := runCodeGit(excluded.WorktreeDir, "status", "--porcelain"); err != nil || status == "" {
		t.Fatalf("excluded dirty worktree was touched: %q, %v", status, err)
	}
	facts := loadCodeDeliveryFacts(session.ID, result.Repositories)
	for _, fact := range facts {
		if fact.Total > 1 {
			t.Fatalf("excluded repository leaked into delivery facts: %#v", facts)
		}
	}
}
