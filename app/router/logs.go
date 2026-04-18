package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func LogsRouter(r fiber.Router) {
	logsRouter := r.Group("logs", middleware.JWT(constant.UserRoleAdmin))
	{
		logsRouter.Post("/login", api.GetLoginLogs)
		logsRouter.Post("/operation", api.GetOperationLogs)
		logsRouter.Get("/system/files", api.GetSystemLogFiles)
		logsRouter.Post("/system", api.GetSystemLogs)
		logsRouter.Post("/clean", api.LogsClean)
	}
}
