package api

import (
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

func FileRecycleStatus(c fiber.Ctx) error {
	serviceService := service.NewSetting()
	settingInfo, err := serviceService.GetInfo()
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(settingInfo.FileRecycleBin))
}
