package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func BackupRouter(r fiber.Router) {
	backupGroup := r.Group("/backup")
	backupGroup.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		backupGroup.Post("/record/search", api.BackupRecordSearch)
		backupGroup.Post("/record/count", api.BackupRecordCount)
		backupGroup.Post("/record/size", api.BackupRecordSize)
		backupGroup.Post("/record/deletes", api.BackupRecordDeletes)
		backupGroup.Post("/record/download", api.BackupRecordDownload)

		backupGroup.Post("/handle", api.BackupHandle)
		backupGroup.Get("/logs", api.BackupLogsStream)
		backupGroup.Post("/recover", api.BackupRecover)
		backupGroup.Post("/recover/byUpload", api.BackupRecoverByUpload)

	}
}
