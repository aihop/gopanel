package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func AgentRouter(r fiber.Router) {
	agentRouter := r.Group("agent")
	agentRouter.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		agentRouter.Get("/status", api.AgentStatus)
		agentRouter.Post("/ensure", api.AgentEnsure)
		agentRouter.Get("/ensure/logs", api.AgentEnsureLogs)
	}
}

