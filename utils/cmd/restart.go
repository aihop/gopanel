package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/aihop/gopanel/utils/systemctl"
)

const goPanelServiceName = "gopanel.service"

var (
	panelRestartOS            = runtime.GOOS
	panelRestartGPCDo         = gpc.Do
	panelRestartServiceExists = systemctl.IsExist
	panelRestartStandalone    = restartStandaloneGoPanelNow
)

func RestartGoPanel() error {
	return RestartGoPanelWithDelay(1 * time.Second)
}

func RestartGoPanelWithDelay(delay time.Duration) error {
	restart, err := prepareGoPanelRestart()
	if err != nil {
		return err
	}
	go func() {
		time.Sleep(delay)
		if err := restart(); err != nil && global.LOG != nil {
			global.LOG.Errorf("restart gopanel failed: %v", err)
		}
	}()
	return nil
}

func RestartServer() error {
	return RestartServerWithDelay(1 * time.Second)
}

func RestartServerWithDelay(delay time.Duration) error {
	go func() {
		time.Sleep(delay)
		if err := restartServerNow(); err != nil {
			global.LOG.Errorf("restart server failed: %v", err)
		}
	}()
	return nil
}

func prepareGoPanelRestart() (func() error, error) {
	if panelRestartOS != "linux" && panelRestartOS != "darwin" {
		return panelRestartStandalone, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, gpcErr := panelRestartGPCDo(ctx, "GOPANEL_SERVICE_ACTION", map[string]interface{}{
		"op":   "status",
		"name": goPanelServiceName,
	})
	if gpcErr == nil && resp != nil && strings.EqualFold(strings.TrimSpace(resp.Output), "active") {
		return func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := panelRestartGPCDo(ctx, "GOPANEL_SERVICE_ACTION", map[string]interface{}{
				"op":   "restart",
				"name": goPanelServiceName,
			})
			return err
		}, nil
	}

	managed, err := panelRestartServiceExists("gopanel")
	if err != nil {
		return nil, fmt.Errorf("检查 GoPanel 服务状态失败: %w", err)
	}
	if managed {
		return nil, fmt.Errorf("面板由系统服务托管，但 gpc helper 不可用；请先重启 gpc 服务后重试")
	}
	return panelRestartStandalone, nil
}

func restartStandaloneGoPanelNow() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable failed: %w", err)
	}
	workDir := filepath.Dir(exePath)

	switch panelRestartOS {
	case "windows":
		cmd := exec.Command("cmd", "/C", "start", "", exePath)
		cmd.Dir = workDir
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start new windows process failed: %w", err)
		}
	case "darwin", "linux":
		shellCmd := fmt.Sprintf("cd %q && nohup %q >/dev/null 2>&1 &", workDir, exePath)
		cmd := exec.Command("sh", "-c", shellCmd)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start new daemon process failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported os for panel restart: %s", panelRestartOS)
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
	return nil
}

func restartServerNow() error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("shutdown", "/r", "/t", "0").Start()
	case "linux":
		return runPrivilegedCommand("reboot")
	case "darwin":
		if err := runPrivilegedCommand("shutdown", "-r", "now"); err == nil {
			return nil
		}
		return runPrivilegedCommand("reboot")
	default:
		return fmt.Errorf("unsupported os for server restart: %s", runtime.GOOS)
	}
}

func runPrivilegedCommand(name string, args ...string) error {
	if err := exec.Command(name, args...).Run(); err == nil {
		return nil
	}
	if runtime.GOOS != "windows" && HasNoPasswordSudo() {
		sudoArgs := append([]string{"-n", name}, args...)
		if err := exec.Command("sudo", sudoArgs...).Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("permission denied: restart requires elevated privileges")
}
