package api

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

func ClearHostMaintenance(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.HostMemoryClearReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	stdout, needPrivilege, message, err := service.NewHostMaintenance().ClearMemoryCaches(req.Mode)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(dto.HostMemoryClearRes{
		Stdout:        stdout,
		NeedPrivilege: needPrivilege,
		Message:       message,
	}))
}

func RelieveCPU(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.HostCPURelieveReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	level, err := service.NewHostMaintenance().RelieveCPU(req.Level)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(dto.HostCPURelieveRes{
		Level:   level,
		Message: "已降低 GoPanel 进程优先级（不影响 HTTP 服务）",
	}))
}
