package api

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestLoadCodeProjectOverviewAggregatesProjectState(t *testing.T) {
	oldDB := global.DB
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "overview.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = database
	t.Cleanup(func() { global.DB = oldDB })
	if err := database.AutoMigrate(&model.AIGroup{}, &model.AIDevSession{}, &model.AITask{}, &model.AIExecutionRun{}); err != nil {
		t.Fatal(err)
	}
	project := model.AIGroup{Name: "project", CreatorID: 1, WorkDir: t.TempDir(), MonthlyTokenBudget: 1000}
	if err := database.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	session := model.AIDevSession{UserID: 1, ProjectID: project.ID, Title: "session", WorkDir: project.WorkDir, CurrentStage: "executing"}
	if err := database.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	task := model.AITask{UserID: 1, SessionID: session.ID, ProjectID: project.ID, Title: "active task", WorkDir: project.WorkDir, Status: "running"}
	if err := database.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	run := model.AIExecutionRun{SessionID: session.ID, TaskID: task.ID, ExecutorID: "codex", Model: "gpt-5", Prompt: "work", Status: "completed", StartedAt: now, CreatedAt: now, TotalTokens: 240}
	if err := database.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	otherSession := model.AIDevSession{UserID: 2, ProjectID: project.ID, Title: "other session", WorkDir: project.WorkDir}
	if err := database.Create(&otherSession).Error; err != nil {
		t.Fatal(err)
	}
	otherRun := model.AIExecutionRun{SessionID: otherSession.ID, ExecutorID: "claude", Model: "private-model", Prompt: "other", Status: "completed", StartedAt: now.Add(time.Minute), CreatedAt: now.Add(time.Minute), TotalTokens: 400}
	if err := database.Create(&otherRun).Error; err != nil {
		t.Fatal(err)
	}
	overview, err := loadCodeProjectOverview(&project, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if overview.TaskCount != 1 || overview.ExecutionSummary.Status != "running" || overview.ExecutionSummary.CurrentTaskID != task.ID {
		t.Fatalf("unexpected project summary: %#v", overview)
	}
	if overview.TokenUsage.TotalTokens != 240 || overview.Budget.UsedTokens != 640 || overview.LatestRun == nil || overview.LatestRun.Model != "gpt-5" {
		t.Fatalf("unexpected project usage: %#v", overview)
	}
}
