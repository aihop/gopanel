package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/gpagent"
	"github.com/gofiber/fiber/v3"
)

func HttpDefaultList(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cf, err := service.GetCaddyFile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if cf == "" {
		return c.JSON(e.Succ())
	}
	return c.JSON(e.Succ(cf))
}

func HttpDefaultGet(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.CaddyReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cf, err := service.GetCaddyFile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if cf == "" {
		return c.JSON(e.Succ())
	}
	adapter, err := service.CaddyGetDomainsConfigAsString(cf, req.PrimaryDomain, req.OtherDomains)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(adapter))
}

func HttpDefaultDelete(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.CaddyReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := service.CaddyRemoveServerBlock(ctx, req.PrimaryDomain, req.OtherDomains)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}

func HttpDefaultCheck(c fiber.Ctx) error {
	type CheckUrlReq struct {
		Domain string `json:"domain"`
	}
	req, err := e.BodyToStruct[CheckUrlReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if strings.TrimSpace(req.Domain) == "" {
		return c.JSON(e.Fail(errors.New("domain cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 检查url是否已经存在
	exist, err := service.CaddyExistDomain(ctx, req.Domain)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"exist": exist,
	}))
}

func HttpDefaultRead(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cf, err := service.GetCaddyFile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(cf))
}

func HttpDefaultUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.CaddyReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Content == "" {
		return c.JSON(e.Fail(fmt.Errorf("content cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if req.PrimaryDomain != "" {
		// 检查域名是否已经存在
		exist, err := service.CaddyExistDomain(ctx, req.PrimaryDomain)
		if err != nil {
			return c.JSON(e.Fail(err))
		}
		if !exist {
			return c.JSON(e.Fail(fmt.Errorf("domain %s does not exist in the configuration", req.PrimaryDomain)))
		}
		content, err := service.CaddyContent()
		if err != nil {
			return c.JSON(e.Fail(err))
		}
		adapter, err := service.CaddyUpdateReplace(content, req.Content, req.PrimaryDomain, req.OtherDomains)
		if err != nil {
			return c.JSON(e.Fail(err))
		}
		req.Content = adapter
	}
	if err := service.CaddyApplyCaddyFile(ctx, req.Content); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"updated": true}))
}

func HttpDefaultRestart(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cf, err := service.GetCaddyFile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.CaddyApplyCaddyFile(ctx, cf); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func HttpDefaultStop(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := gpagent.Do(ctx, "CADDY_STOP", nil); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func HttpDefaultStatus(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "CADDY_STATUS", nil)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	var out any
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(out))
}

func HttpDefaultResolve(c fiber.Ctx) error {
	type ResolveReq struct {
		Domain       string `json:"domain"`
		Proxy        string `json:"proxy"`
		OtherDomains string `json:"otherDomains"`
	}
	req, err := e.BodyToStruct[ResolveReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	protocol := constant.ProtocolHTTPS
	if strings.HasPrefix(req.Domain, "http://") {
		protocol = constant.ProtocolHTTP
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	added, err := service.CaddyAddServerBlock(ctx, req.Domain, req.Proxy, req.OtherDomains, protocol)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"added": added,
	}))
}
