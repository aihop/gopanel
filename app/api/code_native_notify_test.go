package api

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNativeCodeNotifyTrackerSkipsBaselineAndDuplicates(t *testing.T) {
	tracker := &nativeCodeNotifyTracker{}
	states := []string{"completed", "completed", "responding", "needsInput", "needsInput", "responding", "completed"}
	want := []string{"", "", "", service.CodeNotifyApproval, "", "", service.CodeNotifyCompleted}
	for index, state := range states {
		got := tracker.observe(&codexRuntimeState{ResponseState: state})
		if got != want[index] {
			t.Fatalf("state %q: got %q, want %q", state, got, want[index])
		}
	}
}

func TestNativeCodeNotifyTrackerRequiresActiveTurn(t *testing.T) {
	tracker := &nativeCodeNotifyTracker{}
	for _, state := range []string{"idle", "completed", "failed", "needsInput"} {
		if got := tracker.observe(&codexRuntimeState{ResponseState: state}); got != "" {
			t.Fatalf("inactive state %q unexpectedly notified %q", state, got)
		}
	}
	if got := tracker.observe(&codexRuntimeState{ResponseState: "responding"}); got != "" {
		t.Fatalf("responding unexpectedly notified %q", got)
	}
	if got := tracker.observe(&codexRuntimeState{ResponseState: "failed"}); got != service.CodeNotifyFailed {
		t.Fatalf("failed notification = %q", got)
	}
}

func TestNativeCodeTaskState(t *testing.T) {
	tests := map[string][2]string{
		"idle":       {"", ""},
		"responding": {"running", "executing"},
		"needsInput": {"pending_approval", "awaiting_approval"},
		"completed":  {"completed", "completed"},
		"failed":     {"failed", "failed"},
	}
	for responseState, expected := range tests {
		taskStatus, sessionStage := nativeCodeTaskState(&codexRuntimeState{ResponseState: responseState})
		if taskStatus != expected[0] || sessionStage != expected[1] {
			t.Fatalf("runtime state %q = (%q, %q), want (%q, %q)", responseState, taskStatus, sessionStage, expected[0], expected[1])
		}
	}
}

func TestSyncNativeCodeTaskStatusReconcilesSession(t *testing.T) {
	tests := []struct {
		responseState string
		taskStatus    string
		sessionStage  string
	}{
		{responseState: "responding", taskStatus: "running", sessionStage: "executing"},
		{responseState: "needsInput", taskStatus: "pending_approval", sessionStage: "awaiting_approval"},
		{responseState: "completed", taskStatus: "completed", sessionStage: "completed"},
		{responseState: "failed", taskStatus: "failed", sessionStage: "failed"},
	}
	for _, test := range tests {
		t.Run(test.responseState, func(t *testing.T) {
			database, session, task := setupNativeCodeTaskStateTest(t, "active", "interactive", "active")
			state := &codexRuntimeState{ResponseState: test.responseState, UpdatedAt: task.CreatedAt.Add(time.Second)}
			if !syncNativeCodeTaskStatus(session, state) {
				t.Fatal("native runtime state was not synchronized")
			}
			if err := database.First(task, task.ID).Error; err != nil || task.Status != test.taskStatus {
				t.Fatalf("task status = %q, err = %v", task.Status, err)
			}
			if err := database.First(session, session.ID).Error; err != nil || session.CurrentStage != test.sessionStage {
				t.Fatalf("session stage = %q, err = %v", session.CurrentStage, err)
			}
		})
	}
}

func TestSyncNativeCodeTaskStatusDoesNotReopenDeliveringSession(t *testing.T) {
	database, session, task := setupNativeCodeTaskStateTest(t, codeSessionStatusDelivering, "delivery_queued", codeSessionStatusDelivering)
	state := &codexRuntimeState{ResponseState: "completed", UpdatedAt: task.CreatedAt.Add(time.Second)}
	if syncNativeCodeTaskStatus(session, state) {
		t.Fatal("delivering session accepted native runtime state")
	}
	if err := database.First(task, task.ID).Error; err != nil || task.Status != codeSessionStatusDelivering {
		t.Fatalf("task changed: status=%q, err=%v", task.Status, err)
	}
	if err := database.First(session, session.ID).Error; err != nil || session.Status != codeSessionStatusDelivering || session.CurrentStage != "delivery_queued" {
		t.Fatalf("session changed: status=%q stage=%q err=%v", session.Status, session.CurrentStage, err)
	}
}

func TestSyncNativeCodeTaskStatusRejectsStaleRuntime(t *testing.T) {
	database, session, task := setupNativeCodeTaskStateTest(t, "active", "interactive", "active")
	state := &codexRuntimeState{ResponseState: "completed", UpdatedAt: task.CreatedAt.Add(-2 * time.Second)}
	if syncNativeCodeTaskStatus(session, state) {
		t.Fatal("stale native runtime state was synchronized")
	}
	if err := database.First(task, task.ID).Error; err != nil || task.Status != "active" {
		t.Fatalf("task changed: status=%q, err=%v", task.Status, err)
	}
	if err := database.First(session, session.ID).Error; err != nil || session.CurrentStage != "interactive" {
		t.Fatalf("session changed: stage=%q, err=%v", session.CurrentStage, err)
	}
}

func setupNativeCodeTaskStateTest(
	t *testing.T,
	sessionStatus string,
	sessionStage string,
	taskStatus string,
) (*gorm.DB, *model.AIDevSession, *model.AITask) {
	t.Helper()
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "native-task-state.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIDevSession{}, &model.AITask{}, &model.AIInstruction{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	session := &model.AIDevSession{
		UserID: 1, Title: "session", WorkDir: "/tmp", Status: sessionStatus, CurrentStage: sessionStage,
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{
		UserID: 1, SessionID: session.ID, Title: "task", WorkDir: "/tmp", Status: taskStatus,
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	session.LastTaskID = task.ID
	if err := database.Save(session).Error; err != nil {
		t.Fatal(err)
	}
	return database, session, task
}

func TestNativeCodeRuntimeFreshnessUsesLatestInstruction(t *testing.T) {
	taskCreatedAt := time.Date(2026, 7, 30, 1, 0, 0, 0, time.UTC)
	instructionAt := taskCreatedAt.Add(time.Minute)
	session := &model.AIDevSession{LastInstructionAt: &instructionAt}
	task := &model.AITask{CreatedAt: taskCreatedAt}
	if nativeCodeRuntimeIsFresh(session, task, &codexRuntimeState{UpdatedAt: taskCreatedAt.Add(30 * time.Second)}) {
		t.Fatal("runtime state from the previous turn should be stale")
	}
	if !nativeCodeRuntimeIsFresh(session, task, &codexRuntimeState{UpdatedAt: instructionAt}) {
		t.Fatal("runtime state from the latest instruction should be accepted")
	}
}
