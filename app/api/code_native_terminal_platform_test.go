package api

import (
	"os/exec"
	"testing"
)

func TestNativeTerminalPlatformSupportsCurrentHost(t *testing.T) {
	if !nativeTerminalPlatformSupported() {
		t.Fatal("current platform should support native terminals")
	}
}

func TestNativeTerminalRejectsMissingCommand(t *testing.T) {
	if _, err := startNativeTerminal(&exec.Cmd{}, 80, 24); err == nil {
		t.Fatal("empty terminal command should fail")
	}
}
