package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func createCodeGitRepository(t *testing.T) string {
	t.Helper()
	repositoryDir := t.TempDir()
	if _, err := runCodeGit(repositoryDir, "init"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "README.md"), []byte("test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(repositoryDir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestInspectCodeWorktreeCapability(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	available := inspectCodeWorktreeCapability(&model.AIGroup{SourceDirs: []string{repositoryDir}})
	if !available.Available || available.SourceDir != repositoryDir {
		t.Fatalf("unexpected capability: %#v", available)
	}
	secondRepository := createCodeGitRepository(t)
	multiSource := inspectCodeWorktreeCapability(&model.AIGroup{SourceDirs: []string{repositoryDir, secondRepository}})
	if !multiSource.Available || multiSource.RepositoryCount != 2 || len(multiSource.SourceDirs) != 2 {
		t.Fatalf("unexpected multi-source capability: %#v", multiSource)
	}
	notGit := inspectCodeWorktreeCapability(&model.AIGroup{SourceDirs: []string{t.TempDir()}})
	if notGit.Available || notGit.Reason != "not_git" {
		t.Fatalf("unexpected non-git capability: %#v", notGit)
	}
	subdirectory := filepath.Join(repositoryDir, "nested")
	if err := os.Mkdir(subdirectory, 0755); err != nil {
		t.Fatal(err)
	}
	notRoot := inspectCodeWorktreeCapability(&model.AIGroup{SourceDirs: []string{subdirectory}})
	if notRoot.Available || notRoot.Reason != "not_git_root" {
		t.Fatalf("unexpected nested repository capability: %#v", notRoot)
	}
}

func TestCreateAndRollbackCodeSessionWorktree(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 21, UserID: 7, WorkDir: repositoryDir}
	project := &model.AIGroup{SourceDirs: []string{repositoryDir}}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	if session.SourceWorkDir != repositoryDir || !strings.HasPrefix(session.WorktreeBranch, "gopanel/code-21-") {
		t.Fatalf("unexpected session worktree metadata: %#v", session)
	}
	if _, err := os.Stat(filepath.Join(session.WorkDir, "README.md")); err != nil {
		t.Fatalf("worktree content unavailable: %v", err)
	}
	if !isManagedAISessionWorkDir(session.WorkDir, session.UserID) {
		t.Fatal("created worktree was not recognized as managed")
	}
	if isManagedAISessionWorkDir(filepath.Join(aiProjectWorktreeRoot(session.UserID), "session_invalid"), session.UserID) {
		t.Fatal("invalid worktree directory was recognized as managed")
	}
	rollbackCodeSessionWorktree(session)
	if _, err := os.Stat(session.WorkDir); !os.IsNotExist(err) {
		t.Fatalf("worktree still exists: %v", err)
	}
	if _, err := runCodeGit(repositoryDir, "show-ref", "--verify", "refs/heads/"+session.WorktreeBranch); err == nil {
		t.Fatal("worktree branch still exists")
	}
}

func TestCleanupCodeSessionWorktreePreservesUncommittedChanges(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 22, UserID: 7, WorkDir: repositoryDir}
	project := &model.AIGroup{SourceDirs: []string{repositoryDir}}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "README.md"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := cleanupCodeSessionWorktree(session); err == nil {
		t.Fatal("dirty worktree should be preserved")
	}
	if _, err := os.Stat(session.WorkDir); err != nil {
		t.Fatalf("dirty worktree was removed: %v", err)
	}
	rollbackCodeSessionWorktree(session)
}

func TestCleanupCodeSessionWorktreePreservesUnmergedCommit(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 23, UserID: 7, WorkDir: repositoryDir}
	project := &model.AIGroup{SourceDirs: []string{repositoryDir}}
	if err := createCodeSessionWorktree(session, project); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, "result.txt"), []byte("finished\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "result.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "result"); err != nil {
		t.Fatal(err)
	}
	if err := cleanupCodeSessionWorktree(session); err == nil {
		t.Fatal("unmerged worktree should be preserved")
	}
	if _, err := os.Stat(session.WorkDir); err != nil {
		t.Fatalf("unmerged worktree was removed: %v", err)
	}
	if _, err := runCodeGit(repositoryDir, "show-ref", "--verify", "refs/heads/"+session.WorktreeBranch); err != nil {
		t.Fatalf("unmerged branch was removed: %v", err)
	}
	rollbackCodeSessionWorktree(session)
}
