package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func DatabaseRouter(r fiber.Router) {
	databaseRouter := r.Group("database")
	databaseRouter.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		databaseRouter.Post("/list", api.DatabaseList)
		databaseRouter.Post("/create", api.DatabaseCreate)
		databaseRouter.Post("/delete", api.DatabaseDelete)
		databaseRouter.Post("/comment", api.DatabaseComment)
		databaseRouter.Post("/server/list", api.DatabaseServerList)
		databaseRouter.Post("/server/count", api.DatabaseServerCount)
		databaseRouter.Post("/server/create", api.DatabaseServerCreate)
		databaseRouter.Post("/server/update", api.DatabaseServerUpdate)
		databaseRouter.Post("/server/delete", api.DatabaseServerDelete)
		databaseRouter.Post("/server/sync", api.DatabaseServerSync)
		databaseRouter.Post("/server/get", api.DatabaseServerGet)

		databaseRouter.Post("/user/list", api.DatabaseUserList)
		databaseRouter.Post("/user/count", api.DatabaseUserCount)
		databaseRouter.Post("/user/create", api.DatabaseUserCreate)
		databaseRouter.Post("/user/update", api.DatabaseUserUpdate)
		databaseRouter.Post("/user/delete", api.DatabaseUserDelete)
		databaseRouter.Post("/user/get", api.DatabaseUserGet)

		// 新增数据库管理器接口 (Manager)
		managerGroup := databaseRouter.Group("/manager")
		managerGroup.Post("/tables", api.GetDBManagerTables)
		managerGroup.Post("/data", api.GetDBManagerTableData)
		managerGroup.Post("/exec", api.ExecDBManagerSql)
		managerGroup.Post("/insert", api.InsertDBManagerRecord)
		managerGroup.Post("/update", api.UpdateDBManagerRecord)
		managerGroup.Post("/delete", api.DeleteDBManagerRecord)
	}
}
