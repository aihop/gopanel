package api

import (
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/gofiber/fiber/v3"
)

func SSLPushRuleSearch(c fiber.Ctx) error {
	R, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	srv := service.NewSSLPushRuleService()
	data, err := srv.Search(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func SSLPushRuleCount(c fiber.Ctx) error {
	R, err := e.BodyToWhere(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	srv := service.NewSSLPushRuleService()
	data, err := srv.Count(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func SSLPushRuleCreate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.SSLPushRuleCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewSSLPushRuleService().Create(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func SSLPushRuleUpdate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.SSLPushRuleUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewSSLPushRuleService().Update(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func SSLPushRuleDelete(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewSSLPushRuleService().Delete(R.ID); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}
