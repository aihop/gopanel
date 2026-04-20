package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/gpc"
)

func (u *DockerService) operatePodman(operation string) error {
	switch runtime.GOOS {
	case "darwin":
		switch operation {
		case "start":
			return podmanMachineEnsureStarted()
		case "stop":
			out, err := cmd.Exec("podman machine stop")
			if err != nil {
				if strings.TrimSpace(out) != "" {
					return errors.New(out)
				}
				return err
			}
			return nil
		case "restart":
			_, _ = cmd.Exec("podman machine stop")
			return podmanMachineEnsureStarted()
		default:
			return fmt.Errorf("unsupported podman operation: %s", operation)
		}
	default:
		unit := "podman.socket"
		args := []string{operation}
		if operation == "restart" {
			args = []string{"restart"}
		}
		if _, err := exec.LookPath("systemctl"); err == nil {
			if os.Geteuid() != 0 {
				var userErr error
				if systemdUserBusAvailable() {
					out, err := cmd.Exec("systemctl --user " + strings.Join(args, " ") + " " + unit)
					if err == nil {
						if operation == "start" || operation == "restart" {
							return ensurePodmanAPIReady()
						}
						return nil
					}
					if strings.TrimSpace(out) != "" {
						userErr = fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
					} else {
						userErr = err
					}
				}
				_, gerr := gpc.Do(context.Background(), "GOPANEL_SERVICE_ACTION", map[string]interface{}{
					"op":   operation,
					"name": unit,
				})
				if gerr == nil {
					if operation == "start" || operation == "restart" {
						return ensurePodmanAPIReady()
					}
					return nil
				}
				if userErr != nil {
					return fmt.Errorf("%w; fallback gpc failed: %v", userErr, gerr)
				}
				return gerr
			}
			out, err := cmd.Exec("systemctl " + strings.Join(args, " ") + " " + unit)
			if err != nil {
				if strings.TrimSpace(out) != "" {
					return fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
				}
				return err
			}
			if operation == "start" || operation == "restart" {
				return ensurePodmanAPIReady()
			}
			return nil
		}
		if operation == "start" || operation == "restart" {
			return errors.New("podman 已安装但无法自动启动（缺少 systemctl）；请先手动启动 podman 的服务/Socket")
		}
		return nil
	}
}

func ensurePodmanAPIReady() error {
	baseCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	uid := os.Getuid()
	userHost := "unix:///run/user/" + strconv.Itoa(uid) + "/podman/podman.sock"
	rootHost := "unix:///run/podman/podman.sock"

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		okHost := ""
		if uid != 0 && docker.CanPingHost(baseCtx, userHost) {
			okHost = userHost
		} else if docker.CanPingHost(baseCtx, rootHost) {
			okHost = rootHost
		}
		if okHost != "" {
			settingRepo := repo.NewISettingRepo()
			curSetting, err := settingRepo.Get(settingRepo.WithByKey("DockerSockPath"))
			if err != nil {
				_ = settingRepo.UpdateOrCreate("DockerSockPath", okHost)
				return nil
			}
			cur := strings.TrimSpace(curSetting.Value)
			if shouldUpdateDockerSockPath(cur) && cur != okHost {
				_ = settingRepo.UpdateOrCreate("DockerSockPath", okHost)
			}
			return nil
		}
		select {
		case <-baseCtx.Done():
			return fmt.Errorf("podman.socket 已执行，但 Docker API socket 仍不可用（已尝试 %s, %s）", userHost, rootHost)
		case <-ticker.C:
		}
	}
}

func shouldUpdateDockerSockPath(cur string) bool {
	if strings.TrimSpace(cur) == "" {
		return true
	}
	if strings.Contains(cur, "/var/run/docker.sock") {
		return true
	}
	if strings.Contains(cur, "/run/user/") {
		return true
	}
	if strings.Contains(cur, "podman.sock") {
		return true
	}
	return false
}

func systemdUserBusAvailable() bool {
	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Geteuid())
	}
	if _, err := os.Stat(runtimeDir + "/bus"); err == nil {
		return true
	}
	return false
}

func podmanMachineEnsureStarted() error {
	hasAny, _ := podmanMachineHasAny()
	if !hasAny {
		out2, err2 := cmd.Exec("podman machine init")
		if err2 != nil {
			lower2 := strings.ToLower(out2)
			if strings.Contains(lower2, "already exists") || strings.Contains(lower2, "already been created") {
			} else if strings.TrimSpace(out2) != "" {
				return errors.New(out2)
			} else {
				return err2
			}
		}
	}

	out, err := cmd.Exec("podman machine start")
	if err == nil {
		return nil
	}
	lower := strings.ToLower(out)
	if strings.Contains(lower, "already running") {
		return nil
	}
	if strings.TrimSpace(out) != "" {
		return errors.New(out)
	}
	return err
}

func podmanMachineHasAny() (bool, error) {
	out, err := cmd.Exec("podman machine list --format json")
	if err == nil {
		var items []map[string]any
		if e := json.Unmarshal([]byte(out), &items); e == nil {
			return len(items) > 0, nil
		}
	}

	out, err = cmd.Exec("podman machine list")
	if err != nil {
		if strings.TrimSpace(out) != "" {
			return false, errors.New(out)
		}
		return false, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return false, nil
	}
	return true, nil
}

