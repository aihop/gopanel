package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/systemctl"
)

func RestartGoPanel() error {
	return RestartGoPanelWithDelay(1 * time.Second)
}

func RestartGoPanelWithDelay(delay time.Duration) error {
	go func() {
		time.Sleep(delay)
		if err := restartGoPanelNow(); err != nil {
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

func restartGoPanelNow() error {
	if runtime.GOOS == "linux" {
		if err := systemctl.Restart("gopanel"); err == nil {
			return nil
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable failed: %w", err)
	}
	workDir := filepath.Dir(exePath)

	switch runtime.GOOS {
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
		return fmt.Errorf("unsupported os for panel restart: %s", runtime.GOOS)
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
