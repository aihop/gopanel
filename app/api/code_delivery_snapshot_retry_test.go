package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type codeRepositoryRetryMetadata struct {
	Status          string
	RemoteCommit    string
	MergeCommit     string
	PushStatus      string
	PushedCommit    string
	PushError       string
	MergedAt        time.Time
	SourceAppliedAt *time.Time
	CompletedAt     *time.Time
	PushedAt        time.Time
	WorktreeCommit  string
}

func storeCodeRepositoryRetryMetadata(
	t *testing.T,
	repository *model.AIDevSessionRepository,
	status string,
	worktreeCommit string,
	stamp time.Time,
) codeRepositoryRetryMetadata {
	t.Helper()
	metadata := codeRepositoryRetryMetadata{
		Status: status, RemoteCommit: repository.BaseCommit, MergeCommit: repository.BaseCommit,
		PushStatus: codePushPushed, PushedCommit: repository.BaseCommit, PushError: "preserved",
		MergedAt: stamp, PushedAt: stamp.Add(time.Minute), WorktreeCommit: worktreeCommit,
	}
	if status == codeDeliveryCompleted {
		sourceAppliedAt := stamp.Add(90 * time.Second)
		completedAt := stamp.Add(2 * time.Minute)
		metadata.SourceAppliedAt = &sourceAppliedAt
		metadata.CompletedAt = &completedAt
	}
	if err := global.DB.Model(repository).Updates(map[string]any{
		"status": status, "remote_commit": metadata.RemoteCommit, "worktree_commit": worktreeCommit,
		"merge_commit": metadata.MergeCommit, "merged_at": metadata.MergedAt,
		"source_applied_at": metadata.SourceAppliedAt, "completed_at": metadata.CompletedAt,
		"push_status":   metadata.PushStatus,
		"pushed_commit": metadata.PushedCommit, "push_error": metadata.PushError, "pushed_at": metadata.PushedAt,
	}).Error; err != nil {
		t.Fatal(err)
	}
	return metadata
}

func assertCodeRepositoryRetryMetadata(
	t *testing.T,
	repository model.AIDevSessionRepository,
	expected codeRepositoryRetryMetadata,
) {
	t.Helper()
	if repository.Status != expected.Status || repository.RemoteCommit != expected.RemoteCommit ||
		repository.WorktreeCommit != expected.WorktreeCommit || repository.MergeCommit != expected.MergeCommit ||
		repository.PushStatus != expected.PushStatus || repository.PushedCommit != expected.PushedCommit ||
		repository.PushError != expected.PushError {
		t.Fatalf("repository retry metadata changed: %#v", repository)
	}
	if repository.MergedAt == nil || !repository.MergedAt.Equal(expected.MergedAt) ||
		repository.PushedAt == nil || !repository.PushedAt.Equal(expected.PushedAt) {
		t.Fatalf("repository retry timestamps changed: %#v", repository)
	}
	if (repository.SourceAppliedAt == nil) != (expected.SourceAppliedAt == nil) ||
		(repository.SourceAppliedAt != nil && !repository.SourceAppliedAt.Equal(*expected.SourceAppliedAt)) {
		t.Fatalf("repository source application timestamp changed: %#v", repository)
	}
	if (repository.CompletedAt == nil) != (expected.CompletedAt == nil) ||
		(repository.CompletedAt != nil && !repository.CompletedAt.Equal(*expected.CompletedAt)) {
		t.Fatalf("repository completion timestamp changed: %#v", repository)
	}
}

