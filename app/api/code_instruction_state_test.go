package api

import (
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestReconcileCodeTaskStateKeepsActiveInstructionVisible(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "state.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIDevSession{}, &model.AITask{}, &model.AIInstruction{}); err != nil {
		t.Fatal(err)
	}
	session := &model.AIDevSession{UserID: 1, Title: "session", WorkDir: "/tmp", Status: "active", CurrentStage: "executing"}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, Title: "task", WorkDir: "/tmp", Status: "running"}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	instructions := []*model.AIInstruction{
		{SessionID: session.ID, UserID: 1, TaskID: task.ID, Content: "done", Status: "completed"},
		{SessionID: session.ID, UserID: 1, TaskID: task.ID, Content: "approval", Status: "pending_approval"},
		{SessionID: session.ID, UserID: 1, TaskID: task.ID, Content: "queued", Status: "queued"},
	}
	if err := database.Create(&instructions).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Transaction(func(tx *gorm.DB) error {
		return reconcileCodeTaskState(tx, session, task, "completed", "completed")
	}); err != nil {
		t.Fatal(err)
	}
	if task.Status != "pending_approval" || session.CurrentStage != "awaiting_approval" {
		t.Fatalf("unexpected aggregate: task=%q session=%q", task.Status, session.CurrentStage)
	}
	instructions[1].Status = "rejected"
	if err := database.Save(instructions[1]).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Transaction(func(tx *gorm.DB) error {
		return reconcileCodeTaskState(tx, session, task, "cancelled", "approval_rejected")
	}); err != nil {
		t.Fatal(err)
	}
	if task.Status != "queued" || session.CurrentStage != "instruction_queued" {
		t.Fatalf("queued work was hidden: task=%q session=%q", task.Status, session.CurrentStage)
	}
}

func TestCancelQueuedCodeInstructionsReconcilesTaskState(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cancel-state.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIDevSession{}, &model.AITask{}, &model.AIInstruction{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	session := &model.AIDevSession{UserID: 1, Title: "session", WorkDir: "/tmp", Status: "active", CurrentStage: "instruction_queued"}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, Title: "task", WorkDir: "/tmp", Status: "queued"}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	session.LastTaskID = task.ID
	if err := database.Save(session).Error; err != nil {
		t.Fatal(err)
	}
	instruction := &model.AIInstruction{SessionID: session.ID, UserID: 1, TaskID: task.ID, Content: "queued", Status: "queued"}
	if err := database.Create(instruction).Error; err != nil {
		t.Fatal(err)
	}
	cancelled, err := cancelQueuedCodeInstructions(session)
	if err != nil || cancelled != 1 {
		t.Fatalf("cancelled = %d, err = %v", cancelled, err)
	}
	if taskErr := database.First(task, task.ID).Error; taskErr != nil || task.Status != "cancelled" {
		t.Fatalf("task status = %q, err = %v", task.Status, taskErr)
	}
	if sessionErr := database.First(session, session.ID).Error; sessionErr != nil || session.CurrentStage != "cancelled" {
		t.Fatalf("session stage = %q, err = %v", session.CurrentStage, sessionErr)
	}
}
