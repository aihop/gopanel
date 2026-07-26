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

	// 二进制不在 → 走完整安装（需要 download_url）
	binPath := filepath.Join(baseDir, "gp-agent")
	if _, err := os.Stat(binPath); err != nil {
		downloadURL := strings.TrimSpace(getString(params, "download_url"))
		if downloadURL == "" {
			return "", errors.New("gp-agent binary not found, download_url is required")
		}
		return s.actionGoPanelAgentInstall(ctx, params)
	}

	// 二进制在、但 systemd unit 不在：以前这里直接 systemctl restart，
	// 必然报 "Unit gp-agent.service not found"，「一键初始化」永远修不好这种机器。
	// （install.sh 在安装时若还没下载到 gp-agent 二进制，会跳过 unit 配置，
	//   见 install.sh 的「未检测到 gp-agent 二进制，跳过 gp-agent service 配置」）
	// 这种情况根本不需要重新下载，补一个 unit 就行。
	unitCreated, err := ensureGpAgentUnit(ctx, baseDir, serviceName, params)
	if err != nil {
		return "", err
	}

	out, err := systemctl(ctx, "restart", serviceName)
	if err != nil {
		return "", err
	}

	status := "restarted"
	if unitCreated {
		status = "unit_created_and_restarted"
	}
	res := agentEnsureResult{
		Status:      status,
		BaseDir:     baseDir,
		ServiceName: serviceName,
		Output:      strings.TrimSpace(unitNote(unitCreated, serviceName) + "\n" + out),
		AtUnixMs:    time.Now().UnixMilli(),
	}
	b, _ := json.Marshal(res)
	return string(b), nil
}

func unitNote(created bool, serviceName string) string {
	if created {
		return "created missing systemd unit: /etc/systemd/system/" + serviceName
	}
	return ""
}

// ensureGpAgentUnit 保证 systemd unit 存在。
//
// 刻意「只在缺失时才写」：已经存在的 unit 可能带着当前这台机器跑起来所必需的
// 环境（rootless podman 的 XDG_RUNTIME_DIR / DBUS_SESSION_BUS_ADDRESS、
// install.sh 写的 HOME 等）。以前无条件覆盖，rootless 安装做一次「一键初始化」
// 就会把这些环境弄丢，agent 起来了却连不上 podman —— 容器/网站功能全挂。
// 返回 true 表示本次新建了 unit。
func ensureGpAgentUnit(ctx context.Context, baseDir, serviceName string, params map[string]interface{}) (bool, error) {
	unitPath := filepath.Join("/etc/systemd/system", serviceName)
	if st, err := os.Stat(unitPath); err == nil && !st.IsDir() {
		return false, nil // 已存在：保持原样，不动
	}

	runtimeUser := strings.TrimSpace(getString(params, "runtime_user"))
	if runtimeUser == "" {
		// unit 缺失时读不到自己的 User=，退回面板服务的运行用户
		runtimeUser = strings.TrimSpace(readSystemdUser("gopanel.service"))
	}
	if runtimeUser == "" {
		runtimeUser = "root"
	}

	if err := os.WriteFile(unitPath, []byte(buildGpAgentSystemdUnit(baseDir, runtimeUser)), 0o644); err != nil {
		return false, err
	}
	if _, err := systemctl(ctx, "daemon-reload"); err != nil {
		return true, err
	}
	if _, err := systemctl(ctx, "enable", serviceName); err != nil {
		return true, err
	}
	return true, nil
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

	// unit 只在缺失时生成：更新二进制不该顺手改写这台机器已经跑通的 unit
	// （尤其是 rootless podman 依赖的 XDG_RUNTIME_DIR / DBUS_SESSION_BUS_ADDRESS）
	unitParams := map[string]interface{}{"runtime_user": runtimeUser}
	if _, err := ensureGpAgentUnit(ctx, baseDir, serviceName, unitParams); err != nil {
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
		"gitcode.com",
		".gitcode.com",
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
