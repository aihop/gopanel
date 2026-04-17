package api

import (
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/gofiber/fiber/v3"
)

func WebsiteList(c fiber.Ctx) error {
	R, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewWebsite().List(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func WebsiteCount(c fiber.Ctx) error {
	R, err := e.BodyToWhere(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewWebsite().CountByWhere(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func WebsiteCreate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.WebsiteCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewWebsite().Create(R, model.DatabaseModeRemote); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func WebsiteUpdate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.WebsiteUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewWebsite().Update(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func WebsiteDelete(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewWebsite().Delete(R.ID); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}
