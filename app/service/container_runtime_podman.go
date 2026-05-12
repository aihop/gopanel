package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/gpc"
)

func EnsurePodmanAPIReady() error {
	return ensurePodmanAPIReady()
}

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
				syncPreferredUserPodmanHost()
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

func podmanRootlessExpected(host string) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if docker.IsRootlessPodmanHost(host) {
		return true
	}
	if os.Geteuid() != 0 {
		return true
	}
	return false
}

func podmanServiceActiveForHost(ctx context.Context, host string) bool {
	if podmanRootlessExpected(host) {
		return podmanSocketUserServiceActive(ctx)
	}
	return podmanSocketServiceActive(ctx)
}

func syncPreferredUserPodmanHost() {
	if runtime.GOOS != "linux" || os.Geteuid() == 0 {
		return
	}
	candidates := docker.PodmanLinuxUserCandidateHosts()
	if len(candidates) == 0 {
		return
	}
	preferred := strings.TrimSpace(candidates[0])
	if preferred == "" {
		return
	}
	_ = repo.NewISettingRepo().UpdateOrCreate("DockerSockPath", preferred)
}

func ensurePodmanAPIReady() error {
	baseCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	candidates := docker.PodmanLinuxCandidateHosts()
	if docker.StrictCurrentUserRootlessPodman() {
		candidates = docker.PodmanLinuxUserCandidateHosts()
	}
	if len(candidates) == 0 {
		return errors.New("no podman socket candidates found for current linux environment")
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	lastErrs := make(map[string]string)

	for {
		attemptCtx, cancel := context.WithTimeout(baseCtx, 600*time.Millisecond)
		okHost := firstReachablePodmanHost(attemptCtx, candidates, lastErrs)
		cancel()

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
			systemActive := podmanSocketServiceActive(context.Background())
			userActive := podmanSocketUserServiceActive(context.Background())
			var tries []string
			for _, host := range candidates {
				tries = append(tries, fmt.Sprintf("- try: %s -> %s", host, strings.TrimSpace(lastErrs[host])))
			}
			msg := fmt.Sprintf("podman socket 仍不可用。\n- systemd podman.socket active: %v\n- systemd --user podman.socket active: %v\n%s", systemActive, userActive, strings.Join(tries, "\n"))
			if anyRootlessCandidate(candidates) {
				msg += "\n建议：rootless 场景优先启用 linger，并启动用户级 podman.socket"
			} else {
				msg += "\n建议：检查 podman.service/podman.socket 日志与 /run/podman/podman.sock 权限"
			}
			return errors.New(msg)
		case <-ticker.C:
		}
	}
}

func firstReachablePodmanHost(ctx context.Context, candidates []string, lastErrs map[string]string) string {
	if len(candidates) == 0 {
		return ""
	}
	// 普通用户场景显式优先 rootless candidate，避免 system socket 仍可用时持续保留 rootful 配置。
	if os.Geteuid() != 0 {
		for _, host := range candidates {
			if !docker.IsRootlessPodmanHost(host) {
				continue
			}
			if err := docker.PingHost(ctx, host); err == nil {
				return host
			} else if lastErrs != nil {
				lastErrs[host] = err.Error()
			}
		}
	}
	for _, host := range candidates {
		if err := docker.PingHost(ctx, host); err == nil {
			return host
		} else if lastErrs != nil {
			lastErrs[host] = err.Error()
		}
	}
	return ""
}

func podmanSocketServiceActive(ctx context.Context) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	c, cancel := context.WithTimeout(baseCtx, 1200*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(c, "systemctl", "is-active", "podman.socket")
	out, err := cmd.CombinedOutput()
	_ = err
	return strings.TrimSpace(string(out)) == "active"
}

func podmanSocketUserServiceActive(ctx context.Context) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Geteuid())
	}
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	c, cancel := context.WithTimeout(baseCtx, 1200*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(c, "systemctl", "--user", "is-active", "podman.socket")
	cmd.Env = append(os.Environ(),
		"XDG_RUNTIME_DIR="+runtimeDir,
		"DBUS_SESSION_BUS_ADDRESS=unix:path="+filepath.Join(runtimeDir, "bus"),
	)
	out, err := cmd.CombinedOutput()
	_ = err
	return strings.TrimSpace(string(out)) == "active"
}

func anyRootlessCandidate(hosts []string) bool {
	for _, host := range hosts {
		if docker.IsRootlessPodmanHost(host) {
			return true
		}
	}
	return false
}

func shouldUpdateDockerSockPath(cur string) bool {
	if strings.TrimSpace(cur) == "" {
		return true
	}
	if strings.Contains(cur, "/var/run/docker.sock") {
		return true
	}
	// 普通用户场景下，如果当前仍落在系统级 podman.sock，允许切回 rootless。
	// 否则“修复 rootless socket 成功”后，Setting 可能仍然停留在 /run/podman/podman.sock。
	if runtime.GOOS == "linux" && os.Geteuid() != 0 && strings.Contains(cur, "/run/podman/podman.sock") {
		return true
	}
	if docker.StrictCurrentUserRootlessPodman() && !docker.IsRootlessPodmanHost(cur) {
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
