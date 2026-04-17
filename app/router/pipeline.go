package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func PipelineRouter(r fiber.Router) {
	// 放宽给 SUB_ADMIN（子账号/普通管理员），只能看列表、执行和看日志
	group := r.Group("pipeline")
	group.Use(middleware.JWT(constant.UserRoleSubAdmin))
	{
		group.Get("/", api.PipelinePage)
		group.Post("/run", api.PipelineRun)
		group.Post("/stop", api.PipelineStop)
		group.Get("/records", api.PipelineRecordPage)
		group.Get("/logs", api.PipelineLogs)

		// 严格限制给 ADMIN（超级管理员），才能配流水线（搭架子）
		adminOnlyGroup := group.Group("", middleware.JWT(constant.UserRoleAdmin))
		adminOnlyGroup.Post("/", api.PipelineCreate)
		adminOnlyGroup.Put("/", api.PipelineUpdate)
		adminOnlyGroup.Delete("/", api.PipelineDelete)
		adminOnlyGroup.Delete("/record", api.PipelineRecordDelete)
	}
}
