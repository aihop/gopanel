package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func AIAgentRouter(r fiber.Router) {
	group := r.Group("ai")
	// 注意这里使用了 SUB_ADMIN 权限，因为 AI 助手也需要受到目录沙箱的限制
	group.Use(middleware.JWT(constant.UserRoleSubAdmin))
	{
		// WebSocket 端点
		group.Get("/terminal", websocket.New(api.AIAgentWsSSH))

		// Groups APIs
		group.Get("/groups", api.GetAIGroups)
		group.Post("/groups", api.CreateAIGroup)

		// Tasks APIs
		group.Get("/tasks", api.GetAITasks)
		group.Get("/tasks/:id/messages", api.GetAITaskMessages)
		group.Put("/tasks/:id", api.UpdateAITask)
		group.Delete("/tasks/:id", api.DeleteAITask)
	}
}
