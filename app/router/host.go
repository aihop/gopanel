package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func HostRouter(r fiber.Router) {
	hostRouter := r.Group("host").
		Use(middleware.JWT(constant.UserRoleAdmin))
	{

		hostRouter.Get("/firewall/base", api.LoadFirewallBaseInfo)
		hostRouter.Post("/firewall/list", api.SearchFirewallRule)
		hostRouter.Post("/firewall/operate", api.OperateFirewall)
		hostRouter.Post("/firewall/port", api.OperatePortRule)
		hostRouter.Post("/firewall/forward", api.OperateForwardRule)
		hostRouter.Post("/firewall/ip", api.OperateIPRule)
		hostRouter.Post("/firewall/batch", api.BatchOperateRule)
		hostRouter.Post("/firewall/update/port", api.UpdatePortRule)
		hostRouter.Post("/firewall/update/addr", api.UpdateAddrRule)
		hostRouter.Post("/firewall/update/description", api.UpdateFirewallDescription)

		hostRouter.Post("/monitor/list", api.HostMonitorList)
		hostRouter.Post("/monitor/clean", api.CleanMonitor)
		hostRouter.Get("/monitor/netoptions", api.GetNetworkOptions)
		hostRouter.Get("/monitor/iooptions", api.GetIOOptions)

		hostRouter.Post("/maintenance/clear", api.ClearHostMaintenance)
		hostRouter.Post("/maintenance/relieve-cpu", api.RelieveCPU)

		hostRouter.Post("/terminal/sessions", api.CreateHostTerminalSession)
		hostRouter.Get("/terminal/capabilities", api.GetHostTerminalCapabilities)
		hostRouter.Get("/terminal/sessions", api.ListHostTerminalSessions)
		hostRouter.Post("/terminal/sessions/:id/stop", api.StopHostTerminalSession)
		hostRouter.Post("/terminal/sessions/:id/reconnect", api.ReconnectHostTerminalSession)
		hostRouter.Delete("/terminal/sessions/:id", api.DeleteHostTerminalSession)
		hostRouter.Get("/terminal/sessions/:id/audit-events", api.GetHostTerminalAuditEvents)
		hostRouter.Get("/terminal/sessions/:id/ws", middleware.HostTerminalSameOrigin, websocket.New(api.HostTerminalWebSocket))

		// 磁盘管理：扫描大文件 + 清理。删除是不可逆操作，保持 ADMIN 权限，
		// 不像文件管理那样放宽到 SUB_ADMIN。
		hostRouter.Get("/disk/overview", api.HostDiskOverview)
		hostRouter.Get("/disk/gpc", api.HostDiskGpcStatus)
		hostRouter.Post("/disk/scan", api.HostDiskScanStart)
		hostRouter.Get("/disk/scan/result", api.HostDiskScanResult)
		hostRouter.Get("/disk/scan/stream", api.HostDiskScanStream)
		hostRouter.Post("/disk/scan/cancel", api.HostDiskScanCancel)
		hostRouter.Post("/disk/clean", api.HostDiskClean)

	}
}
