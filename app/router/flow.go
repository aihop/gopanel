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
	group.Put("/:id", api.FlowUpdate)
	group.Delete("/:id", api.FlowDelete)
	group.Get("/runs", api.FlowRunPage)
	group.Get("/runs/:id", api.FlowRunGet)
	group.Post("/runs/:id/resume", api.FlowRunResume)
	group.Post("/runs/:id/rebuild", api.FlowRunRebuild)
	group.Get("/:id/code-deliveries", api.FlowCodeDeliverySources)
	group.Get("/:id/code-baseline", api.FlowCodeBaselineSource)
	group.Post("/runs", api.FlowRunCreate)
}
