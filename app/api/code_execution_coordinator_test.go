package api

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestCodeExecutionCoordinatorSerializesSharedWorkspace(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(2)
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

func TestCodeExecutionCoordinatorPreemptsInteractiveLease(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(1)
	interactive, err := coordinator.acquire(context.Background(), []string{"/workspace/shared"}, codeExecutionInteractive, true, false)
	if err != nil {
		t.Fatal(err)
	}
	var cancelled atomic.Bool
	interactive.SetCancel(func() {
		cancelled.Store(true)
		interactive.Release()
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	instruction, err := coordinator.acquire(ctx, []string{"/workspace/shared"}, codeExecutionInstruction, true, true)
	if err != nil {
		t.Fatal(err)
	}
	defer instruction.Release()
	if !cancelled.Load() {
		t.Fatal("interactive execution was not preempted")
	}
}

func TestCodeExecutionCoordinatorEnforcesCapacity(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(1)
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
	coordinator := newCodeExecutionCoordinator(1)
	lease, err := coordinator.acquire(context.Background(), []string{"/workspace/shared"}, codeExecutionInstruction, true, false)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	cancelContext, cancelWait := context.WithTimeout(context.Background(), time.Second)
	defer cancelWait()
	waitDone := make(chan bool, 1)
	go func() {
		waitDone <- coordinator.cancelAndWait(cancelContext, []string{"/workspace/shared"})
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
