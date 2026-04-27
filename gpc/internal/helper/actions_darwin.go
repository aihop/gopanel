//go:build darwin

package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (s *Server) actionChownBaseDir(ctx context.Context, params map[string]interface{}) error {
	_ = ctx
	_ = params
	return errors.New("unsupported platform")
}

func (s *Server) actionEnableForwarding(ctx context.Context, params map[string]interface{}) error {
	_ = ctx
	_ = params
	return errors.New("unsupported platform")
}

func (s *Server) actionRestartHost(ctx context.Context) error {
	_ = ctx
	return errors.New("unsupported platform")
}

func (s *Server) actionFirewallApply(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionGoPanelService(ctx context.Context, params map[string]interface{}) (string, error) {
	op := strings.ToLower(getString(params, "op"))
	if op == "" {
		return "", errors.New("invalid params: op is empty")
	}
	name := getString(params, "name")
	if name == "" {
		name = s.cfg.GoPanelServiceName
	}
	if name == "" {
		return "", errors.New("invalid params: name is empty")
	}

	switch op {
	case "status":
		if loaded, running, _, _ := launchctlStatus(ctx, name); loaded {
			if running {
				return "active", nil
			}
			return "inactive", nil
		}
		pid, ok := readPidfile(s.cfg.GoPanelPidfilePath)
		if !ok || !pidRunning(pid) {
			return "inactive", nil
		}
		return "active", nil
	case "start":
		if loaded, _, _, _ := launchctlStatus(ctx, name); loaded {
			if err := launchctlKickstart(ctx, name); err != nil {
				return "", err
			}
			return "started", nil
		}
		pid, ok := readPidfile(s.cfg.GoPanelPidfilePath)
		if ok && pidRunning(pid) {
			return "already running", nil
		}
		p, err := s.startGoPanelProcess(ctx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("started (pid=%d)", p), nil
	case "stop":
		if loaded, _, _, _ := launchctlStatus(ctx, name); loaded {
			if err := launchctlStop(ctx, name); err != nil {
				return "", err
			}
			return "stopped", nil
		}
		pid, ok := readPidfile(s.cfg.GoPanelPidfilePath)
		if !ok || !pidRunning(pid) {
			_ = os.Remove(s.cfg.GoPanelPidfilePath)
			return "inactive", nil
		}
		if err := stopPid(ctx, pid); err != nil {
			return "", err
		}
		_ = os.Remove(s.cfg.GoPanelPidfilePath)
		return "stopped", nil
	case "restart":
		if loaded, _, _, _ := launchctlStatus(ctx, name); loaded {
			if err := launchctlKickstart(ctx, name); err != nil {
				return "", err
			}
			return "restarted", nil
		}
		_, _ = s.actionGoPanelService(ctx, map[string]interface{}{"op": "stop", "name": name})
		return s.actionGoPanelService(ctx, map[string]interface{}{"op": "start", "name": name})
	default:
		return "", errors.New("invalid params: op")
	}
}

func (s *Server) actionGoPanelInfo(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = params

	type info struct {
		ServiceName string `json:"service_name"`
		Status      string `json:"status"`
		MainPID     string `json:"main_pid"`
		BaseDir     string `json:"base_dir"`
		ConfigPath  string `json:"config_path"`
		PidfilePath string `json:"pidfile_path"`
		AtUnixMs    int64  `json:"at_unix_ms"`
	}

	st, _ := s.actionGoPanelService(ctx, map[string]interface{}{"op": "status", "name": s.cfg.GoPanelServiceName})
	pidStr := ""
	if loaded, _, pid, _ := launchctlStatus(ctx, s.cfg.GoPanelServiceName); loaded && pid > 0 {
		pidStr = strconv.Itoa(pid)
	} else {
		pid, ok := readPidfile(s.cfg.GoPanelPidfilePath)
		if ok && pidRunning(pid) {
			pidStr = strconv.Itoa(pid)
		}
	}

	out := info{
		ServiceName: s.cfg.GoPanelServiceName,
		Status:      strings.TrimSpace(st),
		MainPID:     pidStr,
		BaseDir:     s.cfg.BaseDir,
		ConfigPath:  s.cfg.GoPanelConfigPath,
		PidfilePath: s.cfg.GoPanelPidfilePath,
		AtUnixMs:    time.Now().UnixMilli(),
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Server) actionPodmanSocketRepair(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionRepairPodmanShortName(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionSystemdEnableLinger(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionPodmanRegistriesGet(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionPodmanRegistriesSet(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionPodmanContainerJournalLogs(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionComposeInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

func (s *Server) actionRepairPodmanSubuid(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	_ = params
	return "", errors.New("unsupported platform")
}

var launchctlPidRe = regexp.MustCompile(`\bpid\s*=\s*([0-9]+)\b`)

func launchctlStatus(ctx context.Context, label string) (loaded bool, running bool, pid int, err error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return false, false, 0, nil
	}
	if _, lerr := exec.LookPath("launchctl"); lerr != nil {
		return false, false, 0, nil
	}

	for _, domain := range []string{fmt.Sprintf("gui/%d", os.Getuid()), "system"} {
		target := domain + "/" + label
		c := exec.CommandContext(ctx, "launchctl", "print", target)
		out, runErr := c.CombinedOutput()
		if runErr != nil {
			continue
		}
		loaded = true
		sout := string(out)
		ls := strings.ToLower(sout)
		if strings.Contains(ls, "state = running") || strings.Contains(ls, "state = waiting") {
			running = strings.Contains(ls, "state = running")
		}
		if m := launchctlPidRe.FindStringSubmatch(ls); len(m) == 2 {
			if n, e := strconv.Atoi(m[1]); e == nil && n > 0 {
				pid = n
				if pidRunning(pid) {
					running = true
				}
			}
		}
		return loaded, running, pid, nil
	}
	return false, false, 0, nil
}

func launchctlKickstart(ctx context.Context, label string) error {
	for _, domain := range []string{fmt.Sprintf("gui/%d", os.Getuid()), "system"} {
		target := domain + "/" + label
		c := exec.CommandContext(ctx, "launchctl", "kickstart", "-k", target)
		out, err := c.CombinedOutput()
		if err == nil {
			return nil
		}
		msg := strings.TrimSpace(string(out))
		if msg != "" && !strings.Contains(strings.ToLower(msg), "could not find service") {
			return errors.New(msg)
		}
	}
	return errors.New("launchctl service not found")
}

func launchctlStop(ctx context.Context, label string) error {
	for _, domain := range []string{fmt.Sprintf("gui/%d", os.Getuid()), "system"} {
		target := domain + "/" + label
		c := exec.CommandContext(ctx, "launchctl", "stop", target)
		out, err := c.CombinedOutput()
		if err == nil {
			return nil
		}
		msg := strings.TrimSpace(string(out))
		if msg != "" && !strings.Contains(strings.ToLower(msg), "could not find service") {
			return errors.New(msg)
		}
	}
	return errors.New("launchctl service not found")
}

func (s *Server) startGoPanelProcess(ctx context.Context) (int, error) {
	bin, err := findGoPanelBinary(s.cfg.GoPanelBinaryPath, s.cfg.BaseDir)
	if err != nil {
		return 0, err
	}
	if s.cfg.GoPanelConfigPath == "" {
		return 0, errors.New("gopanel config path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.cfg.GoPanelPidfilePath), 0755); err != nil {
		return 0, err
	}
	c := exec.CommandContext(ctx, bin, "--config", s.cfg.GoPanelConfigPath)
	c.Env = os.Environ()
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Start(); err != nil {
		return 0, err
	}
	pid := c.Process.Pid
	_ = c.Process.Release()
	_ = os.WriteFile(s.cfg.GoPanelPidfilePath, []byte(fmt.Sprintf("%d\n", pid)), 0644)
	return pid, nil
}

func findGoPanelBinary(configured string, baseDir string) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured != "" {
		if st, err := os.Stat(configured); err == nil && !st.IsDir() {
			return configured, nil
		}
		return "", errors.New("gopanel binary not found: " + configured)
	}
	if p, err := exec.LookPath("gopanel"); err == nil {
		return p, nil
	}
	candidates := []string{
		filepath.Join(baseDir, "gopanel"),
		filepath.Join(baseDir, "bin", "gopanel"),
		filepath.Join(baseDir, "server", "gopanel"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", errors.New("gopanel binary not found (set gopanel_binary_path or ensure gopanel in PATH)")
}

func readPidfile(p string) (int, bool) {
	p = strings.TrimSpace(p)
	if p == "" {
		return 0, false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return 0, false
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func pidRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func stopPid(ctx context.Context, pid int) error {
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
		if !pidRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	deadline = time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if !pidRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("failed to stop gopanel process")
}
