//go:build windows

package helper

import (
	"context"
	"path/filepath"
)

func (s *Server) actionGoPanelUninstall(ctx context.Context, params map[string]interface{}) (string, error) {
	return UninstallGoPanel(ctx, s.cfg)
}

// UninstallGoPanel stops services, removes binaries, startup files,
// PID file, and socket for both gopanel and gp-agent on Windows.
func UninstallGoPanel(ctx context.Context, cfg Config) (string, error) {
	report := &uninstallReport{}
	// Windows uninstallation logic is currently not fully implemented in GPC.
	// Typically, Windows uninstallation is handled by install.ps1 / uninstall.ps1.
	
	gopanelBin := defaultGoPanelBinaryPath(cfg.BaseDir) + ".exe"
	gpAgentBin := defaultGpAgentBinaryPath(cfg.BaseDir) + ".exe"

	report.removePath(gopanelBin, "removed gopanel binary")
	report.removePath(gpAgentBin, "removed gp-agent binary")

	return report.result()
}

// CleanupGPC removes the GPC socket or pipe on Windows.
func CleanupGPC(ctx context.Context, cfg Config) (string, error) {
	report := &uninstallReport{}
	report.removePath(filepath.Join(cfg.BaseDir, "gpc.sock"), "removed gpc socket")
	return report.result()
}
