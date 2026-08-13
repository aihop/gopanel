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
