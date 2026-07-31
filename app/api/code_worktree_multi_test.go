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

func createMultiRepositorySession(t *testing.T, sessionID uint) (*model.AIDevSession, *model.AIProject, []string) {
	t.Helper()
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	sourceDirs := []string{createCodeGitRepository(t), createCodeGitRepository(t)}
	project := &model.AIProject{ID: sessionID, Name: "multi", CreatorID: 7, SourceDirs: sourceDirs}
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

func TestCodexMultiWorktreeWritableDirsRepairsSameSessionBranch(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 134)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	repository := repositories[0]
	newBranch := "gopanel/code-134-recovered-1"
	if _, err := runCodeGit(repository.WorktreeDir, "branch", "-m", newBranch); err != nil {
		t.Fatal(err)
	}

	if _, err := codexWritableDirsForSessionWithRepair(session); err != nil {
		t.Fatal(err)
	}
	stored, err := codeSessionRepositoryByCodeID(session.ID, codeSessionRepositoryID(repository.ID))
	if err != nil || stored.Branch != newBranch {
		t.Fatalf("stored repository branch = %q, err=%v", stored.Branch, err)
	}
	manifest, err := os.ReadFile(filepath.Join(session.WorkDir, codeSessionManifestName))
	if err != nil || !strings.Contains(string(manifest), `"branch": "`+newBranch+`"`) {
		t.Fatalf("session manifest was not repaired: %v, %s", err, manifest)
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
	project := &model.AIProject{ID: 84, Name: "workspace", CreatorID: 7, SourceDirs: []string{workspace}, WorkDir: workspace}
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

func TestCreateGitlinkRepositorySnapshotPreservesSourceState(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	parent, child := createGitlinkRepositoryTree(t)
	if err := os.WriteFile(filepath.Join(child, "staged.txt"), []byte("staged snapshot\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(child, "add", "staged.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "working.txt"), []byte("working snapshot\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "untracked.txt"), []byte("untracked snapshot\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sourceStatus, err := runCodeGit(child, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{ID: 85, Name: "gitlink", CreatorID: 7, SourceDirs: []string{parent}, WorkDir: parent}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: 85, UserID: 7, ProjectID: project.ID, WorkDir: parent}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, project, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("unexpected repositories: %#v, %v", repositories, err)
	}
	var childWorktree string
	for _, repository := range repositories {
		if repository.SourceDir == child {
			childWorktree = repository.WorktreeDir
			if !repository.Snapshot || repository.ParentSourceDir != parent || repository.GitlinkPath != "themes/custom" {
				t.Fatalf("snapshot metadata unavailable: %#v", repository)
			}
		}
	}
	if childWorktree == "" {
		t.Fatal("child worktree unavailable")
	}
	for name, expected := range map[string]string{
		"staged.txt": "staged snapshot\n", "working.txt": "working snapshot\n", "untracked.txt": "untracked snapshot\n",
	} {
		content, readErr := os.ReadFile(filepath.Join(childWorktree, name))
		if readErr != nil || string(content) != expected {
			t.Fatalf("snapshot file %s = %q, %v", name, content, readErr)
		}
	}
	worktreeStatus, err := runCodeGit(childWorktree, "status", "--porcelain")
	if err != nil || !strings.Contains(worktreeStatus, "M  staged.txt") || !strings.Contains(worktreeStatus, " M working.txt") || !strings.Contains(worktreeStatus, "?? untracked.txt") {
		t.Fatalf("snapshot state not preserved: %q, %v", worktreeStatus, err)
	}
	unchangedStatus, err := runCodeGit(child, "status", "--porcelain")
	if err != nil || unchangedStatus != sourceStatus {
		t.Fatalf("source repository changed: before=%q after=%q err=%v", sourceStatus, unchangedStatus, err)
	}
}

func TestSyncCodeSessionRepositoryGitlinksStagesChildCommitInParent(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	parent, child := createGitlinkRepositoryTree(t)
	project := &model.AIProject{ID: 86, Name: "gitlink", CreatorID: 7, SourceDirs: []string{parent}, WorkDir: parent}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: 86, UserID: 7, ProjectID: project.ID, WorkDir: parent}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, project); err != nil {
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
	childCommit := commitCodeTestFile(t, childRepository.WorktreeDir, "delivery.txt", "delivery\n")
	if err := syncCodeSessionRepositoryGitlinks(repositories); err != nil {
		t.Fatal(err)
	}
	entry, err := runCodeGit(parentRepository.WorktreeDir, "ls-files", "-s", "--", childRepository.GitlinkPath)
	if err != nil || !strings.Contains(entry, childCommit) {
		t.Fatalf("parent gitlink entry = %q, want %s: %v", entry, childCommit, err)
	}
	staged, err := runCodeGit(parentRepository.WorktreeDir, "diff", "--cached", "--name-only")
	if err != nil || strings.TrimSpace(staged) != childRepository.GitlinkPath {
		t.Fatalf("parent gitlink was not staged: %q, %v", staged, err)
	}
}

func TestCommitAndMergeGitlinkRepositorySession(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	parent, child := createGitlinkRepositoryTree(t)
	project := &model.AIProject{ID: 87, Name: "gitlink", CreatorID: 7, SourceDirs: []string{parent}, WorkDir: parent}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: 87, UserID: 7, ProjectID: project.ID, WorkDir: parent}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, statErr := os.Stat(session.WorkDir); statErr == nil && session.IsolationMode == codeIsolationMultiWorktree {
			rollbackCodeSessionWorktree(session)
		}
	})
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
	if err := os.WriteFile(filepath.Join(childRepository.WorktreeDir, "delivery.txt"), []byte("delivery\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(childRepository.WorktreeDir, "add", "delivery.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := commitCodeSessionRepository(session, codeSessionRepositoryID(childRepository.ID), "feat: child delivery"); err != nil {
		t.Fatal(err)
	}
	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != "merged" {
		t.Fatalf("gitlink delivery failed: %#v, %v", result, err)
	}
	childHead, err := runCodeGit(child, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	parentEntry, err := runCodeGit(parent, "ls-files", "-s", "--", "themes/custom")
	if err != nil || !strings.Contains(parentEntry, childHead) {
		t.Fatalf("parent gitlink = %q, want child %s: %v", parentEntry, childHead, err)
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
