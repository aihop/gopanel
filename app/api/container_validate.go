package api

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/utils/compose"
	udocker "github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/gofiber/fiber/v3"
)

type containerValidateResult struct {
	RuntimeKind    string `json:"runtimeKind"`
	RuntimeHost    string `json:"runtimeHost"`
	ConfiguredHost string `json:"configuredHost"`
	HostPinned     bool   `json:"hostPinned"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	Summary        struct {
		State            string   `json:"state"`
		RuntimeInstalled bool     `json:"runtimeInstalled"`
		InstallSupported bool     `json:"installSupported"`
		InstallOptions   []string `json:"installOptions"`
		Recommended      string   `json:"recommended,omitempty"`
	} `json:"summary"`
	Runtime struct {
		ServiceActive       bool `json:"serviceActive"`
		UserServiceActive   bool `json:"userServiceActive"`
		SystemServiceActive bool `json:"systemServiceActive"`
		ApiReady            bool `json:"apiReady"`
		Rootless            bool `json:"rootless"`
	} `json:"runtime"`
	CLI struct {
		Docker        bool `json:"docker"`
		Podman        bool `json:"podman"`
		DockerCompose bool `json:"dockerCompose"`
		PodmanCompose bool `json:"podmanCompose"`
	} `json:"cli"`
	Compose struct {
		OK     bool   `json:"ok"`
		Bin    string `json:"bin"`
		Prefix string `json:"prefix"`
		Error  string `json:"error,omitempty"`
	} `json:"compose"`
	GPC struct {
		SocketPath string `json:"socketPath"`
		Exists     bool   `json:"exists"`
		Reachable  bool   `json:"reachable"`
		Error      string `json:"error,omitempty"`
	} `json:"gpc"`
	Notes []string `json:"notes"`
}

func ContainerEngineValidate(c fiber.Ctx) error {
	res := containerValidateResult{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	resolved := udocker.ResolveRuntime(context.Background())
	res.RuntimeKind = string(resolved.Kind)
	res.RuntimeHost = resolved.Host
	res.ConfiguredHost = udocker.ConfiguredDockerSockPath()
	res.HostPinned = udocker.RuntimeHostPinned()
	_, err := exec.LookPath("docker")
	res.CLI.Docker = err == nil
	_, err = exec.LookPath("podman")
	res.CLI.Podman = err == nil
	_, err = exec.LookPath("docker-compose")
	res.CLI.DockerCompose = err == nil
	_, err = exec.LookPath("podman-compose")
	res.CLI.PodmanCompose = err == nil
	res.Summary.RuntimeInstalled = res.CLI.Docker || res.CLI.Podman
	res.Summary.InstallSupported = runtime.GOOS == "linux" || runtime.GOOS == "darwin"
	if !res.Summary.RuntimeInstalled && res.Summary.InstallSupported {
		res.Summary.InstallOptions = []string{"podman", "docker"}
		res.Summary.Recommended = "podman"
		if runtime.GOOS == "linux" && os.Geteuid() == 0 {
			res.Summary.Recommended = "docker"
		}
	}
	if res.RuntimeKind == "podman" && runtime.GOOS == "linux" {
		res.Runtime.Rootless = udocker.IsRootlessPodmanHost(resolved.Host)
		res.Runtime.SystemServiceActive = systemctlServiceActive(context.Background(), false, "podman.socket")
		if res.Runtime.Rootless {
			res.Runtime.UserServiceActive = systemctlServiceActive(context.Background(), true, "podman.socket")
			res.Runtime.ServiceActive = res.Runtime.UserServiceActive || res.Runtime.SystemServiceActive
		} else {
			res.Runtime.ServiceActive = res.Runtime.SystemServiceActive
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		defer cancel()
		res.Runtime.ApiReady = udocker.CanPingHost(pingCtx, resolved.Host)
	} else if res.RuntimeKind == "docker" {
		pingCtx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		defer cancel()
		res.Runtime.ApiReady = udocker.CanPingHost(pingCtx, resolved.Host)
		res.Runtime.ServiceActive = res.Runtime.ApiReady
	} else if res.RuntimeKind == "podman" && runtime.GOOS == "darwin" {
		res.Runtime.ServiceActive = res.CLI.Podman
		res.Runtime.ApiReady = commandSucceeds(1200*time.Millisecond, "podman", "info")
	}

	if bin, prefix, err := compose.ResolveCommand(); err != nil {
		res.Compose.OK = false
		res.Compose.Error = err.Error()
	} else {
		res.Compose.OK = true
		res.Compose.Bin = bin
		res.Compose.Prefix = strings.Join(prefix, " ")
	}

	socketPath := gpc.SocketPath()
	res.GPC.SocketPath = socketPath
	if socketPath != "" && runtime.GOOS != "windows" {
		if _, err := os.Stat(socketPath); err == nil {
			res.GPC.Exists = true
		}
		dialCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "unix", socketPath)
		if err == nil {
			res.GPC.Reachable = true
			_ = conn.Close()
		} else {
			res.GPC.Reachable = false
			res.GPC.Error = err.Error()
		}
	}
	res.Summary.State = "ready"
	if !res.Summary.RuntimeInstalled {
		res.Summary.State = "notInstalled"
	} else if !res.Runtime.ApiReady {
		res.Summary.State = "notReady"
	} else if !res.Compose.OK {
		res.Summary.State = "composeMissing"
	}

	if res.RuntimeKind == "podman" && runtime.GOOS == "darwin" {
		if !res.CLI.PodmanCompose && !res.CLI.DockerCompose {
			res.Notes = append(res.Notes, "podmanComposeMissing")
		}
	}
	if res.RuntimeKind == "podman" && runtime.GOOS == "linux" {
		if res.Runtime.ServiceActive && !res.Runtime.ApiReady {
			res.Notes = append(res.Notes, "podmanApiUnavailable")
			if res.Runtime.Rootless {
				res.Notes = append(res.Notes, "podmanRootlessSession")
			} else if os.Getuid() != 0 {
				res.Notes = append(res.Notes, "podmanSocketPermission")
			}
		}
	}
	return c.JSON(e.Succ(res))
}

func commandSucceeds(timeout time.Duration, name string, args ...string) bool {
	if _, err := exec.LookPath(name); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Run() == nil
}

func systemctlServiceActive(ctx context.Context, user bool, unit string) bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	c, cancel := context.WithTimeout(baseCtx, 1200*time.Millisecond)
	defer cancel()
	args := []string{"is-active", unit}
	if user {
		args = append([]string{"--user"}, args...)
	}
	cmd := exec.CommandContext(c, "systemctl", args...)
	if user {
		runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
		if runtimeDir == "" {
			runtimeDir = fmt.Sprintf("/run/user/%d", os.Geteuid())
		}
		cmd.Env = append(os.Environ(),
			"XDG_RUNTIME_DIR="+runtimeDir,
			"DBUS_SESSION_BUS_ADDRESS=unix:path="+filepath.Join(runtimeDir, "bus"),
		)
	}
	out, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)) == "active"
}
