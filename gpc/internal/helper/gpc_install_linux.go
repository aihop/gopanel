//go:build linux

package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type gpcInstallResult struct {
	Status           string `json:"status"`
	SourcePath       string `json:"source_path"`
	TargetPath       string `json:"target_path"`
	ServiceName      string `json:"service_name"`
	RestartScheduled bool   `json:"restart_scheduled"`
	AtUnixMs         int64  `json:"at_unix_ms"`
}

func (s *Server) actionGoPanelGPCInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	sourcePath := strings.TrimSpace(getString(params, "source_path"))
	if sourcePath == "" {
		return "", errors.New("invalid params: source_path is empty")
	}
	targetPath := strings.TrimSpace(getString(params, "target_path"))
	if targetPath == "" {
		targetPath = "/usr/local/bin/gpc"
	}
	serviceName := strings.TrimSpace(getString(params, "service_name"))
	if serviceName == "" {
		serviceName = "gpc.service"
	}
	if !validServiceName(serviceName) {
		return "", errors.New("invalid params: service_name")
	}

	srcAbs, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", err
	}
	baseAbs, err := filepath.Abs(strings.TrimSpace(s.cfg.BaseDir))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(baseAbs, srcAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("invalid params: source_path out of base_dir")
	}
	if filepath.Base(srcAbs) != "gpc" {
		return "", errors.New("invalid params: source_path must point to gpc binary")
	}
	if filepath.Clean(targetPath) != "/usr/local/bin/gpc" {
		return "", errors.New("invalid params: target_path")
	}

	st, err := os.Stat(srcAbs)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", errors.New("invalid params: source_path is a directory")
	}

	targetDir := filepath.Dir(targetPath)
	tmpFile, err := os.CreateTemp(targetDir, ".gpc_tmp_*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	if err := copyExecutableFile(srcAbs, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Chown(tmpPath, 0, 0); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := scheduleLinuxServiceRestart(serviceName); err != nil {
		return "", err
	}

	out := gpcInstallResult{
		Status:           "installed",
		SourcePath:       srcAbs,
		TargetPath:       targetPath,
		ServiceName:      serviceName,
		RestartScheduled: true,
		AtUnixMs:         time.Now().UnixMilli(),
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func copyExecutableFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, 0o755)
}

func scheduleLinuxServiceRestart(serviceName string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("sleep 1; systemctl restart %q >/dev/null 2>&1", serviceName))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
