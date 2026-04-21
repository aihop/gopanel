package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func AppsRouter(r fiber.Router) {
	SecurityRouter := r.Group("security")
	SecurityRouter.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		SecurityRouter.Get("/scan", api.SecurityScan)
		SecurityRouter.Post("/fix/ssh", api.SecurityFixSSH)
	}

	AppsRouter := r.Group("apps")
	AppsRouter.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		// 全部
		AppsRouter.Post("/list", api.AppsList)
		AppsRouter.Post("/sync", api.AppSync)

		// 安装
		AppsRouter.Post("/install", api.AppInstall)
		AppsRouter.Get("/precheck", api.PrecheckAppInstall)
		AppsRouter.Post("/repair/compose", api.RepairCompose)
		AppsRouter.Post("/repair/short-name", api.RepairPodmanShortName)
		AppsRouter.Post("/repair/port-conflict", api.RepairPortConflict)
		AppsRouter.Post("/local/install", api.AppLocalInstall)
		AppsRouter.Get("/local/list", api.AppLocalList)
		AppsRouter.Get("/local/:key", api.AppLocalGet)

		// 已安装的
		AppsRouter.Get("/installed/all", api.ListAppInstalled)
		AppsRouter.Post("/installed/list", api.SearchAppInstalled)
		AppsRouter.Post("/installed/op", api.OperateAppInstalled)
		AppsRouter.Post("/installed/sync", api.SyncAppInstalled)
		AppsRouter.Post("/installed/load-port", api.LoadAppInstalledPort)
		AppsRouter.Post("/installed/conn-info", api.GetAppInstalledConnInfo)
		AppsRouter.Post("/installed/check", api.CheckAppInstalled)
		AppsRouter.Get("/installed/delete/check/:id", api.AppInstalledDeleteCheck)
		AppsRouter.Get("/installed/params/:id", api.GetAppInstalledParams)
		AppsRouter.Post("/installed/params/update", api.UpdateAppInstalledParams)
		AppsRouter.Post("/installed/port/change", api.ChangeAppInstalledPort)
		AppsRouter.Post("/installed/conf", api.GetAppInstalledDefaultConfig)
		AppsRouter.Post("/installed/update/versions", api.UpdateAppInstalledVersions)
		AppsRouter.Post("/installed/ignore", api.IgnoreAppInstalledUpgrade)
		AppsRouter.Get("/ignored/detail", api.GetIgnoredAppDetail)

		// 卸载
		AppsRouter.Post("/uninstall", api.UninstallApp)

		// 基础安装目录
		AppsRouter.Get("/base-dir", api.AppGetBaseDir)

		AppsRouter.Get("/detail/:id", api.AppDetailGet)
		AppsRouter.Get("/:key", api.AppsGet)
		AppsRouter.Get("/install/:name/logs", api.AppInstallLogsStream)
	}
}
