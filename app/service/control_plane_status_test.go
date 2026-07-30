package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyControlPlaneState(t *testing.T) {
	tempDir := t.TempDir()
	existingSocket := filepath.Join(tempDir, "existing.sock")
	if err := os.WriteFile(existingSocket, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		installed bool
		path      string
		err       error
		want      string
	}{
		{name: "missing binary", path: existingSocket, err: errors.New("offline"), want: ControlPlaneMissing},
		{name: "healthy", installed: true, path: existingSocket, want: ControlPlaneHealthy},
		{name: "socket missing", installed: true, path: filepath.Join(tempDir, "missing.sock"), err: os.ErrNotExist, want: ControlPlaneSocketMissing},
		{name: "permission denied", installed: true, path: existingSocket, err: os.ErrPermission, want: ControlPlanePermissionDenied},
		{name: "service stopped", installed: true, path: existingSocket, err: errors.New("connection refused"), want: ControlPlaneServiceStopped},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyControlPlaneState(test.installed, test.path, test.err); got != test.want {
				t.Fatalf("classifyControlPlaneState() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGpcRecoveryCommands(t *testing.T) {
	darwin := gpcRecoveryCommands("darwin", ControlPlanePermissionDenied, true)
	if len(darwin) != 1 || !strings.Contains(darwin[0], "UserName string root") || !strings.Contains(darwin[0], "launchctl bootstrap") {
		t.Fatalf("unexpected macOS recovery command: %#v", darwin)
	}
	linux := gpcRecoveryCommands("linux", ControlPlaneServiceStopped, false)
	if len(linux) != 1 || !strings.Contains(linux[0], "systemctl enable --now gpc.service") {
		t.Fatalf("unexpected Linux recovery command: %#v", linux)
	}
	missing := gpcRecoveryCommands("darwin", ControlPlaneMissing, false)
	if len(missing) != 1 || !strings.Contains(missing[0], "https://gopanel.run") {
		t.Fatalf("unexpected install command: %#v", missing)
	}
}
