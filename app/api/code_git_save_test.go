package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestSaveCodeSessionWorktreeCommitsAllChanges(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 141)
	if err := os.WriteFile(filepath.Join(session.WorkDir, "README.md"), []byte("updated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "new.txt"), []byte("new\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := saveCodeSessionWorktree(session, "")
	if err != nil || result.Status != "committed" || result.Commit == "" {
		t.Fatalf("unexpected save result: %#v, %v", result, err)
	}
	status, err := runCodeGit(session.WorkDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		t.Fatalf("saved worktree is dirty: %q, %v", status, err)
	}
	message, err := runCodeGit(session.WorkDir, "log", "-1", "--pretty=%s")
	if err != nil || message != defaultCodeGitSaveMessage {
		t.Fatalf("commit message = %q, want %q: %v", message, defaultCodeGitSaveMessage, err)
	}
}

func TestSaveCodeSessionWorktreeIncludesDeletion(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 142)
	if err := os.Remove(filepath.Join(session.WorkDir, "README.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "test: save deletion"); err != nil {
		t.Fatal(err)
	}
	changed, err := runCodeGit(session.WorkDir, "show", "--pretty=", "--name-status", "HEAD")
	if err != nil || !strings.Contains(changed, "D\tREADME.md") {
		t.Fatalf("deleted file was not committed: %q, %v", changed, err)
	}
}

func TestSaveCodeSessionWorktreeRejectsCleanWorktree(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 143)
	if _, err := saveCodeSessionWorktree(session, ""); err == nil || !strings.Contains(err.Error(), "没有需要保存") {
		t.Fatalf("clean worktree should be rejected: %v", err)
	}
}

func TestSaveCodeSessionRepositoriesCommitsChangedRepositories(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 144)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	for index := range repositories {
		if err := os.WriteFile(
			filepath.Join(repositories[index].WorktreeDir, "saved.txt"),
			[]byte(repositories[index].LinkName+"\n"), 0600,
		); err != nil {
			t.Fatal(err)
		}
	}
	result, err := saveCodeSessionRepositories(session, "feat: save session")
	if err != nil || result.Status != "committed" || len(result.Repositories) != 2 {
		t.Fatalf("unexpected save result: %#v, %v", result, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, repository := range stored {
		status, statusErr := runCodeGit(repository.WorktreeDir, "status", "--porcelain")
		if statusErr != nil || strings.TrimSpace(status) != "" || repository.Status != "committed" || repository.WorktreeCommit == "" {
			t.Fatalf("repository was not saved: %#v, status=%q, err=%v", repository, status, statusErr)
		}
	}
	delivered, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || delivered.Status != "merged" || len(delivered.Repositories) != 2 {
		t.Fatalf("saved repositories were not ready for delivery: %#v, %v", delivered, err)
	}
}

func TestSaveCodeSessionRepositoriesCommitsGitlinkPointer(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	parent, child := createGitlinkRepositoryTree(t)
	project := &model.AIProject{ID: 145, Name: "gitlink-save", CreatorID: 7, SourceDirs: []string{parent}, WorkDir: parent}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: 145, UserID: 7, ProjectID: project.ID, WorkDir: parent}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	if err := database.Save(session).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	var parentRepository, childRepository *model.AIDevSessionRepository
	for index := range repositories {
		repository := &repositories[index]
		if repository.SourceDir == parent {
			parentRepository = repository
		}
		if repository.SourceDir == child {
			childRepository = repository
		}
	}
	if parentRepository == nil || childRepository == nil {
		t.Fatalf("repository relationship unavailable: %#v", repositories)
	}
	if err := os.WriteFile(filepath.Join(childRepository.WorktreeDir, "saved.txt"), []byte("saved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionRepositories(session, "feat: save nested repository"); err != nil {
		t.Fatal(err)
	}
	childHead, err := runCodeGit(childRepository.WorktreeDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	parentEntry, err := runCodeGit(parentRepository.WorktreeDir, "ls-tree", "HEAD", "--", childRepository.GitlinkPath)
	if err != nil || !strings.Contains(parentEntry, childHead) {
		t.Fatalf("parent gitlink = %q, want child %s: %v", parentEntry, childHead, err)
	}
	for _, repository := range repositories {
		status, statusErr := runCodeGit(repository.WorktreeDir, "status", "--porcelain")
		if statusErr != nil || strings.TrimSpace(status) != "" {
			t.Fatalf("repository %s remains dirty: %q, %v", repository.LinkName, status, statusErr)
		}
	}
	var savedCount int64
	if err := global.DB.Model(&model.AIDevSessionRepository{}).Where(
		"session_id = ? AND status = ? AND worktree_commit <> ?", session.ID, "committed", "",
	).Count(&savedCount).Error; err != nil || savedCount != 2 {
		t.Fatalf("saved metadata count = %d: %v", savedCount, err)
	}
}
