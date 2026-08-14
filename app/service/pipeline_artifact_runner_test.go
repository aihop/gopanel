package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestRunnerZipKeepsNestedRuntimeDependencies(t *testing.T) {
	source := filepath.Join(t.TempDir(), "runner-record")
	files := map[string]string{
		".output/server/index.mjs":                            "built",
		".output/server/node_modules/vue-router/package.json": "runtime-dependency",
		"node_modules/build-only/package.json":                "build-dependency",
	}
	for relative, content := range files {
		target := filepath.Join(source, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	archivePath := filepath.Join(t.TempDir(), "runner.zip")
	if err := createFilteredZipArchive(source, archivePath, true); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()

	entries := make(map[string]bool, len(archive.File))
	for _, entry := range archive.File {
		entries[entry.Name] = true
	}
	root := filepath.Base(source) + "/"
	if !entries[root+".output/server/node_modules/vue-router/package.json"] {
		t.Fatal("Runner ZIP omitted nested runtime dependency")
	}
	if entries[root+"node_modules/build-only/package.json"] {
		t.Fatal("Runner ZIP included root build dependencies")
	}
}

func TestSourceZipOmitsAllNodeModules(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	target := filepath.Join(source, ".output", "server", "node_modules", "vue-router", "package.json")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("dependency"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "source.zip")
	if err := createFilteredZipArchive(source, archivePath, false); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if entry.Name == filepath.Base(source)+"/.output/server/node_modules/vue-router/package.json" {
			t.Fatal("source ZIP unexpectedly included nested node_modules")
		}
	}
}
