package api

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestChunkUploadDirIsScoped(t *testing.T) {
	tmpDir := t.TempDir()
	first := chunkUploadDir(tmpDir, 1, "/srv/project-a", "app.tar.gz")
	if first == chunkUploadDir(tmpDir, 2, "/srv/project-a", "app.tar.gz") {
		t.Fatal("different users must not share a chunk directory")
	}
	if first == chunkUploadDir(tmpDir, 1, "/srv/project-b", "app.tar.gz") {
		t.Fatal("different destinations must not share a chunk directory")
	}
}

func TestMergeChunksDoesNotOverwriteExistingFile(t *testing.T) {
	dstDir := t.TempDir()
	dstFile := filepath.Join(dstDir, "data.txt")
	if err := os.WriteFile(dstFile, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	chunkDir := createChunkFixture(t, "data.txt", []string{"new"})
	if err := mergeChunks("data.txt", chunkDir, dstDir, 1, false); err == nil {
		t.Fatal("expected overwrite=false to reject an existing destination")
	}
	assertFileContent(t, dstFile, "existing")
}

func TestMergeChunksPreservesExistingFileOnFailure(t *testing.T) {
	dstDir := t.TempDir()
	dstFile := filepath.Join(dstDir, "data.txt")
	if err := os.WriteFile(dstFile, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	chunkDir := createChunkFixture(t, "data.txt", []string{"partial"})
	if err := mergeChunks("data.txt", chunkDir, dstDir, 2, true); err == nil {
		t.Fatal("expected a missing chunk to fail the merge")
	}
	assertFileContent(t, dstFile, "existing")
}

func TestMergeChunksReplacesFileAfterCompleteMerge(t *testing.T) {
	dstDir := t.TempDir()
	dstFile := filepath.Join(dstDir, "data.txt")
	if err := os.WriteFile(dstFile, []byte("old-long-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	chunkDir := createChunkFixture(t, "data.txt", []string{"new", "-data"})
	if err := mergeChunks("data.txt", chunkDir, dstDir, 2, true); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, dstFile, "new-data")
}

func createChunkFixture(t *testing.T, fileName string, chunks []string) string {
	t.Helper()
	dir := t.TempDir()
	for index, content := range chunks {
		path := filepath.Join(dir, fmt.Sprintf("%s.%d", fileName, index))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("content = %q, want %q", content, want)
	}
}
