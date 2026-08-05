package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func MobileRouter(r fiber.Router) {
	mobile := r.Group("mobile")
	mobile.Get("/health", api.MobileHealth)
	mobile.Post("/pair/exchange", api.ExchangeMobilePairing)
	mobile.Post("/login", api.LoginMobileDevice)

	management := mobile.Group("/management").Use(middleware.JWT(constant.UserRoleAdmin))
	management.Post("/pair/issue", api.IssueMobilePairing)
	management.Get("/devices", api.ListMobileDevices)
	management.Post("/devices/:id/revoke", api.RevokeMobileDevice)

	app := mobile.Group("/app").Use(middleware.MobileDeviceAuth)
	app.Get("/overview", api.GetMobileOverview)
	app.Get("/attention", api.GetCodeAttention)
	app.Get("/nodes", api.GetMobileNodes)
	app.Post("/logout", api.LogoutMobileDevice)
	app.Get("/system/version", api.SettingSystemVersion)
	app.Get("/system/check", api.SettingSystemCheck)
	app.Post("/system/upgrade", api.SettingSystemUpgrade)
	app.Get("/system/upgrade/logs", api.SettingSystemUpgradeLogs)
	app.Get("/containers", api.GetMobileContainers)
	app.Post("/containers/operate", api.OperateMobileContainer)
	app.Get("/containers/:id/publish-options", api.GetMobileContainerPublishOptions)
	app.Post("/containers/publish-website", api.PublishMobileContainerWebsite)
	app.Get("/resources/websites", api.GetMobileWebsites)
	app.Post("/resources/websites/domains", api.UpdateMobileWebsiteDomainBindings)
	app.Get("/resources/databases", api.GetMobileDatabases)
	app.Get("/resources/ssl", api.GetMobileSSLs)
	app.Get("/resources/apps", api.GetMobileApps)
	app.Get("/projects", api.GetAIProjects)
	app.Get("/projects/:id/git/sync", api.GetCodeProjectSync)
	app.Post("/projects/:id/git/sync", api.SyncCodeProject)
	app.Get("/projects/:id/worktree-capability", api.GetCodeWorktreeCapability)
	app.Post("/projects/:id/terminal", api.OpenCodeProjectTerminal)
	app.Get("/project-terminal/:id/ws", middleware.HostTerminalSameOrigin, websocket.New(api.HostTerminalWebSocket))
	app.Get("/executors", api.GetCodeExecutors)
	app.Get("/terminal", websocket.New(api.AIAgentWsSSH))
	app.Get("/sessions", api.GetAISessions)
	app.Post("/sessions", api.CreateAISession)
	app.Get("/sessions/:id/initialization", api.GetCodeSessionInitialization)
	app.Post("/sessions/:id/initialization/retry", api.RetryCodeSessionInitialization)
	app.Put("/sessions/:id/title", api.UpdateCodeSessionTitle)
	app.Get("/sessions/:id/state", api.GetAISessionState)
	app.Get("/sessions/:id/git/status", api.GetCodeGitStatus)
	app.Post("/sessions/:id/git/save", api.SaveCodeGitChanges)
	app.Get("/sessions/:id/delivery", api.GetCodeDeliveryJob)
	app.Post("/sessions/:id/worktree/merge", api.MergeCodeSessionWorktree)
	app.Get("/sessions/:id/structure", api.GetAISessionStructure)
	app.Get("/sessions/:id/file", api.GetAISessionFile)
	app.Put("/sessions/:id/file", api.SaveAISessionFile)
	app.Post("/sessions/:id/instructions", api.CreateAISessionInstruction)
	app.Post("/sessions/:id/stop", api.StopCodeSessionExecution)
	app.Post("/instructions/:id/retry", api.RetryCodeInstruction)
	app.Post("/approvals/:id/approve", api.ApproveAIApproval)
	app.Post("/approvals/:id/reject", api.RejectAIApproval)
}
