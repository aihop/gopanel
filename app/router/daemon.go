package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func DaemonRouter(r fiber.Router) {
	daemonRouter := r.Group("daemon")
	daemonRouter.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		daemonRouter.Get("/status", api.DaemonStatus)
		daemonRouter.Post("/start", api.DaemonStart)
		daemonRouter.Post("/reload", api.DaemonReload)
		daemonRouter.Post("/stop", api.DaemonStop)

		daemonRouter.Get("/process/list", api.DaemonListProcess)
		daemonRouter.Post("/process/start/:name", api.DaemonStartProcess)
		daemonRouter.Post("/process/stop/:name", api.DaemonStopProcess)
		daemonRouter.Post("/process/reload/:name", api.DaemonReloadProcess)
		daemonRouter.Post("/process/graceful/:name", api.DaemonGracefulRestart)
		daemonRouter.Post("/process/log", api.DaemonProcessLog)
		daemonRouter.Post("/process/log/clear", api.DaemonProcessLogClear)
		daemonRouter.Get("/config/file/load", api.DaemonConfigFileLoad)
		daemonRouter.Post("/config/file/update", api.DaemonConfigFileSave)
		daemonRouter.Post("/config/add", api.DaemonConfigAdd)
		daemonRouter.Post("/config/update", api.DaemonConfigUpdate)
		daemonRouter.Post("/config/delete", api.DaemonConfigDelete)
	}
}
