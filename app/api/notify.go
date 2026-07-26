package api

import (
	"strconv"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

// notifyConfigReq 密码单独一个字段：留空表示不修改。
// 前端拿不到明文，只能留空提交，不这样处理的话用户改个端口就把密码清空了。
type notifyConfigReq struct {
	model.NotifyConfig
	Password string `json:"password"`
}

// NotifyConfigGet 读取通知配置（不含密码明文）
func NotifyConfigGet(c fiber.Ctx) error {
	return c.JSON(e.Succ(service.GetNotifyConfig()))
}

// NotifyConfigSave 保存通知配置
func NotifyConfigSave(c fiber.Ctx) error {
	req, err := e.BodyToStruct[notifyConfigReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.SaveNotifyConfig(req.NotifyConfig, req.Password); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(service.GetNotifyConfig()))
}

// NotifyTest 发一封测试邮件。SMTP 配错的方式太多（端口/加密/授权码/发件人不一致），
// 没有即时反馈用户根本不知道配没配对，直到出事故才发现通知没发出来。
func NotifyTest(c fiber.Ctx) error {
	req, err := e.BodyToStruct[notifyConfigReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.TestNotifyMail(req.NotifyConfig, req.Password); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ("测试邮件已发送，请查收"))
}

// NotifyEventPage 告警事件列表
func NotifyEventPage(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	total, list, err := repo.NewNotify().PageEvents(page, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"total": total, "items": list}))
}

// NotifyEvaluate 立即跑一轮告警评估，用于配置完之后马上验证效果，
// 不用等下一个采集周期
func NotifyEvaluate(c fiber.Ctx) error {
	service.EvaluateAlerts()
	return c.JSON(e.Succ("已执行一轮告警评估"))
}