func TestCodeMultiRepositoryDeliverySnapshotPreservesUnchangedFinishedRepositories(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 930)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	expected := make(map[uint]codeRepositoryRetryMetadata, len(repositories))
	statuses := []string{codeDeliveryCompleted, codeDeliveryMerged}
	for index := range repositories {
		head, headErr := runCodeGit(repositories[index].WorktreeDir, "rev-parse", "HEAD")
		if headErr != nil {
			t.Fatal(headErr)
		}
		expected[repositories[index].ID] = storeCodeRepositoryRetryMetadata(
			t, &repositories[index], statuses[index], head,
			time.Date(2026, time.August, 7, 10, index, 0, 0, time.UTC),
		)
	}
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	completed := expected[repositories[0].ID]
	completed.SourceAppliedAt = nil
	expected[repositories[0].ID] = completed
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, repository := range stored {
		assertCodeRepositoryRetryMetadata(t, repository, expected[repository.ID])
	}
}

func TestCodeMultiRepositoryDeliverySnapshotResetsChangedAndFailedRepositories(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 931)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	firstHead, err := runCodeGit(repositories[0].WorktreeDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	storeCodeRepositoryRetryMetadata(t, &repositories[0], codeDeliveryCompleted, firstHead, time.Now().UTC())
	changedHead := commitCodeTestFile(t, repositories[0].WorktreeDir, "next.txt", "next\n")
	secondHead, err := runCodeGit(repositories[1].WorktreeDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	storeCodeRepositoryRetryMetadata(t, &repositories[1], "conflict", secondHead, time.Now().UTC())

	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		expectedCommit := secondHead
		if stored[index].ID == repositories[0].ID {
			expectedCommit = changedHead
		}
		if stored[index].Status != codeDeliveryPrepared || stored[index].WorktreeCommit != expectedCommit ||
			stored[index].RemoteCommit != repositories[index].RemoteCommit || stored[index].MergeCommit != "" ||
			stored[index].PushStatus != codePushPending || stored[index].PushedCommit != "" ||
			stored[index].PushError != "" || stored[index].MergedAt != nil ||
			stored[index].SourceAppliedAt != nil || stored[index].CompletedAt != nil || stored[index].PushedAt != nil {
			t.Fatalf("repository was not reset for retry: %#v", stored[index])
		}
	}
}

func TestCodeMultiRepositoryDeliverySnapshotResetsGitlinkAncestors(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	parent, child := createGitlinkRepositoryTree(t)
	project := &model.AIProject{ID: 932, Name: "retry-gitlink", CreatorID: 7, SourceDirs: []string{parent}, WorkDir: parent}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: 932, UserID: 7, ProjectID: project.ID, WorkDir: parent}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	var parentRepository, childRepository *model.AIDevSessionRepository
	for index := range repositories {
		repository := &repositories[index]
		head, headErr := runCodeGit(repository.WorktreeDir, "rev-parse", "HEAD")
		if headErr != nil {
			t.Fatal(headErr)
		}
		storeCodeRepositoryRetryMetadata(t, repository, codeDeliveryCompleted, head, time.Now().UTC())
		if repository.SourceDir == parent {
			parentRepository = repository
		}
		if repository.SourceDir == child {
			childRepository = repository
		}
	}
	if parentRepository == nil || childRepository == nil {
		t.Fatalf("gitlink repositories unavailable: %#v", repositories)
	}
	parentHead := parentRepository.WorktreeCommit
	childHead := commitCodeTestFile(t, childRepository.WorktreeDir, "next.txt", "next\n")
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, repository := range stored {
		if repository.Status != codeDeliveryPrepared || repository.MergeCommit != "" || repository.PushStatus != codePushPending {
			t.Fatalf("gitlink delivery dependency was not reset: %#v", repository)
		}
		if repository.SourceDir == parent && repository.WorktreeCommit != parentHead {
			t.Fatalf("parent snapshot commit changed unexpectedly: %#v", repository)
		}
		if repository.SourceDir == child && repository.WorktreeCommit != childHead {
			t.Fatalf("child snapshot commit was not refreshed: %#v", repository)
		}
	}
	if _, err := os.Stat(filepath.Join(childRepository.WorktreeDir, "next.txt")); err != nil {
		t.Fatal(err)
	}
}
