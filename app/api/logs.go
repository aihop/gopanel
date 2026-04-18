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

// @Tags Logs
// @Summary List system log files
// @Accept json
// @Success 200 {object} e.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /logs/system/files [get]
func GetSystemLogFiles(c fiber.Ctx) error {
	logService := service.NewLogService()
	files, err := logService.ListSystemLogFile()
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(files))
}

// @Tags Logs
// @Summary Read system log content
// @Accept json
// @Param request body dto.SearchSystemLog true "request"
// @Success 200 {object} e.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /logs/system [post]
func GetSystemLogs(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SearchSystemLog](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	logService := service.NewLogService()
	content, err := logService.ReadSystemLog(req.Name)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(content))
}

// @Tags Logs
// @Summary Page login logs
// @Accept json
// @Param request body dto.SearchLgLogWithPage true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /logs/login [post]
func GetLoginLogs(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SearchLgLogWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	logService := service.NewLogService()
	total, list, err := logService.PageLoginLog(*req)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}

	return c.JSON(e.Succ(fiber.Map{
		"items": list,
		"total": total,
	}))
}

// @Tags Logs
// @Summary Page operation logs
// @Accept json
// @Param request body dto.SearchOpLogWithPage true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /logs/operation [post]
func GetOperationLogs(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SearchOpLogWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	logService := service.NewLogService()
	total, list, err := logService.PageOperationLog(*req)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}

	return c.JSON(e.Succ(fiber.Map{
		"items": list,
		"total": total,
	}))
}

// @Tags Logs
// @Summary Clean logs
// @Accept json
// @Param request body dto.CleanLog true "request"
// @Success 200 {object} e.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /logs/clean [post]
func LogsClean(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.CleanLog](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	logService := service.NewLogService()
	if err := logService.CleanLogs(req.LogType); err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}
