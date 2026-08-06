package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

func createInitializingCodeSession(t *testing.T, project *model.AIProject) *model.AIDevSession {
	t.Helper()
	session := &model.AIDevSession{
		UserID: project.CreatorID, ProjectID: project.ID, Title: "initializing session",
		AgentName: "codex", WorkDir: project.WorkDir, Status: codeSessionStatusInitializing,
		CurrentStage: codeSessionStageSyncingBase, ApprovalPolicy: codeApprovalPolicySafeAuto,
	}
	if err := global.DB.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	return session
}

func TestInitializeCodeSessionCreatesWorktreeAndTask(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	repository := createCodeGitRepository(t)
	project := codeProjectForRepository(t, repository)
	project.Name, project.CreatorID, project.WorkDir = "project", 7, repository
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := createInitializingCodeSession(t, project)

	if err := initializeCodeSession(session.ID); err != nil {
		t.Fatal(err)
	}
	var stored model.AIDevSession
	if err := database.First(&stored, session.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != codeSessionStatusActive || stored.CurrentStage != "idle" || stored.LastTaskID == 0 || stored.WorktreeBranch == "" {
		t.Fatalf("initialized session = %#v", stored)
	}
	var task model.AITask
	if err := database.First(&task, stored.LastTaskID).Error; err != nil || task.WorkDir != stored.WorkDir {
		t.Fatalf("initialized task = %#v, err = %v", task, err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(&stored) })
}

func TestInitializeCodeSessionHonorsIncludeUncommittedFalse(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	repository := createCodeGitRepository(t)
	project := codeProjectForRepository(t, repository)
	project.Name, project.CreatorID, project.WorkDir = "project", 7, repository
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "uncommitted.txt"), []byte("dirty"), 0600); err != nil {
		t.Fatal(err)
	}
	includeUncommitted := false
	session := createInitializingCodeSession(t, project)
	if err := database.Model(session).Update("include_uncommitted", includeUncommitted).Error; err != nil {
		t.Fatal(err)
	}

	if err := initializeCodeSession(session.ID); err == nil || !strings.Contains(err.Error(), "未提交变更") {
		t.Fatalf("initialization should reject dirty source when snapshots are disabled: %v", err)
	}
	var stored model.AIDevSession
	if err := database.First(&stored, session.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != codeSessionStatusFailed || stored.IncludeUncommitted == nil || *stored.IncludeUncommitted {
		t.Fatalf("stored session did not preserve includeUncommitted=false: %#v", stored)
	}
}

func TestCodeSessionIncludesUncommittedDefaultsToLegacyBehavior(t *testing.T) {
	includeUncommitted := false
	if !codeSessionIncludesUncommitted(&model.AIDevSession{}) {
		t.Fatal("legacy sessions should include uncommitted changes")
	}
	if codeSessionIncludesUncommitted(&model.AIDevSession{IncludeUncommitted: &includeUncommitted}) {
		t.Fatal("explicit false should disable uncommitted snapshots")
	}
}

func TestInitializeCodeSessionPersistsFailureWithoutTask(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	repository, _ := createCodeRemoteRepository(t)
	if _, err := runCodeGit(repository, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git")); err != nil {
		t.Fatal(err)
	}
	project := codeProjectForRepository(t, repository)
	project.Name, project.CreatorID, project.WorkDir = "project", 7, repository
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	session := createInitializingCodeSession(t, project)

	if err := initializeCodeSession(session.ID); err == nil {
		t.Fatal("missing remote should fail initialization")
	}
	var stored model.AIDevSession
	if err := database.First(&stored, session.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != codeSessionStatusFailed || stored.CurrentStage != codeSessionStageInitializationFailed || strings.TrimSpace(stored.InitializationErr) == "" {
		t.Fatalf("failed session = %#v", stored)
	}
	var taskCount int64
	if err := database.Model(&model.AITask{}).Where("session_id = ?", session.ID).Count(&taskCount).Error; err != nil || taskCount != 0 {
		t.Fatalf("failed initialization created tasks: count=%d err=%v", taskCount, err)
	}
}

func TestInitializingCodeSessionRejectsDevelopment(t *testing.T) {
	err := validateCodeSessionDevelopmentOpen(&model.AIDevSession{Status: codeSessionStatusInitializing})
	if err == nil || !strings.Contains(err.Error(), "正在同步远端") {
		t.Fatalf("initializing session should reject development: %v", err)
	}
	err = validateCodeSessionDevelopmentOpen(&model.AIDevSession{Status: codeSessionStatusFailed})
	if err == nil || !strings.Contains(err.Error(), "初始化失败") {
		t.Fatalf("failed session should reject development: %v", err)
	}
}

func TestInitializeCodeSessionIgnoresCompletedSession(t *testing.T) {
	withCodeGovernanceDB(t)
	err := initializeCodeSession(999999)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing session error = %v", err)
	}
}
