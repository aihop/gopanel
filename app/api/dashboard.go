package api

import (
	"errors"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/gofiber/fiber/v3"

	"github.com/aihop/gopanel/constant"
)

// @Tags Dashboard
// @Summary Load os info
// @Accept json
// @Success 200 {object} dto.OsInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /dashboard/base/os [get]
func LoadDashboardOsInfo(c fiber.Ctx) error {
	data, err := dashboardService.LoadOsInfo()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(data))
}

// @Tags Dashboard
// @Summary Load dashboard base info
// @Accept json
// @Param ioOption path string true "request"
// @Param netOption path string true "request"
// @Success 200 {object} dto.DashboardBase
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /dashboard/base/{ioOption}/{netOption} [get]
func LoadDashboardBaseInfo(c fiber.Ctx) error {
	ioOption := c.Params("ioOption")
	if ioOption == "" {
		return c.JSON(e.Fail(errors.New("error ioOption in path")))
	}
	netOption := c.Params("netOption")
	if netOption == "" {
		return c.JSON(e.Fail(errors.New("error netOption in path")))
	}
	data, err := dashboardService.LoadBaseInfo(ioOption, netOption)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(data))
}

// @Tags Dashboard
// @Summary Load dashboard current info
// @Accept json
// @Param request body dto.DashboardReq true "request"
// @Success 200 {object} dto.DashboardCurrent
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /dashboard/current [post]
func LoadDashboardCurrentInfo(c fiber.Ctx) error {
	var req *dto.DashboardReq
	var err error
	if len(c.Body()) > 0 {
		req, err = e.BodyToStruct[dto.DashboardReq](c.Body())
	} else {
		req, err = e.QueriesToStruct[dto.DashboardReq](c.Queries())
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Scope == "" {
		req.Scope = "basic"
	}
	data := dashboardService.LoadCurrentInfo(*req)
	return c.JSON(e.Succ(data))
}

// @Tags Dashboard
// @Summary System restart panel
// @Accept json
// @Param operation path string true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /dashboard/system/restart/{operation} [post]
func SystemRestart(c fiber.Ctx) error {
	operation := c.Params("operation")
	if operation == "" {
		return c.JSON(e.Fail(errors.New("operation is empty")))
	}

	if err := dashboardService.Restart(operation); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ())
}
