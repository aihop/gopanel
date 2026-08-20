package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchAISessionStructureFindsNameAndContent(t *testing.T) {
	workDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workDir, "dist"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workDir, "node_modules", "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "src", "login.go"), []byte("package auth\nfunc Login() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "src", "util.go"), []byte("package auth\nfunc Helper() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "dist", "bundle.js"), []byte("func Login() {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "node_modules", "lib", "index.js"), []byte("func Login() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := searchAISessionStructure(workDir, "Login", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Hits) != 1 || result.Hits[0].Path != "src/login.go" || result.Hits[0].Kind != "name" || result.Hits[0].Line != 2 {
		t.Fatalf("hits = %#v", result.Hits)
	}

	contentHits, err := searchAISessionStructure(workDir, "func Login", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(contentHits.Hits) != 1 || contentHits.Hits[0].Kind != "content" || contentHits.Hits[0].Path != "src/login.go" {
		t.Fatalf("content hits = %#v", contentHits.Hits)
	}
}

func TestSearchAISessionStructureSkipsShortQuery(t *testing.T) {
	result, err := searchAISessionStructure(t.TempDir(), "a", nil)
	if err != nil || len(result.Hits) != 0 {
		t.Fatalf("short query should not search: %#v %v", result, err)
	}
}

func TestStructureSearchPreview(t *testing.T) {
	line, preview := structureSearchPreview("package main\nfunc Login() {}\n", "login")
	if line != 2 || preview != "func Login() {}" {
		t.Fatalf("line=%d preview=%q", line, preview)
	}
}
