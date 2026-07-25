package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func NodeRouter(r fiber.Router) {
	// 被控侧只读摘要：由节点令牌签名鉴权，不走用户 JWT，必须在下面的 JWT 分组之外注册
	r.Get("node/summary", middleware.NodeReadOnlyAuth, api.NodeSummary)

	nodeRouter := r.Group("node").
		Use(middleware.JWT(constant.UserRoleAdmin))
	{
		nodeRouter.Get("/list", api.NodeList)
		nodeRouter.Post("/create", api.NodeCreate)
		nodeRouter.Post("/update", api.NodeUpdate)
		nodeRouter.Post("/del", api.NodeDelete)
		nodeRouter.Post("/probe/:id", api.NodeProbe)
		nodeRouter.Post("/probe", api.NodeProbeDraft)
		nodeRouter.Post("/refresh", api.NodeRefresh)

		nodeRouter.Get("/local/token", api.NodeLocalTokenStatus)
		nodeRouter.Post("/local/token/issue", api.NodeLocalTokenIssue)
		nodeRouter.Post("/local/token/revoke", api.NodeLocalTokenRevoke)
	}
}
