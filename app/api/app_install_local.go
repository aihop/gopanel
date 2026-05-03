package api

import (
	"context"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

func AppLocalList(c fiber.Ctx) error {
	list, err := service.NewLocalAppService().List()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(list))
}
func AppLocalGet(c fiber.Ctx) error {
	key := c.Params("key")
	res, err := service.NewLocalAppService().Get(key)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}
func AppLocalInstall(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppLocalInstallCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	res, err := service.NewLocalAppService().Install(context.Background(), *req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}
