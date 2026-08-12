package api

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestCodeSessionWorkspaceMutationErrorUsesSessionMessage(t *testing.T) {
	if err := codeSessionWorkspaceMutationError(errCodeExecutionBusy); !errors.Is(err, errCodeSessionWorkspaceBusy) {
		t.Fatalf("workspace mutation busy error = %v", err)
	}
	original := errors.New("other error")
	if err := codeSessionWorkspaceMutationError(original); !errors.Is(err, original) {
		t.Fatalf("unrelated error changed: %v", err)
	}
}

func TestCodeSessionGitMutationAllowsInteractiveTerminal(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{ID: 921, UserID: 1, Status: codeSessionStatusActive, WorkDir: t.TempDir()}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	previousCoordinator := codeExecutions
	codeExecutions = newCodeExecutionCoordinator(2, 2)
	t.Cleanup(func() { codeExecutions = previousCoordinator })
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionInteractive, false)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	var cancelled atomic.Bool
	lease.SetCancel(func() { cancelled.Store(true) })
	called := false
	if err := runCodeSessionGitMutation(session, func(*model.AIDevSession) error {
		called = true
		return nil
	}); err != nil || !called {
		t.Fatalf("interactive terminal blocked Git mutation: called=%v err=%v", called, err)
	}
	if cancelled.Load() {
		t.Fatal("Git mutation cancelled the interactive terminal")
	}
}

func TestCodeSessionGitMutationRejectsActiveInstruction(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{ID: 922, UserID: 1, Status: codeSessionStatusActive, WorkDir: t.TempDir()}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	instruction := &model.AIInstruction{SessionID: session.ID, UserID: session.UserID, Status: "running", Content: "test"}
	if err := database.Create(instruction).Error; err != nil {
		t.Fatal(err)
	}
	called := false
	err := runCodeSessionGitMutation(session, func(*model.AIDevSession) error {
		called = true
		return nil
	})
	if err == nil || called {
		t.Fatalf("active instruction did not block Git mutation: called=%v err=%v", called, err)
	}
}
