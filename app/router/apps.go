package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func AppsRouter(r fiber.Router) {
	AppsRouter := r.Group("apps")
	AppsRouter.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		// 全部
		AppsRouter.Post("/search", api.AppsSearch)
		AppsRouter.Post("/sync", api.AppSync)

		// 安装
		AppsRouter.Post("/install", api.AppInstall)
		AppsRouter.Post("/local/install", api.AppLocalInstall)
		AppsRouter.Get("/local/list", api.AppLocalList)
		AppsRouter.Get("/local/:key", api.AppLocalGet)

		// 已安装的
		AppsRouter.Get("/installed/list", api.ListAppInstalled)
		AppsRouter.Post("/installed/search", api.SearchAppInstalled)
		AppsRouter.Post("/installed/op", api.OperateAppInstalled)
		AppsRouter.Post("/installed/sync", api.SyncAppInstalled)
		AppsRouter.Post("/installed/loadport", api.LoadAppInstalledPort)
		AppsRouter.Post("/installed/conninfo", api.GetAppInstalledConnInfo)
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
		AppsRouter.Get("/baseDir", api.AppGetBaseDir)

		AppsRouter.Get("/detail/:id", api.AppDetailGet)
		AppsRouter.Get("/:key", api.AppGet)
		AppsRouter.Get("/install/:name/logs", api.AppInstallLogsStream)
	}
}
