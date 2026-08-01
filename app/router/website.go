package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func WebsiteRouter(r fiber.Router) {
	websiteGroup := r.Group("/website")
	websiteGroup.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		websiteGroup.Post("/list", api.WebsiteList)
		websiteGroup.Post("/create", api.WebsiteCreate)
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
