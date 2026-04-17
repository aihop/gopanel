package api

import (
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/gofiber/fiber/v3"
)

func DatabaseList(c fiber.Ctx) error {
	R, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewDatabase().List(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

// @Tags Database
// @Summary Create database
// @Accept json
// @Param request body request.DatabaseCreate true "request"
// @Success 200 {object} e.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /database/create [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"创建数据库 [name]","formatEN":"Create database [name]"}
func DatabaseCreate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewDatabase().Create(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

// @Tags Database
// @Summary Delete database
// @Accept json
// @Param request body request.DatabaseDelete true "request"
// @Success 200 {object} e.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /database/delete [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"删除数据库 [name]","formatEN":"Delete database [name]"}
func DatabaseDelete(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseDelete](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewDatabase().Delete(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

// @Tags Database
// @Summary Update database description
// @Accept json
// @Param request body request.DatabaseComment true "request"
// @Success 200 {object} e.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /database/comment [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新数据库 [name] 描述","formatEN":"Update database [name] description"}
func DatabaseComment(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseComment](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewDatabase().Comment(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}
