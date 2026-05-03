package api

import (
	"context"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

func AppGetBaseDir(c fiber.Ctx) error {
	return c.JSON(e.Succ(global.CONF.System.BaseDir + "/docker/compose/"))
}
func AppsInstall(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppInstallCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	res, err := service.NewAppService().Install(context.Background(), *req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"installId": res.ID, "name": res.Name}))
}
