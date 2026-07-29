package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
)

func TestNormalizeAIProjectWorkDir(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "project")
	if err := os.Mkdir(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	claims := &token.CustomClaims{Role: constant.UserRoleSubAdmin, FileBaseDir: baseDir}

	got, err := normalizeAIProjectWorkDir(projectDir, claims)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("workDir = %q, want %q", got, want)
	}
	if _, err := normalizeAIProjectWorkDir(filepath.Dir(baseDir), claims); err == nil {
		t.Fatal("expected a directory outside the sub-admin workspace to be rejected")
	}
}

func TestNormalizeAIProjectWorkDirRejectsInvalidPaths(t *testing.T) {
	claims := &token.CustomClaims{Role: constant.UserRoleSuper}
	if _, err := normalizeAIProjectWorkDir("relative/path", claims); err == nil {
		t.Fatal("expected a relative path to be rejected")
	}
	filePath := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := normalizeAIProjectWorkDir(filePath, claims); err == nil {
		t.Fatal("expected a file path to be rejected")
	}
}
