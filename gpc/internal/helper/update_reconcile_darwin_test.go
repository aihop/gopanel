//go:build darwin

package helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPanelUpdateReconcileMarkerTracksRecentUpdate(t *testing.T) {
	baseDir := t.TempDir()
	server := NewServer(Config{BaseDir: baseDir})
	if err := os.WriteFile(filepath.Join(baseDir, "update.lock"), []byte("103018"), 0o644); err != nil {
		t.Fatal(err)
	}
	server.markPanelUpdateReconciled()
	if actual := server.reconciledPanelUpdate(); actual != "103018" {
		t.Fatalf("unexpected reconciled update: %q", actual)
	}
}

func TestPendingPanelUpdateIgnoresStaleLock(t *testing.T) {
	baseDir := t.TempDir()
	server := NewServer(Config{BaseDir: baseDir})
	lockPath := filepath.Join(baseDir, "update.lock")
	if err := os.WriteFile(lockPath, []byte("103017"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-panelUpdateReconcileWindow - time.Minute)
	if err := os.Chtimes(lockPath, stale, stale); err != nil {
		t.Fatal(err)
	}
	if _, ok := server.pendingPanelUpdate(); ok {
		t.Fatal("stale update lock should not trigger a panel restart")
	}
}
