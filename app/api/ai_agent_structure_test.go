package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListAISessionStructureUsesManagedSources(t *testing.T) {
	parent := t.TempDir()
	sourceDir := filepath.Join(parent, "source")
	workspaceDir := filepath.Join(parent, "project_7")
	if err := os.MkdirAll(filepath.Join(sourceDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "src", "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sourceDir, filepath.Join(workspaceDir, "source")); err != nil {
		t.Fatal(err)
	}
	manifest := aiProjectWorkspaceManifest{Version: 1, Sources: []aiProjectWorkspaceSource{{Path: sourceDir, LinkName: "source"}}}
	if err := writeAIProjectWorkspaceManifest(workspaceDir, manifest); err != nil {
		t.Fatal(err)
	}

	result, err := listAISessionStructure(workspaceDir, "source/src", []string{sourceDir})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Name != "main.go" || result.Entries[0].IsDir {
		t.Fatalf("unexpected structure entries: %#v", result.Entries)
	}
}

func TestListAISessionStructureRejectsTraversalAndEscapingSymlink(t *testing.T) {
	workDir := t.TempDir()
	outsideDir := t.TempDir()
	if _, err := listAISessionStructure(workDir, "../", nil); err == nil {
		t.Fatal("expected parent traversal to be rejected")
	}
	if err := os.Symlink(outsideDir, filepath.Join(workDir, "outside")); err != nil {
		t.Fatal(err)
	}
	result, err := listAISessionStructure(workDir, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range result.Entries {
		if entry.Name == "outside" {
			t.Fatal("expected escaping symlink to be hidden")
		}
	}
}

func TestListAISessionStructureHidesLargeDependencyDirectories(t *testing.T) {
	workDir := t.TempDir()
	for _, name := range []string{".git", "node_modules", "src"} {
		if err := os.Mkdir(filepath.Join(workDir, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	result, err := listAISessionStructure(workDir, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Name != "src" {
		t.Fatalf("unexpected structure entries: %#v", result.Entries)
	}
}

func TestAISessionFileReadWriteStaysInsideWorkspace(t *testing.T) {
	workDir := t.TempDir()
	filePath := filepath.Join(workDir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := readAISessionFile(workDir, "main.go", nil)
	if err != nil || result["content"] != "package main\n" {
		t.Fatalf("unexpected file result: %#v, %v", result, err)
	}
	if err := writeAISessionFile(workDir, "main.go", "package changed\n", nil); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filePath)
	if err != nil || string(content) != "package changed\n" {
		t.Fatalf("unexpected saved content: %q, %v", content, err)
	}
	if _, err := readAISessionFile(workDir, "../outside", nil); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestAISessionFileRejectsBinaryAndOversizedContent(t *testing.T) {
	workDir := t.TempDir()
	binaryPath := filepath.Join(workDir, "binary.dat")
	if err := os.WriteFile(binaryPath, []byte{'a', 0, 'b'}, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readAISessionFile(workDir, "binary.dat", nil); err == nil {
		t.Fatal("expected binary file to be rejected")
	}
	largeContent := strings.Repeat("x", maxAISessionFileSize+1)
	if err := writeAISessionFile(workDir, "binary.dat", largeContent, nil); err == nil {
		t.Fatal("expected oversized content to be rejected")
	}
}
