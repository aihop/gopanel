package api

import (
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/gofiber/fiber/v3"
)

func CloudAccountSearch(c fiber.Ctx) error {
	ctx, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	res, err := service.NewCloudAccount().Page(ctx)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(res))
}

func CloudAccountCreate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CloudAccountCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewCloudAccount().Create(req); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func CloudAccountUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CloudAccountUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewCloudAccount().Update(req); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func CloudAccountDelete(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewCloudAccount().Delete(req.ID); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}
