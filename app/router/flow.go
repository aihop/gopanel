package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func FlowRouter(r fiber.Router) {
	group := r.Group("flow", middleware.JWT(constant.UserRoleSubAdmin))
	group.Get("/list", api.FlowPage)
	group.Post("/", api.FlowCreate)
}
