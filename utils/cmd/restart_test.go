package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/aihop/gopanel/utils/gpc"
)

func TestPrepareGoPanelRestartUsesGPCForManagedService(t *testing.T) {
	restorePanelRestartDependencies(t)
	panelRestartOS = "linux"
	operations := make([]string, 0, 2)
	panelRestartGPCDo = func(_ context.Context, _ string, params map[string]interface{}) (*gpc.Response, error) {
		operation, _ := params["op"].(string)
		operations = append(operations, operation)
		return &gpc.Response{OK: true, Output: "active"}, nil
	}
	panelRestartServiceExists = func(string) (bool, error) {
		t.Fatal("service discovery should not run after a successful gpc preflight")
		return false, nil
	}

	restart, err := prepareGoPanelRestart()
	if err != nil {
		t.Fatal(err)
	}
	if err := restart(); err != nil {
		t.Fatal(err)
	}
	if len(operations) != 2 || operations[0] != "status" || operations[1] != "restart" {
		t.Fatalf("unexpected gpc operations: %#v", operations)
	}
}

func TestPrepareGoPanelRestartRejectsManagedServiceWithoutGPC(t *testing.T) {
	restorePanelRestartDependencies(t)
	panelRestartOS = "linux"
	panelRestartGPCDo = func(context.Context, string, map[string]interface{}) (*gpc.Response, error) {
		return &gpc.Response{}, errors.New("gpc unavailable")
	}
	panelRestartServiceExists = func(string) (bool, error) { return true, nil }
	standaloneCalled := false
	panelRestartStandalone = func() error {
		standaloneCalled = true
		return nil
	}

	if _, err := prepareGoPanelRestart(); err == nil {
		t.Fatal("managed service should require gpc")
	}
	if standaloneCalled {
		t.Fatal("managed service must not start a second standalone process")
	}
}

func TestPrepareGoPanelRestartRejectsManagedServiceAfterEmptyGPCResponse(t *testing.T) {
	restorePanelRestartDependencies(t)
	panelRestartOS = "linux"
	panelRestartGPCDo = func(context.Context, string, map[string]interface{}) (*gpc.Response, error) {
		return nil, nil
	}
	panelRestartServiceExists = func(string) (bool, error) { return true, nil }

	if _, err := prepareGoPanelRestart(); err == nil {
		t.Fatal("managed service should reject an empty gpc response")
	}
}

func TestPrepareGoPanelRestartAllowsStandaloneDevelopment(t *testing.T) {
	restorePanelRestartDependencies(t)
	panelRestartOS = "darwin"
	panelRestartGPCDo = func(context.Context, string, map[string]interface{}) (*gpc.Response, error) {
		return &gpc.Response{}, errors.New("gpc unavailable")
	}
	panelRestartServiceExists = func(string) (bool, error) { return false, nil }
	standaloneCalled := false
	panelRestartStandalone = func() error {
		standaloneCalled = true
		return nil
	}

	restart, err := prepareGoPanelRestart()
	if err != nil {
		t.Fatal(err)
	}
	if err := restart(); err != nil {
		t.Fatal(err)
	}
	if !standaloneCalled {
		t.Fatal("standalone development restart was not selected")
	}
}

func restorePanelRestartDependencies(t *testing.T) {
	t.Helper()
	originalOS := panelRestartOS
	originalGPCDo := panelRestartGPCDo
	originalServiceExists := panelRestartServiceExists
	originalStandalone := panelRestartStandalone
	t.Cleanup(func() {
		panelRestartOS = originalOS
		panelRestartGPCDo = originalGPCDo
		panelRestartServiceExists = originalServiceExists
		panelRestartStandalone = originalStandalone
	})
}
