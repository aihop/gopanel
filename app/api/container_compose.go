package api

import (
	"errors"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func ContainerComposeList(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SearchWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	total, list, err := containerService.PageCompose(req)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(dto.PageResult{Items: list, Total: total}))
}
func CreateCompose(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ComposeCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	log, err := containerService.CreateCompose(req)
	if err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(log))
}
func TestCompose(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ComposeCreate](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	isOK, err := containerService.TestCompose(req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(isOK))
}
func OperatorCompose(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ComposeOperation](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	if err := containerService.ComposeOperation(req); err != nil {
		return c.JSON(e.Error(buserr.Err(err)))
	}
	if req.Operation == "delete" {
		appInstallService := service.NewAppInstall()
		appInstall := appInstallService.GetByName(req.Name)
		if appInstall != nil {
			appInstallService.Delete(appInstall.ID)
		}
	}
	return c.JSON(e.Succ())
}
func ComposeUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ComposeUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	if err := containerService.ComposeUpdate(req); err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}
func ComposeLogs(c *websocket.Conn) {
	defer c.Close()
	compose := c.Query("compose")
	since := c.Query("since")
	follow := c.Query("follow") == "true"
	tail := c.Query("tail")
	if err := containerService.ContainerLogs(c, "compose", compose, since, tail, "", follow); err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
}
