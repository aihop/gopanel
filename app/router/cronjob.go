package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func CronjobRouter(r fiber.Router) {
	cronjobGroup := r.Group("/cronjob")
	cronjobGroup.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		cronjobGroup.Post("/list", api.CronjobList)
		cronjobGroup.Post("/create", api.CronjobCreate)
		cronjobGroup.Post("/update", api.CronjobUpdate)
		cronjobGroup.Post("/get", api.CronjobGet)
		cronjobGroup.Post("/delete", api.CronjobDelete)
		cronjobGroup.Post("/status", api.CronjobSetStatus)
		cronjobGroup.Post("/run", api.CronjobRun)
		cronjobGroup.Post("/record/list", api.CronjobRecordList)
		cronjobGroup.Post("/record/delete", api.CronjobRecordDelete)
	}
}
