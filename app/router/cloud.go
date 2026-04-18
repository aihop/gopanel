package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func CloudRouter(r fiber.Router) {
	cloudGroup := r.Group("/cloud")
	cloudGroup.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		cloudGroup.Post("/account/search", api.CloudAccountSearch)
		cloudGroup.Post("/account", api.CloudAccountCreate)
		cloudGroup.Post("/account/update", api.CloudAccountUpdate)
		cloudGroup.Post("/account/del", api.CloudAccountDelete)
		cloudGroup.Get("/cdn/:id/domains", api.CloudCdnDomains)
	}
}
