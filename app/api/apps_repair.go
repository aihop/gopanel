package api

import (
	"context"
	"errors"
	"runtime"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/gofiber/fiber/v3"
)

func RepairCompose(c fiber.Ctx) error {
	if runtime.GOOS != "linux" {
		return c.JSON(e.Error(errors.New("unsupported platform")))
	}
	resp, err := gpc.Do(context.Background(), "COMPOSE_INSTALL", map[string]interface{}{})
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unknown action") {
			return c.JSON(e.Error(errors.New("gpc helper 版本过旧，缺少 COMPOSE_INSTALL 动作；请更新服务器上的 gpc 并重启 gpc.service 后再试")))
		}
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(map[string]any{
		"output": strings.TrimSpace(resp.Output),
	}))
}

