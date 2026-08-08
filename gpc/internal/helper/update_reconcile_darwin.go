//go:build darwin

package helper

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const panelUpdateReconcileWindow = 10 * time.Minute

func (s *Server) reconcilePendingPanelUpdate(ctx context.Context) {
	timer := time.NewTimer(7 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	pending, ok := s.pendingPanelUpdate()
	if !ok || s.reconciledPanelUpdate() == pending {
		return
	}
	_, _ = s.actionGoPanelService(ctx, map[string]interface{}{
		"op":   "restart",
		"name": s.cfg.GoPanelServiceName,
	})
}

func (s *Server) pendingPanelUpdate() (string, bool) {
	lockPath := filepath.Join(s.cfg.BaseDir, "update.lock")
	info, err := os.Stat(lockPath)
	if err != nil || time.Since(info.ModTime()) > panelUpdateReconcileWindow {
		return "", false
	}
	content, err := os.ReadFile(lockPath)
	value := strings.TrimSpace(string(content))
	return value, err == nil && value != ""
}

func (s *Server) reconciledPanelUpdate() string {
	content, err := os.ReadFile(s.panelUpdateReconciledPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func (s *Server) markPanelUpdateReconciled() {
	pending, ok := s.pendingPanelUpdate()
	if !ok {
		return
	}
	markerPath := s.panelUpdateReconciledPath()
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(markerPath, []byte(pending), 0o644)
}

func (s *Server) panelUpdateReconciledPath() string {
	return filepath.Join(s.cfg.BaseDir, "run", "update-restart.lock")
}
