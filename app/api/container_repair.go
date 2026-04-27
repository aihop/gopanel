package api

import (
	"context"
	"errors"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	udocker "github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/gofiber/fiber/v3"
)

type repairPodmanSocketReq struct {
	Group string `json:"group"`
}

func ContainerRepairPodmanSocket(c fiber.Ctx) error {
	if runtime.GOOS != "linux" {
		return c.JSON(e.Error(errors.New("unsupported platform")))
	}
	req, err := e.BodyToStruct[repairPodmanSocketReq](c.Body())
	if err != nil {
		return c.JSON(e.Error(err))
	}
	group := strings.TrimSpace(req.Group)
	if group == "" {
		group = files.GetGroup(os.Getgid())
	}
	group = strings.TrimSpace(group)
	if group == "" {
		return c.JSON(e.Error(errors.New("cannot determine current process group")))
	}

	resolved := udocker.ResolveRuntime(context.Background())
	rootless := udocker.IsRootlessPodmanHost(resolved.Host)
	payload := map[string]interface{}{
		"group": group,
	}
	if rootless {
		payload["rootless"] = true
		payload["uid"] = os.Getuid()
		if curUser, err := user.Current(); err == nil && curUser != nil {
			payload["username"] = strings.TrimSpace(curUser.Username)
		}
	}

	resp, err := gpc.Do(context.Background(), "PODMAN_SOCKET_REPAIR", payload)
	if err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "unknown action") {
			return c.JSON(e.Error(errors.New("gpc helper 版本过旧，缺少 PODMAN_SOCKET_REPAIR 动作；请更新服务器上的 gpc 并重启 gpc.service 后再试")))
		}
		return c.JSON(e.Error(err))
	}
	if err := service.EnsurePodmanAPIReady(); err != nil {
		return c.JSON(e.Error(errors.New(strings.TrimSpace(resp.Output) + "\n" + err.Error())))
	}
	return c.JSON(e.Succ(map[string]any{
		"group":    group,
		"rootless": rootless,
		"output":   strings.TrimSpace(resp.Output),
	}))
}

func ContainerRepairSystemdLinger(c fiber.Ctx) error {
	if runtime.GOOS != "linux" {
		return c.JSON(e.Error(errors.New("unsupported platform")))
	}
	uid := os.Getuid()
	if uid == 0 {
		return c.JSON(e.Succ(map[string]any{
			"uid":    uid,
			"output": "",
		}))
	}
	resp, err := gpc.Do(context.Background(), "SYSTEMD_ENABLE_LINGER", map[string]interface{}{
		"uid": uid,
	})
	if err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "unknown action") {
			return c.JSON(e.Error(errors.New("gpc helper 版本过旧，缺少 SYSTEMD_ENABLE_LINGER 动作；请更新服务器上的 gpc 并重启 gpc.service 后再试")))
		}
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(map[string]any{
		"uid":    uid,
		"output": strings.TrimSpace(resp.Output),
	}))
}
