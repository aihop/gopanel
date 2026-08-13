package api

import (
	"strconv"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func SecurityMonitoringConfigGet(c fiber.Ctx) error {
	config, err := service.GetSecurityMonitoringConfig()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(config))
}

func SecurityMonitoringConfigSave(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	config, err := e.BodyToStruct[model.SecurityMonitoringConfig](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.SaveSecurityMonitoringConfig(config, claims.UserId); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(config))
}

func SecurityEventPage(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 64)
	total, events, err := repo.NewSecurityMonitoring().PageEvents(
		page, limit, c.Query("status"), c.Query("level"), c.Query("sourceType"), uint(sourceID),
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": events, "total": total}))
}

func SecurityEventAnalyze(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return c.JSON(e.Fail(fiber.ErrBadRequest))
	}
	if err := service.AnalyzeSecurityEvent(uint(id)); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func SecurityMonitoringEvaluate(c fiber.Ctx) error {
	service.EvaluateSecurityRisks()
	return c.JSON(e.Succ())
}

func SecurityEventSummary(c fiber.Ctx) error {
	events, err := repo.NewSecurityMonitoring().ActiveSummary(5)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": events, "total": len(events)}))
}
