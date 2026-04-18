package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func HttpRouter(r fiber.Router) {
	caddyRouter := r.Group("/http").
		Use(middleware.LocalOrJWT(constant.UserRoleAdmin))
	{
		caddyRouter.Get("/default/config", api.HttpDefaultRead)
		caddyRouter.Post("/default/update", api.HttpDefaultUpdate)
		caddyRouter.Post("/default/get", api.HttpDefaultGet)
		caddyRouter.Get("/default/list", api.HttpDefaultList)
		caddyRouter.Post("/default/check", api.HttpDefaultCheck)
		caddyRouter.Post("/default/delete", api.HttpDefaultDelete)
		caddyRouter.Post("/default/reload", api.HttpDefaultRestart)
		caddyRouter.Post("/default/stop", api.HttpDefaultStop)
		caddyRouter.Get("/default/status", api.HttpDefaultStatus)
		caddyRouter.Post("/default/resolve", api.HttpDefaultResolve)
	}
}
