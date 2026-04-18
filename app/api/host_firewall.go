package api

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

func LoadFirewallBaseInfo(c fiber.Ctx) error {
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Result(err))
	}
	data, err := svc.Base()
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(data))
}

func SearchFirewallRule(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.RuleSearch](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Result(err))
	}
	data, err := svc.Search(req)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(data))
}

func OperateFirewall(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.FirewallOperation](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err = svc.Operate(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func OperatePortRule(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.PortRuleOperate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err = svc.OperatePortRule(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func OperateForwardRule(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ForwardRuleOperate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err = svc.OperateForwardRule(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func OperateIPRule(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.AddrRuleOperate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err = svc.OperateIPRule(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func UpdatePortRule(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.PortRuleUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err = svc.UpdatePortRule(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func UpdateAddrRule(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.AddrRuleUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err = svc.UpdateAddrRule(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func UpdateFirewallDescription(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.UpdateFirewallDescription](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err = svc.UpdateDescription(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func BatchOperateRule(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.BatchRuleOperate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	svc, err := service.NewFirewall()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err = svc.BatchOperate(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
