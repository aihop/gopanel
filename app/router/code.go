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
		group.Get("/attention", api.GetCodeAttention)
		group.Get("/git/credentials", api.GetCodeGitCredentials)
		group.Post("/git/credentials", api.SaveCodeGitCredential)
		group.Put("/git/credentials/:id", api.SaveCodeGitCredential)
		group.Delete("/git/credentials/:id", api.DeleteCodeGitCredential)

		// Projects APIs
		group.Get("/projects", api.GetAIProjects)
		group.Post("/projects/repositories/discover", api.DiscoverCodeProjectRepositories)
		group.Post("/projects/quality-checks/preflight", api.PreflightCodeProjectQualityChecks)
		group.Post("/projects", api.CreateAIProject)
		group.Put("/projects/:id", api.UpdateAIProject)
		group.Get("/projects/:id/overview", api.GetCodeProjectOverview)
		group.Get("/projects/:id/git/branches", api.GetCodeProjectBranches)
		group.Delete("/projects/:id/git/branches", api.DeleteCodeProjectBranch)
		group.Get("/projects/:id/git/sync", api.GetCodeProjectSync)
		group.Post("/projects/:id/git/sync", api.SyncCodeProject)
		group.Get("/projects/:id/worktree-capability", api.GetCodeWorktreeCapability)
		group.Post("/projects/:id/terminal", api.OpenCodeProjectTerminal)
		group.Get("/project-terminal/:id/ws", middleware.HostTerminalSameOrigin, websocket.New(api.HostTerminalWebSocket))
		group.Get("/projects/:id/database-accesses", api.GetCodeDatabaseAccesses)
		group.Put("/projects/:id/database-accesses", api.SaveCodeDatabaseAccess)
		group.Delete("/projects/:id/database-accesses/:accessId", api.DeleteCodeDatabaseAccess)

		// Dev Sessions APIs
		group.Get("/sessions", api.GetAISessions)
		group.Get("/sessions/:id", api.GetAISession)
		group.Get("/sessions/:id/history", api.GetCodeSessionHistory)
		group.Get("/runs/:id", api.GetCodeExecutionRun)
		group.Get("/sessions/:id/previews", api.GetAISessionPreviews)
		group.Post("/sessions", api.CreateAISession)
		group.Get("/sessions/:id/initialization", api.GetCodeSessionInitialization)
		group.Post("/sessions/:id/initialization/retry", api.RetryCodeSessionInitialization)
		group.Put("/sessions/:id/title", api.UpdateCodeSessionTitle)
		group.Put("/sessions/:id/approval-policy", api.UpdateCodeSessionApprovalPolicy)
		group.Post("/sessions/:id/instructions", api.CreateAISessionInstruction)
		group.Get("/sessions/:id/state", api.GetAISessionState)
		group.Get("/sessions/:id/structure", api.GetAISessionStructure)
		group.Get("/sessions/:id/file", api.GetAISessionFile)
		group.Put("/sessions/:id/file", api.SaveAISessionFile)
		group.Get("/sessions/:id/codex-runtime", api.GetCodexRuntimeState)
		group.Get("/sessions/:id/token-usage", api.GetCodeTokenUsage)
		group.Get("/sessions/:id/audit-events", api.GetCodeAuditEvents)
		group.Get("/sessions/:id/git/status", api.GetCodeGitStatus)
		group.Post("/sessions/:id/git/sync/check", api.CheckCodeSessionGitSync)
		group.Post("/sessions/:id/git/sync", api.SyncCodeSessionGitRepository)
		group.Get("/sessions/:id/git/diff", api.GetCodeGitDiff)
		group.Get("/sessions/:id/git/history", api.GetCodeGitHistory)
		group.Get("/sessions/:id/git/history/diff", api.GetCodeGitHistoryDiff)
		group.Put("/sessions/:id/git/stage", api.UpdateCodeGitStage)
		group.Post("/sessions/:id/git/commit", api.CommitCodeGitChanges)
		group.Post("/sessions/:id/git/save", api.SaveCodeGitChanges)
		group.Post("/sessions/:id/worktree/merge", api.MergeCodeSessionWorktree)
		group.Get("/sessions/:id/delivery", api.GetCodeDeliveryJob)
		group.Get("/sessions/:id/delivery/conflicts", api.GetCodeDeliveryConflicts)
		group.Get("/sessions/:id/delivery/conflicts/file", api.GetCodeDeliveryConflictFile)
		group.Put("/sessions/:id/delivery/conflicts/file", api.SaveCodeDeliveryConflictFile)
		group.Post("/sessions/:id/delivery/conflicts/complete", api.CompleteCodeDeliveryConflicts)
		group.Post("/sessions/:id/delivery/conflicts/manual-confirm", api.ConfirmManualCodeDeliveryConflict)
		group.Get("/sessions/:id/delivery/push", api.GetCodeDeliveryPushStatus)
		group.Post("/sessions/:id/delivery/push", api.PushCodeSessionDelivery)
		group.Post("/sessions/:id/database-query", api.ExecuteCodeDatabaseQuery)
		group.Get("/sessions/:id/quality-checks", api.GetCodeQualityChecks)
		group.Post("/sessions/:id/quality-checks/run", api.RunCodeQualityCheck)
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
