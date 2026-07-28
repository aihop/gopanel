package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePathWithinBase(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "user")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}

	allowed := []string{base, filepath.Join(base, "existing"), filepath.Join(base, "new", "file.txt")}
	for _, path := range allowed {
		if err := ValidatePathWithinBase(base, path); err != nil {
			t.Fatalf("expected %s to be allowed: %v", path, err)
		}
	}

	denied := []string{filepath.Join(root, "user2"), root, filepath.Join(base, "..", "outside")}
	for _, path := range denied {
		if err := ValidatePathWithinBase(base, path); err == nil {
			t.Fatalf("expected %s to be denied", path)
		}
	}
}

func TestValidatePathWithinBaseRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "user")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePathWithinBase(base, filepath.Join(link, "secret.txt")); err == nil {
		t.Fatal("expected symlink escape to be denied")
	}
}
