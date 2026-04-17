package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func DashboardRouter(r fiber.Router) {
	homeRouter := r.Group("dashboard").
		Use(middleware.JWT(constant.UserRoleAdmin))
	{
		homeRouter.Get("/base/os", api.LoadDashboardOsInfo)
		homeRouter.Get("/base/:ioOption/:netOption", api.LoadDashboardBaseInfo)
		homeRouter.Get("/current", api.LoadDashboardCurrentInfo)
		homeRouter.Post("/system/restart/:operation", api.SystemRestart)
	}
}
