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

func TestHostTerminalRejectedDuringCodeDelivery(t *testing.T) {
	database := withCodeGovernanceDB(t)
	if err := database.Create(&model.AIDevSession{
		ID: 921, UserID: 7, Status: codeSessionStatusDelivering, WorkDir: t.TempDir(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	manager := &hostTerminalManager{sessions: make(map[uint]*hostTerminal)}
	if _, err := manager.create(createHostTerminalRequest{Shell: "default", WorkDir: t.TempDir()}, 7, "127.0.0.1"); err == nil {
		t.Fatal("host terminal should be rejected while a Code session is delivering")
	}
}
