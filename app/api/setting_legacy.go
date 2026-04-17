package api

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

func defaultDockerSockPath() string {
	if runtime.GOOS == "windows" {
		return "npipe:////./pipe/docker_engine"
	}
	return "unix:///var/run/docker.sock"
}

func SettingSystemInfo(c fiber.Ctx) error {
	info, err := service.NewSetting().GetInfo()
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	info.SystemIP = service.GetOutboundIP()
	if strings.TrimSpace(info.DockerSockPath) == "" {
		info.DockerSockPath = defaultDockerSockPath()
	}

	return c.JSON(e.Succ(info))
}

func SettingSystemUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SettingUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return c.JSON(e.Fail(fmt.Errorf("key is required")))
	}

	if err := repo.NewISettingRepo().UpdateOrCreate(req.Key, req.Value); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ())
}
