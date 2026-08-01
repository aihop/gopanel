package api

import (
	"errors"
	"testing"
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
