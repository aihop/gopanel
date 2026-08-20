package api

import (
	"context"
	"testing"
	"time"
)

func TestCodeSessionRunnerWaitsForConversationHandover(t *testing.T) {
	runner := &codeSessionRunner{interactive: make(map[uint]chan struct{})}
	runner.setInteractive(21, true)
	done := make(chan struct{})
	go func() {
		runner.waitForConversation(21)
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("queued conversation continued while CLI owned the session")
	case <-time.After(25 * time.Millisecond):
	}
	runner.setInteractive(21, false)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("queued conversation did not resume after handover")
	}
}

func TestHandoverCodeSessionToConversationCancelsOnlyInteractive(t *testing.T) {
	previousCoordinator := codeExecutions
	previousRunner := backgroundCodeRunner
	codeExecutions = newCodeExecutionCoordinator(2, 2)
	backgroundCodeRunner = &codeSessionRunner{interactive: make(map[uint]chan struct{})}
	t.Cleanup(func() {
		codeExecutions = previousCoordinator
		backgroundCodeRunner = previousRunner
	})
	backgroundCodeRunner.setInteractive(21, true)
	interactive, err := codeExecutions.acquireOwned(
		context.Background(), 21, []string{"/workspace/interactive"}, codeExecutionInteractive, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	interactive.SetCancel(func() { interactive.Release() })
	quality, err := codeExecutions.acquireOwned(
		context.Background(), 21, []string{"/workspace/quality"}, codeExecutionQuality, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer quality.Release()
	if err := handoverCodeSessionToConversation(context.Background(), 21); err != nil {
		t.Fatal(err)
	}
	if codeExecutions.hasSessionKind(21, codeExecutionInteractive) {
		t.Fatal("interactive lease survived conversation handover")
	}
	if !codeExecutions.hasSessionKind(21, codeExecutionQuality) {
		t.Fatal("quality lease was cancelled by conversation handover")
	}
}
