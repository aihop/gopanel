package api

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
)

func TestCodeProjectActiveTerminalCheckReportsUnavailableDatabase(t *testing.T) {
	oldDB := global.DB
	global.DB = nil
	t.Cleanup(func() { global.DB = oldDB })
	active, err := codeProjectHasActiveTerminal(&model.AIProject{WorkDir: t.TempDir()})
	if err == nil || active || !strings.Contains(err.Error(), "数据库尚未初始化") {
		t.Fatalf("unexpected terminal check: active=%v err=%v", active, err)
	}
}

func TestCodeProjectTerminalWorkDirUsesSourceDirectoryForAdmin(t *testing.T) {
	sourceDir := t.TempDir()
	project := &model.AIProject{SourceDirs: []string{sourceDir}, WorkDir: t.TempDir()}

	workDir, err := codeProjectTerminalWorkDir(project, 0, &token.CustomClaims{Role: constant.UserRoleAdmin})
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

	if _, err := codeProjectTerminalWorkDir(project, 0, claims); err == nil {
		t.Fatal("sub-admin should not receive a native host shell")
	}
}

// 交付在独立的集成 Worktree 中进行，不独占源仓工作区，
// 因此交付进行中照常可以打开宿主终端；本地快进失败只会被降级记录，不会让交付失败。
func TestHostTerminalStaysAvailableDuringCodeDelivery(t *testing.T) {
	database := withCodeGovernanceDB(t)
	if err := database.Create(&model.AIDevSession{
		ID: 921, UserID: 7, Status: codeSessionStatusDelivering, WorkDir: t.TempDir(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	manager := &hostTerminalManager{sessions: make(map[uint]*hostTerminal)}
	record, err := manager.create(createHostTerminalRequest{Shell: "default", WorkDir: t.TempDir()}, 7, "127.0.0.1")
	if err != nil {
		t.Fatalf("host terminal should stay available while a Code session is delivering: %v", err)
	}
	t.Cleanup(func() { manager.stop(record.ID) })
}
