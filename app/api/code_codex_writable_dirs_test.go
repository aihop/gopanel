package api

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestResolveCodexWritableDirsResolvesSymlinksAndDeduplicates(t *testing.T) {
	realDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "project")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}
	expectedDir, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatal(err)
	}

	resolvedDirs, err := resolveCodexWritableDirs([]string{linkDir, realDir})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resolvedDirs, []string{expectedDir}) {
		t.Fatalf("resolved writable dirs = %#v, want %#v", resolvedDirs, []string{expectedDir})
	}
}

func TestAddCodexWritableDirArgsBeforeSubcommand(t *testing.T) {
	args := []string{"--ask-for-approval", "on-request", "--sandbox", "workspace-write", "exec", "--json", "prompt"}
	got := addCodexWritableDirArgs(args, []string{"/code/one", "/code/two"})
	want := []string{
		"--ask-for-approval", "on-request",
		"--sandbox", "workspace-write",
		"--add-dir", "/code/one",
		"--add-dir", "/code/two",
		"exec", "--json", "prompt",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Codex args = %#v, want %#v", got, want)
	}
}

func TestAddCodexWritableDirArgsToInteractiveCommand(t *testing.T) {
	args := []string{"--sandbox", "workspace-write", "--cd", "/workspace"}
	got := addCodexWritableDirArgs(args, []string{"/code/project"})
	want := []string{"--sandbox", "workspace-write", "--cd", "/workspace", "--add-dir", "/code/project"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("interactive Codex args = %#v, want %#v", got, want)
	}
}

func TestCodexWritableDirsSkipIsolatedWorktree(t *testing.T) {
	session := &model.AIDevSession{
		ProjectID:      7,
		SourceWorkDir:  "/code/project",
		WorktreeBranch: "gopanel/code-1",
	}
	writableDirs, err := codexWritableDirsForSession(session)
	if err != nil {
		t.Fatal(err)
	}
	if len(writableDirs) != 0 {
		t.Fatalf("isolated worktree writable dirs = %#v, want none", writableDirs)
	}
}

func TestCodexWritableDirsForProjectSession(t *testing.T) {
	realDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "project")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}
	expectedDir, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatal(err)
	}
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "project.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIGroup{}); err != nil {
		t.Fatal(err)
	}
	oldDatabase := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = oldDatabase })
	project := &model.AIGroup{Name: "project", SourceDirs: []string{linkDir}, CreatorID: 1}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}

	writableDirs, err := codexWritableDirsForSession(&model.AIDevSession{ProjectID: project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(writableDirs, []string{expectedDir}) {
		t.Fatalf("project writable dirs = %#v, want %#v", writableDirs, []string{expectedDir})
	}
}
