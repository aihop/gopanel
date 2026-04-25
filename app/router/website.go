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

		websiteGroup.Post("/deploy/list", api.WebsiteDeployList)
		websiteGroup.Post("/deploy/switch", api.WebsiteDeploySwitch)
		websiteGroup.Post("/deploy/delete", api.WebsiteDeployDelete)
		websiteGroup.Post("/deploy/trigger", api.WebsiteDeployTrigger)
		websiteGroup.Post("/deploy/snapshot", api.WebsiteDeploySnapshot)
	}
}
