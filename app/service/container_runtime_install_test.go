package service

import (
	"errors"
	"testing"
)

func TestRuntimeInstallFailureRequiresGpcUpdateForUnknownAction(t *testing.T) {
	message, needsAction, output := runtimeInstallFailure(errors.New("unknown action"), " helper output \n")
	if needsAction != "updateGpc" {
		t.Fatalf("unexpected needs action: %q", needsAction)
	}
	if message == "unknown action" {
		t.Fatal("unknown action should be replaced with an actionable message")
	}
	if output != "helper output" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRuntimeInstallFailurePreservesOtherErrors(t *testing.T) {
	message, needsAction, output := runtimeInstallFailure(errors.New("package manager failed"), "")
	if message != "package manager failed" || needsAction != "" || output != "" {
		t.Fatalf("unexpected failure state: message=%q action=%q output=%q", message, needsAction, output)
	}
}
