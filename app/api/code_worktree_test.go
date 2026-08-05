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

func TestCodeGitEnvironmentRestoresServiceAccountSettings(t *testing.T) {
	homeDir := codeExecutorHomeDir()
	if homeDir == "" {
		t.Skip("current service account home is unavailable")
	}
	t.Setenv("HOME", "")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("SSH_AUTH_SOCK", "/tmp/gopanel-test-agent.sock")

	environment := make(map[string]string)
	for _, item := range codeGitEnvironment() {
		key, value, found := strings.Cut(item, "=")
		if found {
			environment[key] = value
		}
	}

	if environment["HOME"] != homeDir {
		t.Fatalf("HOME = %q, want %q", environment["HOME"], homeDir)
	}
	if environment["GIT_TERMINAL_PROMPT"] != "0" {
		t.Fatalf("GIT_TERMINAL_PROMPT = %q", environment["GIT_TERMINAL_PROMPT"])
	}
	if environment["SSH_AUTH_SOCK"] != "/tmp/gopanel-test-agent.sock" {
		t.Fatalf("SSH_AUTH_SOCK was not preserved: %q", environment["SSH_AUTH_SOCK"])
	}
	if !strings.Contains(environment["PATH"], "/usr/local/bin") {
		t.Fatalf("PATH does not include service Git locations: %q", environment["PATH"])
	}
}

func TestInspectCodeWorktreeCapability(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	available := inspectCodeWorktreeCapability(&model.AIProject{SourceDirs: []string{repositoryDir}})
	if !available.Available || available.SourceDir != repositoryDir {
		t.Fatalf("unexpected capability: %#v", available)
	}
	secondRepository := createCodeGitRepository(t)
	multiSource := inspectCodeWorktreeCapability(&model.AIProject{SourceDirs: []string{repositoryDir, secondRepository}})
	if !multiSource.Available || multiSource.RepositoryCount != 2 || len(multiSource.SourceDirs) != 2 {
		t.Fatalf("unexpected multi-source capability: %#v", multiSource)
	}
	notGit := inspectCodeWorktreeCapability(&model.AIProject{SourceDirs: []string{t.TempDir()}})
	if notGit.Available || notGit.Reason != "not_git" {
		t.Fatalf("unexpected non-git capability: %#v", notGit)
	}
	subdirectory := filepath.Join(repositoryDir, "nested")
	if err := os.Mkdir(subdirectory, 0755); err != nil {
		t.Fatal(err)
	}
	notRoot := inspectCodeWorktreeCapability(&model.AIProject{SourceDirs: []string{subdirectory}})
	if notRoot.Available || notRoot.Reason != "not_git" {
		t.Fatalf("unexpected nested repository capability: %#v", notRoot)
	}
}

func TestInspectCodeWorktreeCapabilityDiscoversNestedRepositories(t *testing.T) {
	workspace := t.TempDir()
	first := filepath.Join(workspace, "apps", "api")
	second := filepath.Join(workspace, "services", "worker")
	if err := os.MkdirAll(first, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(second, 0755); err != nil {
		t.Fatal(err)
	}
	for _, repository := range []string{first, second} {
		if _, err := runCodeGit(repository, "init"); err != nil {
			t.Fatal(err)
		}
	}
	capability := inspectCodeWorktreeCapability(&model.AIProject{SourceDirs: []string{workspace}})
	if !capability.Available || capability.RepositoryCount != 2 {
		t.Fatalf("nested repositories were not discovered: %#v", capability)
	}
}

func TestCreateAndRollbackCodeSessionWorktree(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 21, UserID: 7, WorkDir: repositoryDir}
	project := &model.AIProject{SourceDirs: []string{repositoryDir}}
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

func TestCreateCodeSessionWorktreeSnapshotsDirtyRepositoryByDefault(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repositoryDir, "README.md"), []byte("staged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "working.txt"), []byte("untracked\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sourceStatus, err := runCodeGit(repositoryDir, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{ID: 25, UserID: 7, WorkDir: repositoryDir}
	if err := createCodeSessionWorktree(session, &model.AIProject{SourceDirs: []string{repositoryDir}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	worktreeStatus, err := runCodeGit(session.WorkDir, "status", "--porcelain")
	if err != nil || !strings.Contains(worktreeStatus, "M  README.md") || !strings.Contains(worktreeStatus, "?? working.txt") {
		t.Fatalf("dirty state was not copied: %q, %v", worktreeStatus, err)
	}
	unchangedStatus, err := runCodeGit(repositoryDir, "status", "--porcelain")
	if err != nil || unchangedStatus != sourceStatus || session.RepositorySync != "snapshot" {
		t.Fatalf("source repository changed or snapshot metadata missing: before=%q after=%q session=%#v err=%v", sourceStatus, unchangedStatus, session, err)
	}
}

func TestCodeSessionWorktreeBlocksDirectPush(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir, _ := createCodeRemoteRepository(t)
	session := &model.AIDevSession{ID: 24, UserID: 7, WorkDir: repositoryDir}
	if err := createCodeSessionWorktree(session, &model.AIProject{SourceDirs: []string{repositoryDir}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	if _, err := runCodeGit(session.WorkDir, "push", "origin", "HEAD:refs/heads/blocked-direct-push"); err == nil || !strings.Contains(err.Error(), "统一交付") {
		t.Fatalf("direct worktree push should be blocked: %v", err)
	}
}

func TestCleanupCodeSessionWorktreePreservesUncommittedChanges(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 22, UserID: 7, WorkDir: repositoryDir}
	project := &model.AIProject{SourceDirs: []string{repositoryDir}}
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
	project := &model.AIProject{SourceDirs: []string{repositoryDir}}
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
