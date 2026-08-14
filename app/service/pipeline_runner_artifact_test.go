package service

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestExtractRunnerArtifactTarKeepsBuildAndFiltersRuntimeData(t *testing.T) {
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	files := map[string]string{
		"app/.output/server/index.mjs":  "built",
		"app/package.json":              "{}",
		"app/node_modules/pkg/index.js": "dependency",
		"app/storage/database.sqlite":   "runtime-data",
		"app/.gopanel_shims/docker":     "shim",
	}
	for name, content := range files {
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	destination := t.TempDir()
	excluded := map[string]struct{}{"storage": {}}
	if err := extractRunnerArtifactTar(context.Background(), &archive, destination, "app", excluded); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(destination, ".output", "server", "index.mjs"))
	if err != nil || string(content) != "built" {
		t.Fatalf("built artifact missing: %q, %v", content, err)
	}
	for _, relative := range []string{"node_modules", "storage", ".gopanel_shims"} {
		if _, err := os.Stat(filepath.Join(destination, relative)); !os.IsNotExist(err) {
			t.Fatalf("excluded path %s was exported: %v", relative, err)
		}
	}
}

func TestExtractRunnerArtifactTarRejectsTraversal(t *testing.T) {
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	if err := writer.WriteHeader(&tar.Header{Name: "app/../../escape.txt", Mode: 0644, Size: 1, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extractRunnerArtifactTar(context.Background(), &archive, t.TempDir(), "app", nil); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestRunnerArtifactExcludedPathsUsesPersistentConfig(t *testing.T) {
	pipeline := &model.Pipeline{RunnerConfig: `{"persistentPaths":["storage","/var/www/app/uploads"]}`}
	excluded := runnerArtifactExcludedPaths(pipeline, "/var/www/app")
	for _, path := range []string{"storage", "uploads"} {
		if _, ok := excluded[path]; !ok {
			t.Fatalf("persistent path %s not excluded: %#v", path, excluded)
		}
	}
}

func TestValidatePipelineRunnerArtifactRequiresConfiguredStartEntry(t *testing.T) {
	pipeline := &model.Pipeline{RunnerConfig: `{"startCommand":"node .output/server/index.mjs"}`}
	artifactDir := t.TempDir()
	if err := validatePipelineRunnerArtifact(pipeline, artifactDir); err == nil {
		t.Fatal("expected missing Nuxt entry to reject the artifact")
	}
	entry := filepath.Join(artifactDir, ".output", "server", "index.mjs")
	if err := os.MkdirAll(filepath.Dir(entry), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, []byte("built"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validatePipelineRunnerArtifact(pipeline, artifactDir); err != nil {
		t.Fatal(err)
	}
}

func TestRunnerStartCommandArtifactPathHandlesRelativeEntry(t *testing.T) {
	if got := runnerStartCommandArtifactPath("node ./.output/server/index.mjs"); got != ".output/server/index.mjs" {
		t.Fatalf("artifact path = %q", got)
	}
	if got := runnerStartCommandArtifactPath("python server/app.py"); got != "server/app.py" {
		t.Fatalf("Python artifact path = %q", got)
	}
	if got := runnerStartCommandArtifactPath("./app --port 3000"); got != "app" {
		t.Fatalf("binary artifact path = %q", got)
	}
	if got := runnerStartCommandArtifactPath("npm run start"); got != "" {
		t.Fatalf("script command should use HTTP readiness only, got %q", got)
	}
}
