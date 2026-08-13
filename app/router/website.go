package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func WebsiteRouter(r fiber.Router) {
	r.Post("/website-diagnostics/:id/events", api.ReceiveWebsiteDiagnosticEvent)
	websiteGroup := r.Group("/website")
	websiteGroup.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		websiteGroup.Post("/list", api.WebsiteList)
		websiteGroup.Post("/create", api.WebsiteCreate)
		websiteGroup.Get("/:id/diagnostics/settings", api.GetWebsiteDiagnosticSetting)
		websiteGroup.Put("/:id/diagnostics/settings", api.SaveWebsiteDiagnosticSetting)
		websiteGroup.Post("/:id/diagnostics/hook-secret", api.RotateWebsiteDiagnosticHookSecret)
		websiteGroup.Get("/:id/diagnostics/issues", api.ListWebsiteDiagnosticIssues)
		websiteGroup.Get("/:id/diagnostics/issues/:issueId", api.GetWebsiteDiagnosticIssue)
		websiteGroup.Post("/:id/diagnostics/issues/:issueId/action", api.UpdateWebsiteDiagnosticIssue)
		websiteGroup.Post("/:id/diagnostics/issues/:issueId/code", api.HandoffWebsiteDiagnosticIssue)
		websiteGroup.Post("/:id/diagnostics/issues/:issueId/verify", api.VerifyWebsiteDiagnosticIssue)
		websiteGroup.Get("/:id/diagnostics/probes", api.ListWebsiteDiagnosticProbes)
		websiteGroup.Put("/:id/diagnostics/probes", api.SaveWebsiteDiagnosticProbes)
		websiteGroup.Post("/:id/diagnostics/probes/:probeId/run", api.RunWebsiteDiagnosticProbe)
		websiteGroup.Put("/:id", api.WebsiteUpdate)
		websiteGroup.Delete("/:id", api.WebsiteDelete)
		websiteGroup.Post("/log", api.WebsiteLog)
		websiteGroup.Post("/log/today-ip", api.WebsiteLogTodayIPStats)
		websiteGroup.Post("/app-deploy/list", api.AppDeployList)
		websiteGroup.Post("/app-deploy/switch", api.AppDeploySwitch)
		websiteGroup.Post("/app-deploy/delete", api.AppDeployDelete)
		websiteGroup.Post("/app-deploy/trigger", api.AppDeployTrigger)
		websiteGroup.Post("/app-deploy/snapshot", api.AppDeploySnapshot)
	}
}
