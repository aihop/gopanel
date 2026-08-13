package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func SecurityMonitoringRouter(r fiber.Router) {
	group := r.Group("/security-monitoring", middleware.JWT(constant.UserRoleAdmin))
	group.Get("/config", api.SecurityMonitoringConfigGet)
	group.Put("/config", api.SecurityMonitoringConfigSave)
	group.Get("/events", api.SecurityEventPage)
	group.Post("/events/:id/analyze", api.SecurityEventAnalyze)
	group.Post("/evaluate", api.SecurityMonitoringEvaluate)
}
