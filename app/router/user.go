package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func UserRouter(r fiber.Router) {
	userRouter := r.Group("user")

	userRouter.Use(middleware.JWT(constant.UserRoleSubAdmin))
	{
		userRouter.Post("/info", api.UserInfo)
		userRouter.Post("/reset", api.ResetAccount)
		userRouter.Post("/editPassword", api.ResetPassword)
		userRouter.Post("/editInfo", api.UserEditInfo)
		userRouter.Post("/token", api.UserToken)

		// SubAdmin 账号管理 (只有 SUPER 或 ADMIN 可以操作)
		adminOnlyGroup := userRouter.Group("", middleware.JWT(constant.UserRoleAdmin))
		adminOnlyGroup.Post("/create", api.CreateUser)
		adminOnlyGroup.Post("/update", api.UpdateUser)
		adminOnlyGroup.Post("/delete", api.DeleteUser)
		adminOnlyGroup.Post("/search", api.PageUser)
	}
}
