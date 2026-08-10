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

func TestLoadCodeTokenUsageAggregatesSessionProjectAndDay(t *testing.T) {
	oldDB := global.DB
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "tokens.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = database
	t.Cleanup(func() { global.DB = oldDB })
	if err := database.AutoMigrate(&model.AIProject{}, &model.AIDevSession{}, &model.AIExecutionRun{}); err != nil {
		t.Fatal(err)
	}
	first := model.AIDevSession{UserID: 1, ProjectID: 9, Title: "one", WorkDir: t.TempDir()}
	second := model.AIDevSession{UserID: 1, ProjectID: 9, Title: "two", WorkDir: t.TempDir()}
	other := model.AIDevSession{UserID: 1, ProjectID: 10, Title: "other", WorkDir: t.TempDir()}
	if err := database.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	runs := []model.AIExecutionRun{
		{SessionID: first.ID, ExecutorID: "codex", Prompt: "a", Status: "completed", StartedAt: now, CreatedAt: now, InputTokens: 10, OutputTokens: 5, CachedInputTokens: 3, TotalTokens: 15},
		{SessionID: second.ID, ExecutorID: "claude", Prompt: "b", Status: "completed", StartedAt: now, CreatedAt: now, InputTokens: 20, OutputTokens: 7, ReasoningTokens: 2, TotalTokens: 27},
		{SessionID: other.ID, ExecutorID: "codex", Prompt: "c", Status: "completed", StartedAt: now, CreatedAt: now, TotalTokens: 99},
	}
	if err := database.Create(&runs).Error; err != nil {
		t.Fatal(err)
	}
	usage, err := loadCodeTokenUsage(&first)
	if err != nil {
		t.Fatal(err)
	}
	if usage.Session.TotalTokens != 15 || usage.Session.Runs != 1 || usage.Project.TotalTokens != 42 || usage.Project.Runs != 2 {
		t.Fatalf("unexpected token usage: %#v", usage)
	}
	if len(usage.Daily) != 1 || usage.Daily[0].TotalTokens != 42 || usage.Daily[0].CachedInputTokens != 3 {
		t.Fatalf("unexpected daily usage: %#v", usage.Daily)
	}
}

func TestSumCodeTokenUsageBackfillsLegacyRawOutput(t *testing.T) {
	oldDB := global.DB
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "tokens-backfill.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = database
	t.Cleanup(func() { global.DB = oldDB })
	if err := database.AutoMigrate(&model.AIExecutionRun{}); err != nil {
		t.Fatal(err)
	}
	raw := `{"type":"turn.completed","usage":{"input_tokens":120,"cached_input_tokens":80,"output_tokens":30,"reasoning_output_tokens":10}}`
	run := model.AIExecutionRun{ExecutorID: "codex", Prompt: "legacy", RawOutput: raw, Status: "completed", StartedAt: time.Now()}
	if err := database.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	usage, err := sumCodeTokenUsage(database.Model(&model.AIExecutionRun{}))
	if err != nil {
		t.Fatal(err)
	}
	if usage.InputTokens != 120 || usage.OutputTokens != 30 || usage.CachedInputTokens != 80 || usage.ReasoningTokens != 10 || usage.TotalTokens != 150 {
		t.Fatalf("unexpected backfilled usage: %#v", usage)
	}
	if err := database.First(&run, run.ID).Error; err != nil || run.TotalTokens != 150 {
		t.Fatalf("legacy usage was not persisted: %#v, %v", run, err)
	}
}
