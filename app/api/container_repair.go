package api

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/gofiber/fiber/v3"
)

type repairPodmanSocketReq struct {
	Group string `json:"group"`
}

func RepairPodmanSocket(c fiber.Ctx) error {
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

	resp, err := gpc.Do(context.Background(), "PODMAN_SOCKET_REPAIR", map[string]interface{}{
		"group": group,
	})
	if err != nil {
		return c.JSON(e.Error(err))
	}
	if err := service.EnsurePodmanAPIReady(); err != nil {
		return c.JSON(e.Error(errors.New(strings.TrimSpace(resp.Output) + "\n" + err.Error())))
	}
	return c.JSON(e.Succ(map[string]any{
		"group":  group,
		"output": strings.TrimSpace(resp.Output),
	}))
}

func RepairSystemdLinger(c fiber.Ctx) error {
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
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(map[string]any{
		"uid":    uid,
		"output": strings.TrimSpace(resp.Output),
	}))
}

