package helper

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type uninstallReport struct {
	lines    []string
	failures []string
}

func (r *uninstallReport) notef(format string, args ...interface{}) {
	r.lines = append(r.lines, fmt.Sprintf(format, args...))
}

func (r *uninstallReport) failf(format string, args ...interface{}) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *uninstallReport) removePath(path string, label string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	removed, err := removePathIfExists(path)
	if err != nil {
		r.failf("%s failed: %s (%v)", label, path, err)
		return
	}
	if removed {
		r.notef("%s: %s", label, path)
		return
	}
	r.notef("skip missing: %s", path)
}

func (r *uninstallReport) result() (string, error) {
	out := strings.Join(r.lines, "\n")
	if len(r.failures) == 0 {
		return out, nil
	}
	msg := strings.Join(r.failures, "; ")
	if out != "" {
		msg = out + "\n" + msg
	}
	return out, errors.New(msg)
}

func removePathIfExists(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func defaultGoPanelBinaryPath(baseDir string) string {
	return filepath.Join(strings.TrimSpace(baseDir), "gopanel")
}

func defaultGpAgentBinaryPath(baseDir string) string {
	return filepath.Join(strings.TrimSpace(baseDir), "gp-agent")
}

func defaultGpAgentSocketPath(baseDir string) string {
	return filepath.Join(strings.TrimSpace(baseDir), "agent", "run", "gp-agent.sock")
}
