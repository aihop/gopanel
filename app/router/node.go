package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
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

		nodeRouter.Get("/local/control", api.NodeLocalControlStatus)
		nodeRouter.Post("/local/control/issue", api.NodeLocalControlIssue)
		nodeRouter.Post("/local/control/revoke", api.NodeLocalControlRevoke)
	}

	// 代理转发。单独一个前缀而不是挂在 node 分组下，避免 /node/:id/* 把
	// /node/list、/node/probe/:id 这些具体路由吃掉（Fiber 按注册顺序匹配，容易踩坑）
	proxyRouter := r.Group("node-proxy").
		Use(middleware.JWT(constant.UserRoleAdmin))
	{
		proxyRouter.All("/:id/*", api.NodeProxy)
	}

	// WebSocket 代理（远程终端、容器日志）。浏览器用 ?auth=<JWT> 通过主控鉴权，
	// NodeProxyWsQuery 在升级前算出转发给节点的查询串（剥掉 auth/token，其余透传）
	wsProxyRouter := r.Group("node-proxy-ws").
		Use(middleware.JWT(constant.UserRoleAdmin))
	{
		wsProxyRouter.Get("/:id/*", middleware.NodeProxyWsQuery, websocket.New(api.NodeProxyWs))
	}
}
