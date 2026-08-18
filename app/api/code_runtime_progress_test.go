package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestLoadCodeRuntimeGitChangesKeepsCommittedAndWorkingFiles(t *testing.T) {
	repositoryDir := t.TempDir()
	runCodeRuntimeGit(t, repositoryDir, "init")
	runCodeRuntimeGit(t, repositoryDir, "config", "user.email", "runtime@example.com")
	runCodeRuntimeGit(t, repositoryDir, "config", "user.name", "Runtime Test")
	writeCodeRuntimeTestFile(t, repositoryDir, "existing.txt", "base\n")
	runCodeRuntimeGit(t, repositoryDir, "add", "existing.txt")
	runCodeRuntimeGit(t, repositoryDir, "commit", "-m", "base")
	baseCommit := strings.TrimSpace(runCodeRuntimeGit(t, repositoryDir, "rev-parse", "HEAD"))

	writeCodeRuntimeTestFile(t, repositoryDir, "existing.txt", "base\ncommitted\n")
	runCodeRuntimeGit(t, repositoryDir, "add", "existing.txt")
	runCodeRuntimeGit(t, repositoryDir, "commit", "-m", "task change")
	writeCodeRuntimeTestFile(t, repositoryDir, "existing.txt", "base\ncommitted\nworking\n")
	writeCodeRuntimeTestFile(t, repositoryDir, "untracked.txt", "new\n")

	session := &model.AIDevSession{
		WorkDir: repositoryDir, SourceWorkDir: repositoryDir, WorktreeBranch: "task/runtime",
		BaseCommit: baseCommit, IsolationMode: codeIsolationSingleWorktree,
	}
	files, changedFiles, additions, deletions, available := loadCodeRuntimeGitChanges(session)
	if !available || changedFiles != 2 {
		t.Fatalf("unexpected runtime git summary: available=%v files=%d paths=%v", available, changedFiles, files)
	}
	if strings.Join(files, ",") != "existing.txt,untracked.txt" {
		t.Fatalf("unexpected changed paths: %v", files)
	}
	if additions != 2 || deletions != 0 {
		t.Fatalf("unexpected line stats: +%d -%d", additions, deletions)
	}
}

func runCodeRuntimeGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	output, err := runCodeGit(root, args...)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func writeCodeRuntimeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
