//go:build darwin

package helper

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

func (s *Server) actionGoPanelUninstall(ctx context.Context, params map[string]interface{}) (string, error) {
	return UninstallGoPanel(ctx, s.cfg)
}

// UninstallGoPanel stops services, removes binaries, startup files,
// PID file, and socket for both gopanel and gp-agent on macOS.
func UninstallGoPanel(ctx context.Context, cfg Config) (string, error) {
	report := &uninstallReport{}
	for _, item := range []struct {
		label string
		plist string
	}{
		{label: "io.aihop.gopanel", plist: "/Library/LaunchDaemons/io.aihop.gopanel.plist"},
		{label: "io.aihop.gp-agent", plist: "/Library/LaunchDaemons/io.aihop.gp-agent.plist"},
	} {
		if err := launchctlBootoutPlist(ctx, item.plist); err != nil {
			report.notef("skip bootout %s: %v", item.label, err)
		} else {
			report.notef("booted out service: %s", item.label)
		}
	}

	if pid, ok := readPidfile(cfg.GoPanelPidfilePath); ok && pidRunning(pid) {
		if err := stopPid(ctx, pid); err != nil {
			report.notef("skip stopping pid %d: %v", pid, err)
		} else {
			report.notef("stopped process pid=%d", pid)
		}
	}

	report.removePath(defaultGoPanelBinaryPath(cfg.BaseDir), "removed binary")
	report.removePath(defaultGpAgentBinaryPath(cfg.BaseDir), "removed binary")
	report.removePath("/Library/LaunchDaemons/io.aihop.gopanel.plist", "removed startup file")
	report.removePath("/Library/LaunchDaemons/io.aihop.gp-agent.plist", "removed startup file")
	report.removePath(cfg.GoPanelPidfilePath, "removed runtime file")
	report.removePath(defaultGpAgentSocketPath(cfg.BaseDir), "removed runtime file")

	return report.result()
}

// CleanupGPC removes gpc socket. On macOS gpc has no launchd service
// and binary location varies, so only the socket file is cleaned.
func CleanupGPC(ctx context.Context, cfg Config) (string, error) {
	report := &uninstallReport{}
	report.removePath(filepath.Join(cfg.BaseDir, "gpc.sock"), "removed gpc socket")
	return report.result()
}

func launchctlBootoutPlist(ctx context.Context, plistPath string) error {
	if _, err := exec.LookPath("launchctl"); err != nil {
		return err
	}
	c := exec.CommandContext(ctx, "launchctl", "bootout", "system", plistPath)
	out, err := c.CombinedOutput()
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		return err
	}
	return execErrorAllowingMissing(msg)
}

func execErrorAllowingMissing(msg string) error {
	ls := strings.ToLower(strings.TrimSpace(msg))
	switch {
	case ls == "":
		return nil
	case strings.Contains(ls, "could not find service"),
		strings.Contains(ls, "input/output error"),
		strings.Contains(ls, "service is not loaded"),
		strings.Contains(ls, "no such process"),
		strings.Contains(ls, "not found"):
		return nil
	default:
		return &stringError{s: msg}
	}
}

type stringError struct {
	s string
}

func (e *stringError) Error() string {
	return e.s
}
