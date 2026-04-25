package api

import (
	"context"
	"errors"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/aihop/gopanel/utils/convertor"
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
	total, _ := service.NewWebsite().CountByWhere(&gormx.Wherex{})
	return c.JSON(e.Succ(dto.PageResult{
		Items: data,
		Total: total,
	}))
}

func WebsiteCreate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.WebsiteCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = service.NewWebsite().Create(ctx, R, model.DatabaseModeRemote); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func WebsiteUpdate(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.JSON(e.Fail(errors.New("ID not found")))
	}
	R, err := e.BodyToStruct[request.WebsiteUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = service.NewWebsite().Update(ctx, R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func WebsiteDelete(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.JSON(e.Fail(errors.New("ID not found")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := service.NewWebsite().Delete(ctx,convertor.ToUint(id)); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func WebsiteLog(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.WebsiteLogRead](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	res, err := service.NewWebsite().ReadWebsiteLog(*req)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(res))
}

func WebsiteLogTodayIPStats(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.WebsiteLogTodayIPStats](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	res, err := service.NewWebsite().ReadWebsiteTodayIPStats(*req)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(res))
}
