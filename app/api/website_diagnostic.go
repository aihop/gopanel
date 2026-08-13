package api

import (
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/convertor"
	"github.com/gofiber/fiber/v3"
)

func GetWebsiteDiagnosticSetting(c fiber.Ctx) error {
	websiteID := convertor.ToUint(c.Params("id"))
	if websiteID == 0 {
		return c.JSON(e.Fail(buserr.New("ErrWebsiteDiagnosticInvalidWebsiteID")))
	}
	setting, err := service.NewWebsite().GetDiagnosticSetting(websiteID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(setting))
}

func SaveWebsiteDiagnosticSetting(c fiber.Ctx) error {
	websiteID := convertor.ToUint(c.Params("id"))
	if websiteID == 0 {
		return c.JSON(e.Fail(buserr.New("ErrWebsiteDiagnosticInvalidWebsiteID")))
	}
	var input request.WebsiteDiagnosticSetting
	if err := c.Bind().JSON(&input); err != nil {
		return c.JSON(e.Fail(err))
	}
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	setting := model.WebsiteDiagnosticSetting{
		WebsiteID: websiteID, CodeProjectID: input.CodeProjectID, Enabled: input.Enabled,
		CaddyMonitoring: input.CaddyMonitoring, ActiveProbes: input.ActiveProbes,
		BackendHook: input.BackendHook, BrowserHook: input.BrowserHook, AutoAnalysis: input.AutoAnalysis,
		MonitorHTTP4xx: input.MonitorHTTP4xx, MonitorHTTP5xx: input.MonitorHTTP5xx,
		MonitorUpstreamErrors: input.MonitorUpstreamErrors, MonitorSlowRequests: input.MonitorSlowRequests,
		MonitorBusinessErrors: input.MonitorBusinessErrors, MonitorBrowserErrors: input.MonitorBrowserErrors,
		MonitorResourceErrors: input.MonitorResourceErrors, SlowRequestThresholdMS: input.SlowRequestThresholdMS,
		TriggerCount: input.TriggerCount, TriggerWindowMinutes: input.TriggerWindowMinutes,
		RetentionDays: input.RetentionDays, DefaultExecutorID: input.DefaultExecutorID, ApprovalPolicy: input.ApprovalPolicy,
	}
	view, err := service.NewWebsite().SaveDiagnosticSetting(&setting, claims.UserId, claims.Role == constant.UserRoleSuper)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(view))
}
