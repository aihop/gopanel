package api

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func createMultiRepositorySession(t *testing.T, sessionID uint) (*model.AIDevSession, *model.AIGroup, []string) {
	t.Helper()
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	sourceDirs := []string{createCodeGitRepository(t), createCodeGitRepository(t)}
	project := &model.AIGroup{ID: sessionID, Name: "multi", CreatorID: 7, SourceDirs: sourceDirs}
	workDir, err := syncAIProjectWorkspace(project, sourceDirs)
	if err != nil {
		t.Fatal(err)
	}
	project.WorkDir = workDir
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: sessionID, UserID: 7, ProjectID: project.ID, Title: "session", WorkDir: project.WorkDir}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	if err := database.Save(session).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := os.Stat(session.WorkDir); err == nil && session.IsolationMode == codeIsolationMultiWorktree {
			rollbackCodeSessionWorktree(session)
		}
	})
	return session, project, sourceDirs
}

func TestCreateMultiRepositorySessionWorktrees(t *testing.T) {
	session, _, sourceDirs := createMultiRepositorySession(t, 81)
	if session.IsolationMode != codeIsolationMultiWorktree || session.SourceWorkDir != "" || session.WorktreeBranch != "" {
		t.Fatalf("unexpected session metadata: %#v", session)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("unexpected repositories: %#v, %v", repositories, err)
	}
	wantSources := append([]string(nil), sourceDirs...)
	gotSources := []string{repositories[0].SourceDir, repositories[1].SourceDir}
	if !reflect.DeepEqual(gotSources, wantSources) && !reflect.DeepEqual(gotSources, []string{wantSources[1], wantSources[0]}) {
		t.Fatalf("repository sources = %#v, want %#v", gotSources, wantSources)
	}
	for _, repository := range repositories {
		if !isPathInside(repository.WorktreeDir, session.WorkDir) {
			t.Fatalf("worktree outside session workspace: %#v", repository)
		}
		if repository.TargetBranch == "" || repository.BaseCommit == "" || repository.SyncStatus != "local" {
			t.Fatalf("repository baseline metadata unavailable: %#v", repository)
		}
		if _, err := os.Stat(filepath.Join(repository.WorktreeDir, "README.md")); err != nil {
			t.Fatalf("worktree content unavailable: %v", err)
		}
	}
	writableDirs, err := codexWritableDirsForSession(session)
	if err != nil || len(writableDirs) < 6 {
		t.Fatalf("unexpected writable dirs: %#v, %v", writableDirs, err)
	}
	for _, sourceDir := range sourceDirs {
		for _, writableDir := range writableDirs {
			if writableDir == sourceDir {
				t.Fatalf("source directory was exposed as writable: %s", sourceDir)
			}
		}
	}
}

func TestCreateMultiRepositorySessionDiscoversWorkspaceRepositories(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	workspace := t.TempDir()
	sourceDirs := []string{filepath.Join(workspace, "backend"), filepath.Join(workspace, "admin")}
	for _, sourceDir := range sourceDirs {
		if err := os.MkdirAll(sourceDir, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodeGit(sourceDir, "init"); err != nil {
			t.Fatal(err)
		}
		commitCodeTestFile(t, sourceDir, "README.md", "test\n")
	}
	project := &model.AIGroup{ID: 84, Name: "workspace", CreatorID: 7, SourceDirs: []string{workspace}, WorkDir: workspace}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: 84, UserID: 7, ProjectID: project.ID, WorkDir: workspace}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 || session.IsolationMode != codeIsolationMultiWorktree {
		t.Fatalf("workspace repositories were not isolated: %#v, %v", repositories, err)
	}
	for _, repository := range repositories {
		if !repositoryWithinSourceDirs(repository.SourceDir, project.SourceDirs) {
			t.Fatalf("repository escaped workspace boundary: %#v", repository)
		}
	}
}

func TestCommitAndMergeMultiRepositorySession(t *testing.T) {
	session, project, sourceDirs := createMultiRepositorySession(t, 82)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range repositories {
		repository := &repositories[index]
		filename := repository.LinkName + ".txt"
		if err := os.WriteFile(filepath.Join(repository.WorktreeDir, filename), []byte(repository.LinkName+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodeGit(repository.WorktreeDir, "add", filename); err != nil {
			t.Fatal(err)
		}
		result, err := commitCodeSessionRepository(session, codeSessionRepositoryID(repository.ID), "feat: update "+repository.LinkName)
		if err != nil || result.RepositoryID == "" || result.Commit == "" {
			t.Fatalf("unexpected commit result: %#v, %v", result, err)
		}
	}
	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != "merged" || len(result.Repositories) != 2 {
		t.Fatalf("unexpected merge result: %#v, %v", result, err)
	}
	for index, repository := range repositories {
		filename := repository.LinkName + ".txt"
		content, err := os.ReadFile(filepath.Join(sourceDirs[index], filename))
		if err != nil && len(sourceDirs) == 2 {
			content, err = os.ReadFile(filepath.Join(repository.SourceDir, filename))
		}
		if err != nil || string(content) != repository.LinkName+"\n" {
			t.Fatalf("merged content unavailable for %s: %q, %v", repository.LinkName, content, err)
		}
	}
	var stored model.AIDevSession
	if err := global.DB.First(&stored, session.ID).Error; err != nil || stored.WorkDir != project.WorkDir || stored.IsolationMode != "" {
		t.Fatalf("session was not restored: %#v, %v", stored, err)
	}
}

func TestMultiRepositoryDeliveryResumesAfterConflict(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 83)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range repositories {
		repository := &repositories[index]
		if err := os.WriteFile(filepath.Join(repository.WorktreeDir, "README.md"), []byte("worktree "+repository.LinkName+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodeGit(repository.WorktreeDir, "add", "README.md"); err != nil {
			t.Fatal(err)
		}
		if _, err := commitCodeSessionRepository(session, codeSessionRepositoryID(repository.ID), "feat: update "+repository.LinkName); err != nil {
			t.Fatal(err)
		}
	}
	conflictRepository := &repositories[1]
	if err := os.WriteFile(filepath.Join(conflictRepository.SourceDir, "README.md"), []byte("source change\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(conflictRepository.SourceDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(conflictRepository.SourceDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "source change"); err != nil {
		t.Fatal(err)
	}

	first, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err == nil || !strings.Contains(err.Error(), "隔离工作区解决") || first.Status != "" {
		t.Fatalf("unexpected conflict result: %#v, %v", first, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil || stored[0].Status != "committed" || stored[1].Status != "committed" {
		t.Fatalf("unexpected persisted delivery state: %#v, %v", stored, err)
	}
	if err := os.WriteFile(filepath.Join(conflictRepository.WorktreeDir, "README.md"), []byte("resolved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(conflictRepository.WorktreeDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(conflictRepository.WorktreeDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "--no-edit"); err != nil {
		t.Fatal(err)
	}

	second, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || second.Status != "merged" {
		t.Fatalf("delivery did not resume: %#v, %v", second, err)
	}
}
