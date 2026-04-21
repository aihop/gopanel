package api

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/utils/compose"
	udocker "github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/gofiber/fiber/v3"
)

type containerPrecheckResult struct {
	RuntimeKind string `json:"runtimeKind"`
	RuntimeHost string `json:"runtimeHost"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Runtime     struct {
		ServiceActive bool `json:"serviceActive"`
		ApiReady      bool `json:"apiReady"`
	} `json:"runtime"`
	CLI         struct {
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

func ContainerPrecheck(c fiber.Ctx) error {
	res := containerPrecheckResult{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	resolved := udocker.ResolveRuntime(context.Background())
	res.RuntimeKind = string(resolved.Kind)
	res.RuntimeHost = resolved.Host
	if res.RuntimeKind == "podman" && runtime.GOOS == "linux" {
		active := false
		if _, err := exec.LookPath("systemctl"); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
			defer cancel()
			cmd := exec.CommandContext(ctx, "systemctl", "is-active", "podman.socket")
			out, _ := cmd.CombinedOutput()
			active = strings.TrimSpace(string(out)) == "active"
		}
		res.Runtime.ServiceActive = active
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
		res.Runtime.ApiReady = false
	}

	_, err := exec.LookPath("docker")
	res.CLI.Docker = err == nil
	_, err = exec.LookPath("podman")
	res.CLI.Podman = err == nil
	_, err = exec.LookPath("docker-compose")
	res.CLI.DockerCompose = err == nil
	_, err = exec.LookPath("podman-compose")
	res.CLI.PodmanCompose = err == nil

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

	if res.RuntimeKind == "podman" && runtime.GOOS == "darwin" {
		if !res.CLI.PodmanCompose && !res.CLI.DockerCompose {
			res.Notes = append(res.Notes, "podman compose provider 未就绪：建议安装 podman-compose 或启用 podman compose provider")
		}
	}
	if res.RuntimeKind == "podman" && runtime.GOOS == "linux" {
		if res.Runtime.ServiceActive && !res.Runtime.ApiReady {
			uid := os.Getuid()
			res.Notes = append(res.Notes, "podman.socket 已启动但 Docker API 不可用：通常是 socket 权限/组不匹配或 rootless user session 未就绪")
			if uid != 0 {
				res.Notes = append(res.Notes, "可尝试：修复 podman.socket 权限（SocketGroup）或启用 linger（loginctl enable-linger）")
			}
		}
	}
	if !res.GPC.Reachable && res.GPC.SocketPath != "" && runtime.GOOS != "windows" {
		res.Notes = append(res.Notes, "gpc helper 未连接：需要 sudo 启动 gpc helper 并配置 --file-roots")
	}
	return c.JSON(e.Succ(res))
}
