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

func TestCodeProjectTerminalWorkDirUsesSingleRepositorySessionWorktree(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session, _ := createDeliveryWorktree(t, 922)
	session.ProjectID = 922
	project := &model.AIProject{ID: session.ProjectID, Name: "single", CreatorID: session.UserID}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}

	workDir, err := codeProjectTerminalWorkDir(
		project, session.ID, &token.CustomClaims{UserId: session.UserID, Role: constant.UserRoleAdmin},
	)
	if err != nil || workDir != session.WorkDir {
		t.Fatalf("workDir = %q, want session Worktree %q: %v", workDir, session.WorkDir, err)
	}
	branch, err := runCodeGit(workDir, "branch", "--show-current")
	if err != nil || strings.TrimSpace(branch) != session.WorktreeBranch {
		t.Fatalf("terminal branch = %q, want %q: %v", branch, session.WorktreeBranch, err)
	}
}

func TestCodeProjectTerminalWorkDirUsesMultiRepositorySessionWorkspace(t *testing.T) {
	session, project, _ := createMultiRepositorySession(t, 923)
	workDir, err := codeProjectTerminalWorkDir(
		project, session.ID, &token.CustomClaims{UserId: session.UserID, Role: constant.UserRoleAdmin},
	)
	if err != nil || workDir != session.WorkDir {
		t.Fatalf("workDir = %q, want multi-repository workspace %q: %v", workDir, session.WorkDir, err)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("session repositories unavailable: %#v, %v", repositories, err)
	}
	for _, repository := range repositories {
		branch, branchErr := runCodeGit(repository.WorktreeDir, "branch", "--show-current")
		if branchErr != nil || strings.TrimSpace(branch) != repository.Branch {
			t.Fatalf("repository %s branch = %q, want %q: %v", repository.LinkName, branch, repository.Branch, branchErr)
		}
	}
}

func TestCodeProjectTerminalWorkDirUsesDirectSessionWorkspace(t *testing.T) {
	database := withCodeGovernanceDB(t)
	sourceDir := createCodeGitRepository(t)
	project := &model.AIProject{ID: 926, Name: "direct", CreatorID: 7, SourceDirs: []string{sourceDir}}
	session := &model.AIDevSession{
		ID: 926, UserID: project.CreatorID, ProjectID: project.ID, WorkDir: sourceDir,
		IsolationMode: codeIsolationDirect, Status: codeSessionStatusActive,
	}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	workDir, err := codeProjectTerminalWorkDir(
		project, session.ID, &token.CustomClaims{UserId: session.UserID, Role: constant.UserRoleAdmin},
	)
	if err != nil || workDir != sourceDir {
		t.Fatalf("workDir = %q, want direct source %q: %v", workDir, sourceDir, err)
	}
}

func TestCodeProjectTerminalWorkDirDoesNotFallbackForInvalidSession(t *testing.T) {
	database := withCodeGovernanceDB(t)
	projectDir := t.TempDir()
	project := &model.AIProject{ID: 924, Name: "project", CreatorID: 7, SourceDirs: []string{projectDir}}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	claims := &token.CustomClaims{UserId: 7, Role: constant.UserRoleAdmin}
	if workDir, err := codeProjectTerminalWorkDir(project, 999, claims); err == nil || workDir != "" {
		t.Fatalf("missing session fell back to project directory: workDir=%q err=%v", workDir, err)
	}
	foreign := &model.AIDevSession{ID: 924, UserID: 7, ProjectID: 925, WorkDir: projectDir}
	if err := database.Create(foreign).Error; err != nil {
		t.Fatal(err)
	}
	if workDir, err := codeProjectTerminalWorkDir(project, foreign.ID, claims); err == nil || workDir != "" {
		t.Fatalf("foreign session fell back to project directory: workDir=%q err=%v", workDir, err)
	}
}

func TestCodeProjectTerminalSessionIDRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"0", "-1", "invalid"} {
		if sessionID, err := codeProjectTerminalSessionID(value); err == nil || sessionID != 0 {
			t.Fatalf("session_id %q accepted as %d: %v", value, sessionID, err)
		}
	}
	if sessionID, err := codeProjectTerminalSessionID(""); err != nil || sessionID != 0 {
		t.Fatalf("empty project terminal session = %d: %v", sessionID, err)
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
