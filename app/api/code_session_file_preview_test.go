package api

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

var png1x1 = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54,
	0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01,
	0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44,
	0xae, 0x42, 0x60, 0x82,
}

func TestConversationImageContentType(t *testing.T) {
	if got := conversationImageContentType("shot.PNG"); got != "image/png" {
		t.Fatalf("png type = %q", got)
	}
	if got := conversationImageContentType("photo.jpeg"); got != "image/jpeg" {
		t.Fatalf("jpeg type = %q", got)
	}
	if conversationImageContentType("icon.svg") != "" {
		t.Fatal("svg must not preview as an image")
	}
	if conversationImageContentType("main.go") != "" {
		t.Fatal("source files must not preview as images")
	}
}

func TestReadAISessionImagePreview(t *testing.T) {
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "shot.png"), png1x1, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := readAISessionImagePreview(workDir, "shot.png", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result["path"] != "shot.png" || result["contentType"] != "image/png" {
		t.Fatalf("unexpected preview meta: %#v", result)
	}
	decoded, err := base64.StdEncoding.DecodeString(result["content"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(png1x1) {
		t.Fatalf("decoded size = %d, want %d", len(decoded), len(png1x1))
	}

	if _, err := readAISessionImagePreview(workDir, "main.go", nil); err == nil {
		t.Fatal("expected a non-image file to be rejected")
	}
	if _, err := readAISessionImagePreview(workDir, "../shot.png", nil); err == nil {
		t.Fatal("expected a path escape to be rejected")
	}
}

func TestResolveAISessionFilePathKeepsEditorSizeLimit(t *testing.T) {
	workDir := t.TempDir()
	oversized := make([]byte, maxAISessionFileSize+1)
	if err := os.WriteFile(filepath.Join(workDir, "big.go"), oversized, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolveAISessionFilePath(workDir, "big.go", nil); err == nil {
		t.Fatal("expected the editor path helper to reject files over 2 MB")
	}
}
