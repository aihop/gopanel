package api

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/utils/convertor"
	"github.com/gofiber/fiber/v3"
)

// ---- 主控侧：节点管理 ----

// NodeList 节点列表，附带最近一次采集的摘要与告警
func NodeList(c fiber.Ctx) error {
	list, err := service.NewNode().List()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(list))
}

// NodeCreate 新增节点
func NodeCreate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.NodeCreateReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewNode().Create(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

// NodeUpdate 更新节点
func NodeUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.NodeUpdateReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewNode().Update(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

// NodeDelete 删除节点
func NodeDelete(c fiber.Ctx) error {
	req, err := e.BodyToStruct[struct {
		ID uint `json:"id" validate:"required"`
	}](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewNode().Delete(req.ID); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

// NodeProbe 立即采集单个节点
func NodeProbe(c fiber.Ctx) error {
	id, _ := convertor.ToInt(c.Params("id"))
	res, err := service.NewNode().Probe(uint(id))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}

// NodeProbeDraft 保存前测试连接
func NodeProbeDraft(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.NodeCreateReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	res, err := service.NewNode().ProbeDraft(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}

// NodeRefresh 手动触发一轮全量采集
func NodeRefresh(c fiber.Ctx) error {
	service.CollectAllNodes()
	list, err := service.NewNode().List()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(list))
}

// ---- 被控侧：本机只读接入 ----

// NodeSummary 返回本机摘要。走节点令牌签名鉴权，不需要用户登录态。
func NodeSummary(c fiber.Ctx) error {
	return c.JSON(e.Succ(service.LocalNodeSummary()))
}

// NodeLocalTokenStatus 查询本机是否已开启只读接入
func NodeLocalTokenStatus(c fiber.Ctx) error {
	return c.JSON(e.Succ(fiber.Map{"enabled": service.LocalNodeTokenEnabled()}))
}

// NodeLocalTokenIssue 签发本机只读令牌，明文仅本次返回
func NodeLocalTokenIssue(c fiber.Ctx) error {
	token, err := service.IssueLocalNodeToken()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"accessToken": token}))
}

// NodeLocalTokenRevoke 关闭本机只读接入
func NodeLocalTokenRevoke(c fiber.Ctx) error {
	if err := service.RevokeLocalNodeToken(); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
