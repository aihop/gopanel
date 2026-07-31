package api

import (
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
)

func TestCodeProjectTerminalWorkDirUsesSourceDirectoryForAdmin(t *testing.T) {
	sourceDir := t.TempDir()
	project := &model.AIProject{SourceDirs: []string{sourceDir}, WorkDir: t.TempDir()}

	workDir, err := codeProjectTerminalWorkDir(project, &token.CustomClaims{Role: constant.UserRoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	expected, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	if workDir != expected {
		t.Fatalf("workDir = %q, want source directory %q", workDir, expected)
	}
}

func TestCodeProjectTerminalWorkDirRejectsSubAdmin(t *testing.T) {
	project := &model.AIProject{SourceDirs: []string{t.TempDir()}}
	claims := &token.CustomClaims{Role: constant.UserRoleSubAdmin, FileBaseDir: t.TempDir()}

	if _, err := codeProjectTerminalWorkDir(project, claims); err == nil {
		t.Fatal("sub-admin should not receive a native host shell")
	}
}
