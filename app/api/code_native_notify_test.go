package api

import (
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
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

func TestNativeCodeTaskStatus(t *testing.T) {
	tests := map[string]string{
		"idle":       "",
		"responding": "running",
		"needsInput": "pending_approval",
		"completed":  "completed",
		"failed":     "failed",
	}
	for responseState, expected := range tests {
		actual := nativeCodeTaskStatus(&codexRuntimeState{ResponseState: responseState})
		if actual != expected {
			t.Fatalf("runtime state %q = task status %q, want %q", responseState, actual, expected)
		}
	}
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
