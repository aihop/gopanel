//go:build linux

package helper

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (s *Server) actionGoPanelUninstall(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = params

	report := &uninstallReport{}
	serviceNames := uniqueNonEmptyStrings("gopanel.service", s.cfg.GoPanelServiceName, "gp-agent.service")

	if _, err := exec.LookPath("systemctl"); err == nil {
		for _, name := range serviceNames {
			out, runErr := systemctl(ctx, "disable", "--now", name)
			if runErr != nil {
				report.notef("skip disable/stop %s: %v", name, runErr)
				continue
			}
			if strings.TrimSpace(out) == "" {
				report.notef("disabled and stopped service: %s", name)
			} else {
				report.notef("disabled and stopped service: %s (%s)", name, strings.TrimSpace(out))
			}
		}
	} else {
		report.notef("skip systemd operations: %v", err)
	}

	if pid, ok := readPidfileForUninstall(s.cfg.GoPanelPidfilePath); ok && pidRunningForUninstall(pid) {
		if err := stopPidForUninstall(ctx, pid); err != nil {
			report.notef("skip stopping pid %d: %v", pid, err)
		} else {
			report.notef("stopped process pid=%d", pid)
		}
	}

	report.removePath(defaultGoPanelBinaryPath(s.cfg.BaseDir), "removed binary")
	report.removePath(defaultGpAgentBinaryPath(s.cfg.BaseDir), "removed binary")
	report.removePath("/etc/systemd/system/gopanel.service", "removed startup file")
	report.removePath("/etc/systemd/system/gp-agent.service", "removed startup file")
	report.removePath(s.cfg.GoPanelPidfilePath, "removed runtime file")
	report.removePath(defaultGpAgentSocketPath(s.cfg.BaseDir), "removed runtime file")

	if _, err := exec.LookPath("systemctl"); err == nil {
		if out, runErr := systemctl(ctx, "daemon-reload"); runErr != nil {
			report.notef("skip systemctl daemon-reload: %v", runErr)
		} else if strings.TrimSpace(out) == "" {
			report.notef("reloaded systemd daemon")
		} else {
			report.notef("reloaded systemd daemon (%s)", strings.TrimSpace(out))
		}
	}

	return report.result()
}

func readPidfileForUninstall(p string) (int, bool) {
	p = strings.TrimSpace(p)
	if p == "" {
		return 0, false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func pidRunningForUninstall(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func stopPidForUninstall(ctx context.Context, pid int) error {
	if pid <= 0 {
		return nil
	}
	_ = syscall.Kill(pid, syscall.SIGTERM)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !pidRunningForUninstall(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	deadline = time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if !pidRunningForUninstall(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("failed to stop gopanel process")
}

func uniqueNonEmptyStrings(items ...string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		item = filepath.Clean(item)
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
