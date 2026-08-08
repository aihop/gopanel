//go:build darwin

package helper

import (
	"context"
	"errors"
	"testing"
)

func TestRestartLoadedLaunchdServiceFallsBackToTerminatingManagedProcess(t *testing.T) {
	originalTerminate := terminateLaunchdProcess
	t.Cleanup(func() { terminateLaunchdProcess = originalTerminate })
	terminatedPID := 0
	terminateLaunchdProcess = func(pid int) error {
		terminatedPID = pid
		return nil
	}

	err := restartLoadedLaunchdService(context.Background(), "missing.test.service", 4321)
	if err != nil {
		t.Fatal(err)
	}
	if terminatedPID != 4321 {
		t.Fatalf("unexpected terminated pid: %d", terminatedPID)
	}
}

func TestRestartLoadedLaunchdServiceRequiresManagedPIDForFallback(t *testing.T) {
	originalTerminate := terminateLaunchdProcess
	t.Cleanup(func() { terminateLaunchdProcess = originalTerminate })
	terminateLaunchdProcess = func(int) error {
		return errors.New("must not terminate without a managed pid")
	}

	if err := restartLoadedLaunchdService(context.Background(), "missing.test.service", 0); err == nil {
		t.Fatal("missing managed pid should preserve the kickstart failure")
	}
}
