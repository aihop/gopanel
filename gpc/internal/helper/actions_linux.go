//go:build linux

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
	"time"
)

func (s *Server) actionChownBaseDir(ctx context.Context, params map[string]interface{}) error {
	pathStr := getString(params, "path")
	if pathStr == "" {
		pathStr = s.cfg.BaseDir
	}
	if pathStr == "" {
		return errors.New("invalid params: path/base_dir is empty")
	}
	uid, ok := getInt(params, "uid")
	if !ok {
		return errors.New("invalid params: uid is required")
	}
	gid, ok := getInt(params, "gid")
	if !ok {
		return errors.New("invalid params: gid is required")
	}

	target, err := filepath.Abs(pathStr)
	if err != nil {
		return err
	}
	base := ""
	if s.cfg.BaseDir != "" {
		base, _ = filepath.Abs(s.cfg.BaseDir)
	}
	if base != "" {
		rel, err := filepath.Rel(base, target)
		if err != nil {
			return err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return errors.New("invalid params: path out of base_dir")
		}
	}

	return filepath.WalkDir(target, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return os.Lchown(p, uid, gid)
	})
}

func (s *Server) actionEnableForwarding(ctx context.Context, params map[string]interface{}) error {
	_ = params
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1\n"), 0644); err != nil {
		return err
	}
	return nil
}

func (s *Server) actionRestartHost(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "reboot")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Server) actionFirewallApply(ctx context.Context, params map[string]interface{}) (string, error) {
	backend := strings.ToLower(getString(params, "backend"))
	if backend == "" {
		backend = "auto"
	}
	protoStr := strings.ToLower(getString(params, "protocol"))
	if protoStr == "" {
		protoStr = "tcp"
	}
	ports, err := getIntSlice(params, "ports")
	if err != nil {
		return "", err
	}
	if len(ports) == 0 {
		return "", errors.New("invalid params: ports is empty")
	}

	if backend == "auto" {
		if ufwActive(ctx) {
			backend = "ufw"
		} else {
			backend = "iptables"
		}
	}

	switch backend {
	case "ufw":
		if err := applyUfw(ctx, ports, protoStr); err != nil {
			return "", err
		}
		return "ufw applied", nil
	case "iptables":
		if err := applyIptables(ctx, ports, protoStr); err != nil {
			return "", err
		}
		return "iptables applied", nil
	case "nftables":
		if err := applyIptables(ctx, ports, protoStr); err != nil {
			return "", err
		}
		return "iptables applied", nil
	default:
		return "", errors.New("invalid params: backend")
	}
}

func ufwActive(ctx context.Context) bool {
	if _, err := exec.LookPath("ufw"); err != nil {
		return false
	}
	c := exec.CommandContext(ctx, "ufw", "status")
	out, err := c.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Status: active")
}

func applyUfw(ctx context.Context, ports []int, protoStr string) error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return err
	}
	for _, p := range ports {
		arg := fmt.Sprintf("%d/%s", p, protoStr)
		c := exec.CommandContext(ctx, "ufw", "--force", "allow", arg)
		out, err := c.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func applyIptables(ctx context.Context, ports []int, protoStr string) error {
	if _, err := exec.LookPath("iptables"); err != nil {
		return err
	}
	for _, p := range ports {
		dport := strconv.Itoa(p)
		check := exec.CommandContext(ctx, "iptables", "-C", "INPUT", "-p", protoStr, "--dport", dport, "-j", "ACCEPT")
		if err := check.Run(); err == nil {
			continue
		}
		add := exec.CommandContext(ctx, "iptables", "-I", "INPUT", "-p", protoStr, "--dport", dport, "-j", "ACCEPT")
		out, err := add.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
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
	if !validServiceName(name) {
		return "", errors.New("invalid params: name")
	}

	switch op {
	case "status":
		c := exec.CommandContext(ctx, "systemctl", "is-active", name)
		out, err := c.CombinedOutput()
		if err != nil {
			return strings.TrimSpace(string(out)), nil
		}
		return strings.TrimSpace(string(out)), nil
	case "start", "stop", "restart":
		c := exec.CommandContext(ctx, "systemctl", op, name)
		out, err := c.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out)), nil
	default:
		return "", errors.New("invalid params: op")
	}
}

