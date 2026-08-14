package service

import (
	"archive/zip"
	"io"
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

// #213 现场：npm 嵌套去重在 .output/server/node_modules 下留了一个指向目录的
// 软链接，归档时被当成普通文件 os.Open，读目录 fd 直接报 "is a directory"，
// 整个流水线在构建成功之后才失败在留档这一步。
func TestRunnerZipArchivesSymlinkToDirectory(t *testing.T) {
	source := filepath.Join(t.TempDir(), "runner-record")
	realDir := filepath.Join(source, ".output/server/node_modules/entities")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "package.json"), []byte("real"), 0644); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(source, ".output/server/node_modules/@vue/compiler-core/node_modules")
	if err := os.MkdirAll(linkDir, 0755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(linkDir, "entities")
	if err := os.Symlink("../../../entities", linkPath); err != nil {
		t.Skipf("symlink unsupported on this platform: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "runner.zip")
	if err := createFilteredZipArchive(source, archivePath, true); err != nil {
		t.Fatalf("archiving a symlinked directory must not fail: %v", err)
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()

	root := filepath.Base(source) + "/"
	wanted := root + ".output/server/node_modules/@vue/compiler-core/node_modules/entities"
	var entry *zip.File
	for _, candidate := range archive.File {
		if candidate.Name == wanted {
			entry = candidate
			break
		}
	}
	if entry == nil {
		t.Fatalf("symlink entry missing from archive: %s", wanted)
	}
	if entry.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink entry lost its mode bit: %v", entry.Mode())
	}
	reader, err := entry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	target, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(target) != "../../../entities" {
		t.Fatalf("symlink target not preserved, got %q", target)
	}
}
