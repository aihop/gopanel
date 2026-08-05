package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func TestLoadCodeAttentionItemsPrioritizesApproval(t *testing.T) {
	database := withCodeGovernanceDB(t)
	if err := database.AutoMigrate(&model.AIApproval{}); err != nil {
		t.Fatal(err)
	}
	session := model.AIDevSession{UserID: 7, ProjectID: 1, Title: "session", Status: codeSessionStatusFailed, CurrentStage: codeSessionStageInitializationFailed, InitializationErr: "clone failed"}
	if err := database.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	approval := model.AIApproval{SessionID: session.ID, InstructionID: 9, RequestUserID: 7, Title: "approval", Content: "git push origin main", Status: "pending"}
	if err := database.Create(&approval).Error; err != nil {
		t.Fatal(err)
	}
	items, err := loadCodeAttentionItems(7, 100)
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %#v, err = %v", items, err)
	}
	if items[0].Type != "approval" || items[0].ApprovalID != approval.ID || len(items[0].Actions) != 2 {
		t.Fatalf("attention = %#v", items[0])
	}
}

func TestLoadCodeAttentionItemsReturnsRecoverableInitialization(t *testing.T) {
	database := withCodeGovernanceDB(t)
	if err := database.AutoMigrate(&model.AIApproval{}); err != nil {
		t.Fatal(err)
	}
	session := model.AIDevSession{UserID: 11, ProjectID: 2, Title: "session", Status: codeSessionStatusFailed, CurrentStage: codeSessionStageInitializationFailed, InitializationErr: "remote unavailable"}
	if err := database.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	items, err := loadCodeAttentionItems(11, 100)
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %#v, err = %v", items, err)
	}
	if items[0].Type != "initialization_failed" || len(items[0].Actions) != 1 || items[0].Actions[0].Type != "retry_initialization" {
		t.Fatalf("attention = %#v", items[0])
	}
}

func TestLoadCodeAttentionItemsLimitsAfterFiltering(t *testing.T) {
	database := withCodeGovernanceDB(t)
	if err := database.AutoMigrate(&model.AIApproval{}); err != nil {
		t.Fatal(err)
	}
	attentionSession := model.AIDevSession{UserID: 12, ProjectID: 2, Title: "needs attention", Status: "active"}
	if err := database.Create(&attentionSession).Error; err != nil {
		t.Fatal(err)
	}
	approval := model.AIApproval{SessionID: attentionSession.ID, RequestUserID: 12, Title: "approval", Content: "deploy", Status: "pending"}
	if err := database.Create(&approval).Error; err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 3; index++ {
		session := model.AIDevSession{UserID: 12, ProjectID: 2, Title: fmt.Sprintf("recent %d", index), Status: "active"}
		if err := database.Create(&session).Error; err != nil {
			t.Fatal(err)
		}
	}
	items, err := loadCodeAttentionItems(12, 1)
	if err != nil || len(items) != 1 || items[0].ApprovalID != approval.ID {
		t.Fatalf("items = %#v, err = %v", items, err)
	}
}

func TestLoadCodeAttentionItemsUsesLatestRunAndUserScope(t *testing.T) {
	database := withCodeGovernanceDB(t)
	if err := database.AutoMigrate(&model.AIApproval{}); err != nil {
		t.Fatal(err)
	}
	session := model.AIDevSession{UserID: 3, ProjectID: 1, Title: "session", Status: "active", CurrentStage: "completed"}
	other := model.AIDevSession{UserID: 4, ProjectID: 1, Title: "other", Status: "failed", CurrentStage: "failed"}
	if err := database.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	older := time.Now().Add(-time.Minute)
	runs := []model.AIExecutionRun{
		{CreatedAt: older, SessionID: session.ID, ExecutorID: "codex", Prompt: "one", Status: "failed", ErrorMessage: "old failure", StartedAt: older},
		{CreatedAt: time.Now(), SessionID: session.ID, ExecutorID: "codex", Prompt: "two", Status: "completed", StartedAt: time.Now()},
		{SessionID: other.ID, ExecutorID: "codex", Prompt: "other", Status: "failed", ErrorMessage: "private", StartedAt: time.Now()},
	}
	if err := database.Create(&runs).Error; err != nil {
		t.Fatal(err)
	}
	items, err := loadCodeAttentionItems(3, 100)
	if err != nil || len(items) != 0 {
		t.Fatalf("items = %#v, err = %v", items, err)
	}
}

func TestUseMobileCodeAttentionPaths(t *testing.T) {
	items := []codeAttentionItem{{Actions: []codeAttentionAction{{Path: "/api/code/approvals/9/approve"}, {Type: "open_session"}}}}
	useMobileCodeAttentionPaths(items)
	if items[0].Actions[0].Path != "/api/mobile/app/approvals/9/approve" || items[0].Actions[1].Path != "" {
		t.Fatalf("mobile actions = %#v", items[0].Actions)
	}
}

func TestBuildCodeAttentionItemIgnoresOlderFailedTask(t *testing.T) {
	session := model.AIDevSession{ID: 3, LastTaskID: 12}
	run := model.AIExecutionRun{ID: 4, TaskID: 11, Status: "failed", ErrorMessage: "old failure"}
	if item := buildCodeAttentionItem(session, nil, nil, &run); item != nil {
		t.Fatalf("older task created attention: %#v", item)
	}
}
