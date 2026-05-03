package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/systemctl"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (u *DockerService) OperateDocker(req dto.DockerOperation) error {
	req.Operation = strings.TrimSpace(req.Operation)
	if req.Operation == "" {
		return errors.New("operation is empty")
	}
	if isPodmanRuntimeConfigured() {
		return u.operatePodman(req.Operation)
	}
	if runtime.GOOS == "darwin" {
		return u.operateDockerOnDarwin(req.Operation)
	}
	service := "docker"
	h, err := systemctl.DefaultHandler(service)
	if err != nil {
		return err
	}
	if req.Operation == "stop" {
		socketHandle, err := systemctl.DefaultHandler("docker.socket")
		if err == nil {
			status, err := socketHandle.CheckStatus()
			if err == nil && status.IsActive {
				if std, err := socketHandle.ExecuteAction("stop"); err != nil {
					global.LOG.Errorf("handle stop docker.socket failed, err: %v", std)
				}
			}
		}
	}
	if req.Operation == "restart" {
		if err := validateDockerConfig(); err != nil {
			return err
		}
	}
	if isDockerSnapInstalled() {
		command := fmt.Sprintf("snap %s docker", req.Operation)
		stdout, err := cmd.Exec(command)
		if err != nil {
			return fmt.Errorf("failed to restart docker: %v", stdout)
		}
		return nil
	}
	result, err := h.ExecuteAction(req.Operation)
	if err != nil {
		return errors.New(result.Output)
	}
	return nil
}
func (u *DockerService) operateDockerOnDarwin(operation string) error {
	switch operation {
	case "start":
		if _, err := cmd.Exec(`open -a Docker`); err != nil {
			return fmt.Errorf("failed to start Docker Desktop: %w", err)
		}
		return u.waitDockerStatus(constant.StatusRunning, 90*time.Second)
	case "stop":
		if _, err := cmd.Exec(`osascript -e 'quit app "Docker"'`); err != nil {
			return fmt.Errorf("failed to stop Docker Desktop: %w", err)
		}
		return u.waitDockerStatus(constant.Stopped, 30*time.Second)
	case "restart":
		if err := validateDockerConfig(); err != nil {
			return err
		}
		if u.LoadDockerStatus() == constant.StatusRunning {
			if _, err := cmd.Exec(`osascript -e 'quit app "Docker"'`); err != nil {
				return fmt.Errorf("failed to stop Docker Desktop: %w", err)
			}
			if err := u.waitDockerStatus(constant.Stopped, 30*time.Second); err != nil {
				return err
			}
		}
		if _, err := cmd.Exec(`open -a Docker`); err != nil {
			return fmt.Errorf("failed to start Docker Desktop: %w", err)
		}
		return u.waitDockerStatus(constant.StatusRunning, 90*time.Second)
	default:
		return fmt.Errorf("unsupported docker operation: %s", operation)
	}
}
func (u *DockerService) waitDockerStatus(target string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if u.LoadDockerStatus() == target {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("docker status did not change to %s within %s", target, timeout)
		}
		time.Sleep(2 * time.Second)
	}
}
func isPodmanRuntimeConfigured() bool {
	return docker.ResolveRuntime(context.Background()).Kind == docker.RuntimePodman
}
func podmanRootlessHomeFromHost(host string) string {
	if !docker.IsRootlessPodmanHost(host) {
		return ""
	}
	host = strings.TrimSpace(host)
	if !strings.HasPrefix(host, "unix://") {
		return ""
	}
	sockPath := strings.TrimPrefix(host, "unix://")
	var uidStr string
	if idx := strings.Index(sockPath, "/run/user/"); idx >= 0 {
		rest := sockPath[idx+len("/run/user/"):]
		if cut := strings.Index(rest, "/"); cut > 0 {
			uidStr = rest[:cut]
		}
	}
	if uidStr == "" {
		if runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeDir != "" {
			runtimeDir = strings.TrimSuffix(runtimeDir, "/")
			if parts := strings.Split(runtimeDir, "/"); len(parts) > 0 {
				last := parts[len(parts)-1]
				if _, err := strconv.Atoi(last); err == nil {
					uidStr = last
				}
			}
		}
	}
	if uidStr == "" {
		return ""
	}
	usr, err := user.LookupId(uidStr)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(usr.HomeDir)
}
func ensureLinuxDockerConfigRuntime() error {
	if runtime.GOOS != "linux" {
		return errors.New("unsupported platform")
	}
	if isPodmanRuntimeConfigured() {
		return errors.New("podman runtime does not support docker daemon.json management")
	}
	return nil
}
func validateDockerConfig() error {
	if !cmd.Which("dockerd") {
		return nil
	}
	stdout, err := cmd.Exec("dockerd --validate")
	if strings.Contains(stdout, "unknown flag: --validate") {
		return nil
	}
	if err != nil || (stdout != "" && strings.TrimSpace(stdout) != "configuration OK") {
		return fmt.Errorf("docker configuration validation failed, err: %v", stdout)
	}
	return nil
}
func isDockerSnapInstalled() bool {
	stdout, err := cmd.Exec("which docker")
	if err != nil {
		return false
	}
	stdout = strings.TrimSpace(stdout)
	return strings.Contains(stdout, "snap")
}
func restartDocker() error {
	if isDockerSnapInstalled() {
		stdout, err := cmd.Exec("snap restart docker")
		if err != nil {
			return fmt.Errorf("failed to restart docker: %v", stdout)
		}
		return nil
	}
	return systemctl.Restart("docker")
}
