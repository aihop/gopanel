package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func CodeRouter(r fiber.Router) {
	group := r.Group("code")
	// Code 工作区使用 SUB_ADMIN 权限，确保普通管理员仍受目录沙箱限制。
	group.Use(middleware.JWT(constant.UserRoleSubAdmin))
	{
		// WebSocket 端点
		group.Get("/terminal", websocket.New(api.AIAgentWsSSH))
		group.Get("/executors", api.GetCodeExecutors)

		// Groups APIs
		group.Get("/groups", api.GetAIGroups)
		group.Post("/groups", api.CreateAIGroup)
		group.Put("/groups/:id", api.UpdateAIGroup)
		group.Get("/groups/:id/worktree-capability", api.GetCodeWorktreeCapability)

		// Dev Sessions APIs
		group.Get("/sessions", api.GetAISessions)
		group.Get("/sessions/:id", api.GetAISession)
		group.Get("/sessions/:id/history", api.GetCodeSessionHistory)
		group.Get("/runs/:id", api.GetCodeExecutionRun)
		group.Get("/sessions/:id/previews", api.GetAISessionPreviews)
		group.Post("/sessions", api.CreateAISession)
		group.Post("/sessions/:id/instructions", api.CreateAISessionInstruction)
		group.Get("/sessions/:id/state", api.GetAISessionState)
		group.Get("/sessions/:id/codex-runtime", api.GetCodexRuntimeState)
		group.Post("/sessions/:id/stop", api.StopCodeSessionExecution)
		group.Post("/instructions/:id/retry", api.RetryCodeInstruction)
		group.Get("/approvals", api.GetAIApprovals)
		group.Post("/approvals/:id/approve", api.ApproveAIApproval)
		group.Post("/approvals/:id/reject", api.RejectAIApproval)

		// Tasks APIs
		group.Get("/tasks", api.GetAITasks)
		group.Get("/tasks/:id/messages", api.GetAITaskMessages)
		group.Put("/tasks/:id", api.UpdateAITask)
		group.Delete("/tasks/:id", api.DeleteAITask)
	}
}
