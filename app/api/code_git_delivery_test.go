package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func createDeliveryWorktree(t *testing.T, sessionID uint) (*model.AIDevSession, string) {
	t.Helper()
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: sessionID, UserID: 7, WorkDir: repositoryDir}
	if err := createCodeSessionWorktree(session, &model.AIGroup{SourceDirs: []string{repositoryDir}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := os.Stat(session.WorkDir); err == nil {
			rollbackCodeSessionWorktree(session)
		}
	})
	return session, repositoryDir
}

func TestCommitAndMergeCodeSessionWorktree(t *testing.T) {
	session, sourceDir := createDeliveryWorktree(t, 31)
	if err := os.WriteFile(filepath.Join(session.WorkDir, "result.txt"), []byte("done\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "result.txt"); err != nil {
		t.Fatal(err)
	}
	committed, err := commitCodeSessionWorktree(session, "feat: add result")
	if err != nil || committed.Status != "committed" || committed.Commit == "" {
		t.Fatalf("unexpected commit result: %#v, %v", committed, err)
	}
	merged, err := mergeCodeSessionWorktree(session)
	if err != nil || merged.Status != "merged" || merged.Commit == "" {
		t.Fatalf("unexpected merge result: %#v, %v", merged, err)
	}
	content, err := os.ReadFile(filepath.Join(sourceDir, "result.txt"))
	if err != nil || string(content) != "done\n" {
		t.Fatalf("merged file unavailable: %q, %v", content, err)
	}
	if _, err := os.Stat(session.WorkDir); !os.IsNotExist(err) {
		t.Fatalf("merged worktree was not cleaned: %v", err)
	}
}

func TestMergeCodeSessionWorktreeReportsAndAbortsConflict(t *testing.T) {
	session, sourceDir := createDeliveryWorktree(t, 32)
	if err := os.WriteFile(filepath.Join(session.WorkDir, "README.md"), []byte("worktree\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := commitCodeSessionWorktree(session, "feat: worktree change"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("source\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(sourceDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(sourceDir, "commit", "-m", "source change"); err != nil {
		t.Fatal(err)
	}
	result, err := mergeCodeSessionWorktree(session)
	if err != nil || result.Status != "conflict" || len(result.ConflictFiles) != 1 || result.ConflictFiles[0] != "README.md" {
		t.Fatalf("unexpected conflict result: %#v, %v", result, err)
	}
	status, err := runCodeGit(sourceDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		t.Fatalf("source repository was not restored after conflict: %q, %v", status, err)
	}
}

func TestCommitCodeSessionWorktreeRequiresStagedChanges(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 33)
	if _, err := commitCodeSessionWorktree(session, "empty"); err == nil {
		t.Fatal("commit without staged changes should fail")
	}
}
