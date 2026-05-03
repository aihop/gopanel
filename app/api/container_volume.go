package api

import (
	"errors"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func ContainerVolumeList(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.SearchWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	total, list, err := service.NewIContainerService().PageVolume(R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(dto.PageResult{Items: list, Total: total}))
}
func ListVolume(c fiber.Ctx) error {
	list, err := service.NewIContainerService().ListVolume()
	if err != nil {
		return c.JSON(e.Result(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(list))
}
func DeleteVolume(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.BatchDelete](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().DeleteVolume(R); err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}
func CreateVolume(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.VolumeCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().CreateVolume(R); err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}
