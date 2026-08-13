package service

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSystemLogTailBoundsPlainFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gopanel-2026-08-12.log")
	if err := os.WriteFile(path, []byte("prefix-0123456789"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := readSystemLogTail(dir, "2026-08-12", 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "0123456789" || !result.Truncated || result.ReturnedBytes != 10 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestReadSystemLogTailFindsGzipArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gopanel-2026-08-11.log.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := gzip.NewWriter(file)
	if _, err := writer.Write([]byte(strings.Repeat("a", 20) + "tail")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	result, err := readSystemLogTail(dir, "2026-08-11", 8)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "aaaatail" || !result.Truncated || result.FileName != filepath.Base(path) {
		t.Fatalf("unexpected gzip result: %#v", result)
	}
}

func TestResolveSystemLogPathRejectsTraversal(t *testing.T) {
	if _, err := resolveSystemLogPath(t.TempDir(), "../../etc/passwd"); err == nil {
		t.Fatal("expected invalid log name")
	}
}
