package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestPrepareCodeProjectSnapshotCopiesCurrentContentAndExcludesCaches(t *testing.T) {
	database := flowTestDatabase(t)
	source := t.TempDir()
	gitInRepo(t, source, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(source, "tracked.txt"), []byte("committed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, source, "add", "tracked.txt")
	gitInRepo(t, source, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-q", "-m", "initial")
	commit := gitInRepo(t, source, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(source, "tracked.txt"), []byte("uncommitted\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "untracked.txt"), []byte("local result\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, ".env"), []byte("SECRET=local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "node_modules", "cached"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "node_modules", "cached", "index.js"), []byte("cache"), 0644); err != nil {
		t.Fatal(err)
	}

	project := model.AIProject{Name: "local", CreatorID: 7, SourceDirs: []string{source}, PrimaryRepository: source}
	if err := database.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	pipeline := &model.Pipeline{ID: 91, PipelineKey: "local-code", SourceType: "code", CodeProjectID: project.ID}
	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "stale.txt"), []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	logger := GetPipelineLogger(991001)
	defer RemovePipelineLogger(991001)
	actualCommit, digest, err := NewPipelineService(database).prepareCodeProjectSnapshot(context.Background(), logger, pipeline, workspace, commit)
	if err != nil {
		t.Fatal(err)
	}
	if actualCommit != commit || !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("snapshot identity = %q, %q", actualCommit, digest)
	}
	content, err := os.ReadFile(filepath.Join(workspace, "tracked.txt"))
	if err != nil || string(content) != "uncommitted\n" {
		t.Fatalf("current tracked content was not copied: %q, %v", content, err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "untracked.txt")); err != nil {
		t.Fatalf("untracked content was not copied: %v", err)
	}
	for _, excluded := range []string{".git", ".env", "node_modules", "stale.txt"} {
		if _, err := os.Stat(filepath.Join(workspace, excluded)); !os.IsNotExist(err) {
			t.Fatalf("excluded or stale path %s remains: %v", excluded, err)
		}
	}
}

func TestPrepareCodeProjectSnapshotSupportsLocalDirectoryWithoutGit(t *testing.T) {
	database := flowTestDatabase(t)
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "index.html"), []byte("local"), 0644); err != nil {
		t.Fatal(err)
	}
	project := model.AIProject{Name: "static", CreatorID: 7, SourceDirs: []string{source}}
	if err := database.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	pipeline := &model.Pipeline{ID: 92, PipelineKey: "local-static", SourceType: "code", CodeProjectID: project.ID}
	logger := GetPipelineLogger(991002)
	defer RemovePipelineLogger(991002)
	commit, digest, err := NewPipelineService(database).prepareCodeProjectSnapshot(
		context.Background(), logger, pipeline, filepath.Join(t.TempDir(), "workspace"), "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if commit != "" || !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("local directory identity = %q, %q", commit, digest)
	}
}

func TestCopyCodeProjectSourcesMaterializesMultipleDirectories(t *testing.T) {
	first := filepath.Join(t.TempDir(), "api")
	second := filepath.Join(t.TempDir(), "web")
	for path, content := range map[string]string{first: "api", second: "web"} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "source.txt"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	destination := t.TempDir()
	if err := copyCodeProjectSources(context.Background(), []string{first, second}, destination); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"api": "api", "web": "web"} {
		copied, err := os.ReadFile(filepath.Join(destination, name, "source.txt"))
		if err != nil || string(copied) != content {
			t.Fatalf("materialized %s = %q, %v", name, copied, err)
		}
		info, err := os.Lstat(filepath.Join(destination, name))
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("%s should be a real directory: %v, %v", name, info, err)
		}
	}
}

func TestCopyCodeProjectSourcesRejectsEscapingSymlink(t *testing.T) {
	source := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(source, "outside")); err != nil {
		t.Fatal(err)
	}
	if err := copyCodeProjectSources(context.Background(), []string{source}, t.TempDir()); err == nil {
		t.Fatal("expected an escaping symlink to be rejected")
	}
}
