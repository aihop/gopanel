package api

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func TestCodeExecutionCoordinatorSerializesSharedWorkspace(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(2, 2)
	first, err := coordinator.acquire(context.Background(), []string{"/workspace/shared"}, codeExecutionInstruction, true, false)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	if _, err := coordinator.acquire(context.Background(), []string{"/workspace/shared"}, codeExecutionQuality, false, false); !errors.Is(err, errCodeExecutionBusy) {
		t.Fatalf("shared workspace error = %v", err)
	}
	second, err := coordinator.acquire(context.Background(), []string{"/workspace/isolated"}, codeExecutionInstruction, true, false)
	if err != nil {
		t.Fatalf("isolated workspace should run concurrently: %v", err)
	}
	second.Release()
}

func TestCodeExecutionCoordinatorWaitsWithoutPreemptingInteractiveLease(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(1, 1)
	interactive, err := coordinator.acquire(context.Background(), []string{"/workspace/shared"}, codeExecutionInteractive, true, false)
	if err != nil {
		t.Fatal(err)
	}
	var cancelled atomic.Bool
	interactive.SetCancel(func() {
		cancelled.Store(true)
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		instruction, acquireErr := coordinator.acquire(ctx, []string{"/workspace/shared"}, codeExecutionInstruction, true, true)
		if instruction != nil {
			instruction.Release()
		}
		result <- acquireErr
	}()
	select {
	case err := <-result:
		t.Fatalf("instruction did not wait for interactive lease: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if cancelled.Load() {
		t.Fatal("interactive execution was preempted")
	}
	interactive.Release()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestCodeExecutionCoordinatorDeliveryDoesNotWaitForInteractiveWorktree(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(2, 2)
	session := &model.AIDevSession{ID: 31, WorkDir: "/workspace/session", SourceWorkDir: "/workspace/source"}
	interactive, err := coordinator.acquireSession(context.Background(), session, codeExecutionInteractive, false)
	if err != nil {
		t.Fatal(err)
	}
	defer interactive.Release()
	var cancelled atomic.Bool
	interactive.SetCancel(func() { cancelled.Store(true) })
	delivery, err := coordinator.acquireSession(context.Background(), session, codeExecutionDelivery, false)
	if err != nil {
		t.Fatalf("delivery should lock only the source repository: %v", err)
	}
	delivery.Release()
	if cancelled.Load() {
		t.Fatal("delivery cancelled the interactive terminal")
	}
}

func TestCodeExecutionCoordinatorNewSessionDoesNotInterruptSharedWorkspace(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(2, 2)
	firstSession := &model.AIDevSession{ID: 21, WorkDir: "/workspace/shared"}
	first, err := coordinator.acquireSession(context.Background(), firstSession, codeExecutionInteractive, false)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	var interrupted atomic.Bool
	first.SetCancel(func() { interrupted.Store(true) })

	secondSession := &model.AIDevSession{ID: 22, WorkDir: "/workspace/shared"}
	if _, err := coordinator.acquireSession(context.Background(), secondSession, codeExecutionInteractive, false); !errors.Is(err, errCodeExecutionBusy) {
		t.Fatalf("new session error = %v", err)
	}
	if interrupted.Load() {
		t.Fatal("existing session was interrupted by a new session")
	}
}

func TestCodeExecutionCoordinatorEnforcesCapacity(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(1, 1)
	first, err := coordinator.acquire(context.Background(), []string{"/workspace/one"}, codeExecutionInstruction, true, false)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	if _, err := coordinator.acquire(context.Background(), []string{"/workspace/two"}, codeExecutionQuality, false, false); !errors.Is(err, errCodeExecutionCapacity) {
		t.Fatalf("capacity error = %v", err)
	}
}

func TestCodeExecutionCoordinatorDeliversCancellationSetBeforeHandler(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(1, 1)
	lease, err := coordinator.acquireSession(context.Background(), &model.AIDevSession{ID: 7, WorkDir: "/workspace/shared"}, codeExecutionInstruction, true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	cancelContext, cancelWait := context.WithTimeout(context.Background(), time.Second)
	defer cancelWait()
	waitDone := make(chan bool, 1)
	go func() {
		waitDone <- coordinator.cancelSessionAndWait(cancelContext, 7)
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lease.SetCancel(cancel)
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("late cancellation handler was not called")
	}
	lease.Release()
	if cancelled := <-waitDone; !cancelled {
		t.Fatal("active lease was not reported as cancelled")
	}
}

func TestCodeExecutionCoordinatorCancelsOnlyTargetSession(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(2, 2)
	first, err := coordinator.acquireSession(context.Background(), &model.AIDevSession{ID: 11, WorkDir: "/workspace/one"}, codeExecutionInstruction, true)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	second, err := coordinator.acquireSession(context.Background(), &model.AIDevSession{ID: 12, WorkDir: "/workspace/two"}, codeExecutionInstruction, true)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Release()
	var firstCancelled atomic.Bool
	first.SetCancel(func() {
		firstCancelled.Store(true)
		first.Release()
	})
	var secondCancelled atomic.Bool
	second.SetCancel(func() {
		secondCancelled.Store(true)
		second.Release()
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !coordinator.cancelSessionAndWait(ctx, 12) {
		t.Fatal("target session lease was not cancelled")
	}
	if firstCancelled.Load() {
		t.Fatal("another session lease was cancelled")
	}
	if !secondCancelled.Load() {
		t.Fatal("target session cancel handler was not called")
	}
}

func TestCodeExecutionCoordinatorCancelsOnlyRequestedSessionKind(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(2, 2)
	interactive, err := coordinator.acquireOwned(context.Background(), 21, []string{"/workspace/interactive"}, codeExecutionInteractive, false)
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := coordinator.acquireOwned(context.Background(), 21, []string{"/workspace/instruction"}, codeExecutionInstruction, false)
	if err != nil {
		t.Fatal(err)
	}
	interactive.SetCancel(func() { interactive.Release() })
	var instructionCancelled atomic.Bool
	instruction.SetCancel(func() {
		instructionCancelled.Store(true)
		instruction.Release()
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !coordinator.cancelSessionKindAndWait(ctx, 21, codeExecutionInteractive) {
		t.Fatal("interactive lease was not cancelled")
	}
	if instructionCancelled.Load() {
		t.Fatal("instruction lease was cancelled with interactive terminal")
	}
	instruction.Release()
}
