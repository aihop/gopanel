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
		websiteGroup.Post("/create", api.WebsiteCreate)
		websiteGroup.Post("/update", api.WebsiteUpdate)
		websiteGroup.Post("/delete", api.WebsiteDelete)
		websiteGroup.Post("/list", api.WebsiteList)
		websiteGroup.Post("/count", api.WebsiteCount)

		// Deployments
		websiteGroup.Post("/deploy/list", api.WebsiteDeployList)
		websiteGroup.Post("/deploy/switch", api.WebsiteDeploySwitch)
		websiteGroup.Post("/deploy/delete", api.WebsiteDeployDelete)
		websiteGroup.Post("/deploy/trigger", api.WebsiteDeployTrigger)
		websiteGroup.Post("/deploy/snapshot", api.WebsiteDeploySnapshot)

	}
}
