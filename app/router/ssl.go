package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func SSLRouter(r fiber.Router) {
	sslGroup := r.Group("/ssl")
	sslGroup.Use(middleware.JWT(constant.UserRoleAdmin))
	{
		sslGroup.Post("", api.SSLCreate)
		sslGroup.Post("/search", api.SSLSearch)
		sslGroup.Post("/list", api.SSLList)
		sslGroup.Post("/count", api.SSLCount)
		sslGroup.Post("/del", api.SSLDelete)
		sslGroup.Post("/update", api.SSLUpdate)
		sslGroup.Post("/apply", api.SSLApply)
		sslGroup.Post("/obtain", api.SSLObtain)
		sslGroup.Post("/push-cdn", api.SSLPushCDN)
		sslGroup.Post("/renew", api.SSLRenew)
		sslGroup.Get("/:id/logs", api.SSLLogsStream)
		sslGroup.Get("/:id", api.SSLGet)
		sslGroup.Get("/website/:websiteId", api.SSLGetByWebsite)

		pushRule := sslGroup.Group("/push-rule")
		pushRule.Post("/search", api.SSLPushRuleSearch)
		pushRule.Post("/count", api.SSLPushRuleCount)
		pushRule.Post("/", api.SSLPushRuleCreate)
		pushRule.Post("/update", api.SSLPushRuleUpdate)
		pushRule.Post("/del", api.SSLPushRuleDelete)
	}
}
