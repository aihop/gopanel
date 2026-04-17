package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func UserRouter(r fiber.Router) {
	userRouter := r.Group("user")
	// 注意这里：我们将 userRouter 的中间件权限放宽为 SUB_ADMIN，
	// 因为 /info、/editPassword 等接口子账号也需要访问，
	// 而那些只有超级管理员才能用的管理接口，我们在下方套一层 Group 并设置 ADMIN 权限。
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
