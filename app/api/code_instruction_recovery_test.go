package api

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRecoverInterruptedCodeInstructionsMarksRunningStateFailed(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "recovery.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&model.AIDevSession{}, &model.AITask{}, &model.AIInstruction{},
		&model.AIExecutionRun{}, &model.AITimelineEvent{},
	); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	session := &model.AIDevSession{UserID: 1, Title: "session", WorkDir: "/tmp", Status: "active", CurrentStage: "executing"}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, Title: "task", WorkDir: "/tmp", Status: "running"}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	instruction := &model.AIInstruction{SessionID: session.ID, UserID: 1, TaskID: task.ID, Content: "run", Status: "running"}
	if err := database.Create(instruction).Error; err != nil {
		t.Fatal(err)
	}
	run := &model.AIExecutionRun{SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID, ExecutorID: "codex", Prompt: "run", Status: "running", StartedAt: time.Now()}
	if err := database.Create(run).Error; err != nil {
		t.Fatal(err)
	}

	if err := recoverInterruptedCodeInstructions(); err != nil {
		t.Fatal(err)
	}
	if err := database.First(instruction, instruction.ID).Error; err != nil || instruction.Status != "failed" {
		t.Fatalf("instruction status = %q, err = %v", instruction.Status, err)
	}
	if err := database.First(task, task.ID).Error; err != nil || task.Status != "failed" {
		t.Fatalf("task status = %q, err = %v", task.Status, err)
	}
	if err := database.First(session, session.ID).Error; err != nil || session.CurrentStage != "failed" {
		t.Fatalf("session stage = %q, err = %v", session.CurrentStage, err)
	}
	if err := database.First(run, run.ID).Error; err != nil {
		t.Fatal(err)
	}
	if run.Status != "failed" || run.ExitCode != -1 || run.CompletedAt == nil || !strings.Contains(run.ErrorMessage, "服务重启") {
		t.Fatalf("unexpected recovered run: %#v", run)
	}
	var event model.AITimelineEvent
	if err := database.Where("instruction_id = ? AND event_type = ?", instruction.ID, "execution_interrupted").First(&event).Error; err != nil {
		t.Fatal(err)
	}
}
