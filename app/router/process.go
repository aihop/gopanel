package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func ProcessRouter(r fiber.Router) {
	processRouter := r.Group("process")
	processRouter.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		// 系统进程监控
		processRouter.Get("/ws", websocket.New(api.ProcessWs))
		processRouter.Post("/list", api.ListProcess)
		processRouter.Post("/stop", api.StopProcess)
		processRouter.Post("/checkPort", api.CheckProcessPort)
	}
}
