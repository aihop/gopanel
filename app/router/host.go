package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func HostRouter(r fiber.Router) {
	hostRouter := r.Group("host").
		Use(middleware.JWT(constant.UserRoleAdmin))
	{

		hostRouter.Get("/firewall/base", api.LoadFirewallBaseInfo)
		hostRouter.Post("/firewall/search", api.SearchFirewallRule)
		hostRouter.Post("/firewall/operate", api.OperateFirewall)
		hostRouter.Post("/firewall/port", api.OperatePortRule)
		hostRouter.Post("/firewall/forward", api.OperateForwardRule)
		hostRouter.Post("/firewall/ip", api.OperateIPRule)
		hostRouter.Post("/firewall/batch", api.BatchOperateRule)
		hostRouter.Post("/firewall/update/port", api.UpdatePortRule)
		hostRouter.Post("/firewall/update/addr", api.UpdateAddrRule)
		hostRouter.Post("/firewall/update/description", api.UpdateFirewallDescription)

		hostRouter.Post("/monitor/search", api.LoadMonitor)
		hostRouter.Post("/monitor/clean", api.CleanMonitor)
		hostRouter.Get("/monitor/netoptions", api.GetNetworkOptions)
		hostRouter.Get("/monitor/iooptions", api.GetIOOptions)

		hostRouter.Post("/maintenance/clear", api.ClearHostMaintenance)
		hostRouter.Post("/maintenance/relieve-cpu", api.RelieveCPU)

	}
}
