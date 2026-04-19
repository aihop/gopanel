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
	if !res.GPC.Reachable && res.GPC.SocketPath != "" && runtime.GOOS != "windows" {
		res.Notes = append(res.Notes, "gpc helper 未连接：需要 sudo 启动 gpc helper 并配置 --file-roots")
	}
	return c.JSON(e.Succ(res))
}
