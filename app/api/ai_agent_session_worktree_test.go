package api

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestShouldCreateCodeSessionWorktreeSkipsProjectTerminal(t *testing.T) {
	project := &model.AIProject{SourceDirs: []string{createCodeGitRepository(t)}}
	if shouldCreateCodeSessionWorktree("terminal", true, project) {
		t.Fatal("project terminal should use the project directory directly")
	}
	if !shouldCreateCodeSessionWorktree("codex", false, project) {
		t.Fatal("AI project session should retain automatic worktree isolation")
	}
	if !shouldCreateCodeSessionWorktree("codex", true, nil) {
		t.Fatal("explicit isolation should be preserved outside projects")
	}
	if shouldCreateCodeSessionWorktree("codex", false, nil) {
		t.Fatal("non-project session should not be isolated by default")
	}
}
