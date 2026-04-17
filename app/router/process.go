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

		processRouter.Get("/daemon/status", api.StatusSupervisord)
		processRouter.Post("/daemon/start", api.StartSupervisord)
		processRouter.Post("/daemon/reload", api.ReloadSupervisord)
		processRouter.Post("/daemon/stop", api.StopSupervisord)

		processRouter.Get("/daemon/process/list", api.DaemonListProcess)
		processRouter.Post("/daemon/process/start/:name", api.DaemonStartProcess)
		processRouter.Post("/daemon/process/stop/:name", api.DaemonStopProcess)
		processRouter.Post("/daemon/process/reload/:name", api.DaemonReloadProcess)
		// 平滑重启
		processRouter.Post("/daemon/process/graceful/:name", api.DaemonGracefulRestart)
		processRouter.Post("/daemon/process/log", api.DaemonProcessLog)
		processRouter.Post("/daemon/process/log/clean", api.DaemonProcessLogClean)
		processRouter.Post("/daemon/process/startBatch", api.DaemonStartBatchProcess)
		processRouter.Post("/daemon/process/stopBatch", api.DaemonStopBatchProcess)
		processRouter.Post("/daemon/process/reloadBatch", api.DaemonReloadBatchProcess)

		processRouter.Get("/daemon/config/file/load", api.DaemonConfigFileLoad)
		processRouter.Post("/daemon/config/file/update", api.DaemonConfigFileSave)
		processRouter.Get("/daemon/config/list", api.DaemonConfigList)
		processRouter.Post("/daemon/config/add", api.DaemonConfigAdd)
		processRouter.Post("/daemon/config/update", api.DaemonConfigUpdate)
		processRouter.Post("/daemon/config/delete", api.DaemonConfigDelete)
	}
}
