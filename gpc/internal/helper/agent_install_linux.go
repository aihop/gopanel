//go:build linux

package helper

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type agentEnsureResult struct {
	Status      string `json:"status"`
	BaseDir     string `json:"base_dir"`
	ServiceName string `json:"service_name"`
	Output      string `json:"output"`
	AtUnixMs    int64  `json:"at_unix_ms"`
}

func (s *Server) actionGoPanelAgentEnsure(ctx context.Context, params map[string]interface{}) (string, error) {
	baseDir := strings.TrimSpace(getString(params, "base_dir"))
	if baseDir == "" {
		baseDir = strings.TrimSpace(s.cfg.BaseDir)
	}
	if baseDir == "" {
		return "", errors.New("base_dir is empty")
	}
	serviceName := strings.TrimSpace(getString(params, "service_name"))
	if serviceName == "" {
		serviceName = "gp-agent.service"
	}

	binPath := filepath.Join(baseDir, "gp-agent")
	if _, err := os.Stat(binPath); err != nil {
		downloadURL := strings.TrimSpace(getString(params, "download_url"))
		if downloadURL == "" {
			return "", errors.New("gp-agent binary not found, download_url is required")
		}
		out, err := s.actionGoPanelAgentInstall(ctx, params)
		if err != nil {
			return out, err
		}
	}

	out, err := systemctl(ctx, "restart", serviceName)
	if err != nil {
		return "", err
	}

	res := agentEnsureResult{
		Status:      "restarted",
		BaseDir:     baseDir,
		ServiceName: serviceName,
		Output:      out,
		AtUnixMs:    time.Now().UnixMilli(),
	}
	b, _ := json.Marshal(res)
	return string(b), nil
}

func (s *Server) actionGoPanelAgentInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	baseDir := strings.TrimSpace(getString(params, "base_dir"))
	if baseDir == "" {
		baseDir = strings.TrimSpace(s.cfg.BaseDir)
	}
	if baseDir == "" {
		return "", errors.New("base_dir is empty")
	}
	serviceName := strings.TrimSpace(getString(params, "service_name"))
	if serviceName == "" {
		serviceName = "gp-agent.service"
	}
	downloadURL := strings.TrimSpace(getString(params, "download_url"))
	if downloadURL == "" {
		return "", errors.New("download_url is empty")
	}
	if err := validateDownloadURL(downloadURL); err != nil {
		return "", err
	}

	runtimeUser := strings.TrimSpace(getString(params, "runtime_user"))
	if runtimeUser == "" {
		runtimeUser = strings.TrimSpace(readSystemdUser("gopanel.service"))
	}
	if runtimeUser == "" {
		runtimeUser = "root"
	}

	tmpDir, err := os.MkdirTemp("", "gp-agent-install-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "pkg.tar.gz")
	if err := downloadToFile(ctx, downloadURL, archivePath); err != nil {
		return "", err
	}

	extractDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return "", err
	}
	if err := extractTarGz(archivePath, extractDir); err != nil {
		return "", err
	}

	srcBin, err := findFileByName(extractDir, "gp-agent")
	if err != nil {
		return "", err
	}

	dstBin := filepath.Join(baseDir, "gp-agent")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", err
	}
	if err := copyFile(srcBin, dstBin, 0o755); err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Join(baseDir, "agent", "run"), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(baseDir, "caddy", "data"), 0o755); err != nil {
		return "", err
	}

	unitPath := filepath.Join("/etc/systemd/system", serviceName)
	unitContent := buildGpAgentSystemdUnit(baseDir, runtimeUser)
	if err := os.WriteFile(unitPath, []byte(unitContent), 0o644); err != nil {
		return "", err
	}

	if _, err := systemctl(ctx, "daemon-reload"); err != nil {
		return "", err
	}
	if _, err := systemctl(ctx, "enable", serviceName); err != nil {
		return "", err
	}
	out, err := systemctl(ctx, "restart", serviceName)
	if err != nil {
		return "", err
	}

	res := agentEnsureResult{
		Status:      "installed",
		BaseDir:     baseDir,
		ServiceName: serviceName,
		Output:      out,
		AtUnixMs:    time.Now().UnixMilli(),
	}
	b, _ := json.Marshal(res)
	return string(b), nil
}

func validateDownloadURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "https" {
		return errors.New("download_url must be https")
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return errors.New("download_url host is empty")
	}
	if !isAllowedDownloadHost(host) {
		return errors.New("download_url host not allowed")
	}
	return nil
}

func isAllowedDownloadHost(host string) bool {
	for _, pattern := range allowedDownloadHostPatterns() {
		if matchesDownloadHostPattern(host, pattern) {
			return true
		}
	}
	return false
}

func allowedDownloadHostPatterns() []string {
	patterns := []string{
		"gopanel.cn",
		".gopanel.cn",
		"aihop.io",
		".aihop.io",
		"github.com",
		"github-releases.githubusercontent.com",
		"objects.githubusercontent.com",
		"githubusercontent.com",
		".githubusercontent.com",
		".aliyuncs.com",
	}
	if extra := strings.TrimSpace(os.Getenv("GPAGENT_ALLOWED_DOWNLOAD_HOSTS")); extra != "" {
		for _, item := range strings.Split(extra, ",") {
			if value := strings.ToLower(strings.TrimSpace(item)); value != "" {
				patterns = append(patterns, value)
			}
		}
	}
	return patterns
}

func matchesDownloadHostPattern(host, pattern string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	if strings.HasPrefix(pattern, ".") {
		return strings.HasSuffix(host, pattern)
	}
	return host == pattern
}

func downloadToFile(ctx context.Context, rawURL, dst string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("download failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractTarGz(archivePath, dstDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(h.Name)
		if filepath.IsAbs(name) || name == "." || name == string(os.PathSeparator) || name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) {
			continue
		}
		target := filepath.Join(dstDir, name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(h.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
		}
	}
	return nil
}

func findFileByName(root, name string) (string, error) {
	errFound := errors.New("found")
	var found string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == name {
			found = p
			return errFound
		}
		return nil
	})
	if err != nil && !errors.Is(err, errFound) {
		return "", err
	}
	if found == "" {
		return "", errors.New("gp-agent binary not found in package")
	}
	return found, nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func buildGpAgentSystemdUnit(baseDir, runtimeUser string) string {
	b := strings.Builder{}
	b.WriteString("[Unit]\n")
	b.WriteString("Description=GoPanel Agent (gp-agent)\n")
	b.WriteString("After=network.target docker.socket podman.socket\n")
	b.WriteString("Wants=docker.socket podman.socket\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString("User=" + runtimeUser + "\n")
	b.WriteString("Group=" + runtimeUser + "\n")
	b.WriteString("WorkingDirectory=" + baseDir + "\n")
	b.WriteString("ExecStart=" + filepath.Join(baseDir, "gp-agent") + " service --base-dir " + baseDir + "\n")
	b.WriteString("Restart=always\n")
	b.WriteString("RestartSec=2\n\n")
	b.WriteString(`Environment="HOME=` + baseDir + `"` + "\n")
	b.WriteString(`Environment="CADDY_DATA_DIR=` + filepath.Join(baseDir, "caddy", "data") + `"` + "\n\n")
	b.WriteString("AmbientCapabilities=CAP_NET_BIND_SERVICE\n")
	b.WriteString("CapabilityBoundingSet=CAP_NET_BIND_SERVICE\n\n")
	b.WriteString("LimitNOFILE=65535\n")
	b.WriteString("OOMScoreAdjust=-100\n\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")
	return b.String()
}

func systemctl(ctx context.Context, args ...string) (string, error) {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return "", err
	}
	c := exec.CommandContext(ctx, "systemctl", args...)
	out, err := c.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		if s == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, s)
	}
	return s, nil
}

func readSystemdUser(serviceName string) string {
	unitPath := filepath.Join("/etc/systemd/system", serviceName)
	b, err := os.ReadFile(unitPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "User=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "User="))
		}
	}
	return ""
}