func (s *Server) actionGoPanelInfo(ctx context.Context, params map[string]interface{}) (string, error) {
	name := getString(params, "name")
	if name == "" {
		name = s.cfg.GoPanelServiceName
	}
	if name == "" {
		return "", errors.New("invalid params: name is empty")
	}
	if !validServiceName(name) {
		return "", errors.New("invalid params: name")
	}

	type info struct {
		ServiceName string `json:"service_name"`
		Status      string `json:"status"`
		MainPID     string `json:"main_pid"`
		BaseDir     string `json:"base_dir"`
		ConfigPath  string `json:"config_path"`
		PidfilePath string `json:"pidfile_path"`
		AtUnixMs    int64  `json:"at_unix_ms"`
	}

	st, _ := s.actionGoPanelService(ctx, map[string]interface{}{"op": "status", "name": name})
	pid := strings.TrimSpace(systemctlShow(ctx, name, "MainPID"))

	out := info{
		ServiceName: name,
		Status:      strings.TrimSpace(st),
		MainPID:     pid,
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

func systemctlShow(ctx context.Context, name, prop string) string {
	c := exec.CommandContext(ctx, "systemctl", "show", name, "-p", prop, "--value")
	out, err := c.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var serviceNameRe = regexp.MustCompile(`^[A-Za-z0-9_.@-]+(\.(service|socket))?$`)

func validServiceName(s string) bool {
	return serviceNameRe.MatchString(s)
}

func (s *Server) actionPodmanSocketRepair(ctx context.Context, params map[string]interface{}) (string, error) {
	group := strings.TrimSpace(getString(params, "group"))
	if group == "" {
		return "", errors.New("invalid params: group is empty")
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return "", err
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if err := os.MkdirAll("/etc/systemd/system/podman.socket.d", 0755); err != nil {
		return "", err
	}
	content := "[Socket]\nSocketUser=root\nSocketGroup=" + group + "\nSocketMode=0660\nDirectoryMode=0755\n"
	if err := os.WriteFile("/etc/systemd/system/podman.socket.d/override.conf", []byte(content), 0644); err != nil {
		return "", err
	}

	steps := [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "stop", "podman.socket"},
		{"rm", "-f", "/run/podman/podman.sock"},
		{"systemctl", "start", "podman.socket"},
	}
	var outs []string
	for _, args := range steps {
		c := exec.CommandContext(ctx, args[0], args[1:]...)
		out, err := c.CombinedOutput()
		s := strings.TrimSpace(string(out))
		if s != "" {
			outs = append(outs, s)
		}
		if err != nil {
			if args[0] == "systemctl" && len(args) >= 3 && args[1] == "stop" && args[2] == "podman.socket" {
				continue
			}
			return strings.Join(outs, "\n"), fmt.Errorf("%w: %s", err, s)
		}
	}
	return strings.Join(outs, "\n"), nil
}

func (s *Server) actionSystemdEnableLinger(ctx context.Context, params map[string]interface{}) (string, error) {
	uid, ok := getInt(params, "uid")
	if !ok {
		return "", errors.New("invalid params: uid is required")
	}
	if _, err := exec.LookPath("loginctl"); err != nil {
		return "", err
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	c := exec.CommandContext(ctx, "loginctl", "enable-linger", strconv.Itoa(uid))
	out, err := c.CombinedOutput()
	sout := strings.TrimSpace(string(out))
	if err != nil {
		return sout, fmt.Errorf("%w: %s", err, sout)
	}
	return sout, nil
}
