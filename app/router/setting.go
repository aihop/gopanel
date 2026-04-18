package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func SettingRouter(r fiber.Router) {
	settingGroup := r.Group("/setting")
	settingGroup.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		settingGroup.Post("/system/info", api.SettingSystemInfo)
		settingGroup.Post("/system/update", api.SettingSystemUpdate)

		settingGroup.Get("/system/version", api.SettingSystemVersion)
		settingGroup.Get("/system/check", api.SettingSystemCheck)
		settingGroup.Post("/system/upgrade", api.SettingSystemUpgrade)
		settingGroup.Get("/system/upgrade/logs", api.SettingSystemUpgradeLogs)
		settingGroup.Post("/system/restart", api.SettingSystemRestart)
		settingGroup.Post("/system/restart/:operation", api.SettingSystemRestart)

		settingGroup.Post("/system/config", api.SettingSystemConfig)
		settingGroup.Post("/system/port", api.SettingSystemPort)

		settingGroup.Post("/system/entrance", api.SettingSystemEntrance)

		settingGroup.Post("/system/clear", api.SettingSystemClearDir)
		settingGroup.Post("/system/baseDir", api.SettingSystemBaseDir)
		settingGroup.Post("/system/apiToken", api.SettingSystemApiTokenUpdate)

	}
}
