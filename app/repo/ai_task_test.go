package repo

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDeleteTaskRemovesTaskRecordsAndResetsSession(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ai-task.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&model.AIDevSession{}, &model.AITask{}, &model.AIMessage{}, &model.AIInstruction{},
		&model.AIApproval{}, &model.AIExecutionRun{}, &model.AIPreview{}, &model.AITimelineEvent{},
	); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	session := &model.AIDevSession{UserID: 1, Title: "session", WorkDir: "/tmp", CurrentStage: "failed"}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, Title: "task", WorkDir: "/tmp", Status: "failed"}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	session.LastTaskID = task.ID
	if err := database.Save(session).Error; err != nil {
		t.Fatal(err)
	}
	instruction := &model.AIInstruction{SessionID: session.ID, UserID: 1, TaskID: task.ID, Content: "run", Status: "failed"}
	if err := database.Create(instruction).Error; err != nil {
		t.Fatal(err)
	}
	records := []any{
		&model.AIMessage{SessionID: session.ID, TaskID: task.ID, Role: "agent", Content: "done"},
		&model.AIApproval{SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID, RequestUserID: 1, Title: "approve", Content: "content"},
		&model.AIExecutionRun{SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID, ExecutorID: "codex", Prompt: "run", Status: "failed", StartedAt: time.Now()},
		&model.AIPreview{SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID, Title: "preview", URL: "https://example.com"},
		&model.AITimelineEvent{SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID, EventType: "result", Title: "done"},
	}
	for _, record := range records {
		if err := database.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := (&aiTaskRepo{}).DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	var taskCount int64
	if err := database.Model(&model.AITask{}).Where("id = ?", task.ID).Count(&taskCount).Error; err != nil || taskCount != 0 {
		t.Fatalf("remaining task count = %d, err = %v", taskCount, err)
	}
	for _, target := range []any{
		&model.AIMessage{}, &model.AIInstruction{}, &model.AIApproval{},
		&model.AIExecutionRun{}, &model.AIPreview{}, &model.AITimelineEvent{},
	} {
		var count int64
		if err := database.Model(target).Where("task_id = ?", task.ID).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("remaining %T count = %d, err = %v", target, count, err)
		}
	}
	if err := database.First(session, session.ID).Error; err != nil || session.LastTaskID != 0 || session.CurrentStage != "idle" {
		t.Fatalf("session was not reset: %#v, err = %v", session, err)
	}
}
