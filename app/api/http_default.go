package api

import (
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/init/caddy"
	"github.com/aihop/gopanel/utils/files"
	"github.com/gofiber/fiber/v3"
)

type CaddyReq struct {
	PrimaryDomain string `json:"primaryDomain"`
	OtherDomains  string `json:"otherDomains"`
	Content       string `json:"content"`
}

func HttpDefaultList(c fiber.Ctx) error {
	fileUtil := files.NewFileOp()
	content, err := fileUtil.GetContent(caddy.CaddyFilePath())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	if len(content) == 0 || string(content) == "" {
		return c.JSON(e.Succ())
	}
	adapter, err := caddy.CaddyFileToJson(content)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(string(adapter)))
}

func HttpDefaultGet(c fiber.Ctx) error {
	req, err := e.BodyToStruct[CaddyReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	fileUtil := files.NewFileOp()
	content, err := fileUtil.GetContent(caddy.CaddyFilePath())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	if len(content) == 0 || string(content) == "" {
		return c.JSON(e.Succ())
	}
	adapter, err := service.NewCaddy().GetDomainsConfigAsString(string(content), req.PrimaryDomain, req.OtherDomains)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(adapter))
}

func HttpDefaultDelete(c fiber.Ctx) error {
	req, err := e.BodyToStruct[CaddyReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	res, err := service.NewCaddy().RemoveServerBlock(req.PrimaryDomain, req.OtherDomains)
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
	// 检查url是否已经存在
	exist, err := service.NewCaddy().ExistDomain(req.Domain)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"exist": exist,
	}))
}

func HttpDefaultRead(c fiber.Ctx) error {
	fileUtil := files.NewFileOp()
	content, err := fileUtil.GetContent(caddy.CaddyFilePath())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(string(content)))
}

func HttpDefaultUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[CaddyReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Content == "" {
		return c.JSON(e.Fail(fmt.Errorf("content cannot be empty")))
	}
	fileUtil := files.NewFileOp()
	if req.PrimaryDomain != "" {
		// 检查域名是否已经存在
		exist, err := service.NewCaddy().ExistDomain(req.PrimaryDomain)
		if err != nil {
			return c.JSON(e.Fail(err))
		}
		if !exist {
			return c.JSON(e.Fail(fmt.Errorf("domain %s does not exist in the configuration", req.PrimaryDomain)))
		}
		content, err := fileUtil.GetContent(caddy.CaddyFilePath())
		if err != nil {
			return c.JSON(e.Fail(err))
		}
		adapter, err := service.NewCaddy().UpdateReplace(string(content), req.Content, req.PrimaryDomain, req.OtherDomains)
		if err != nil {
			return c.JSON(e.Fail(err))
		}
		req.Content = adapter
	}
	err = fileUtil.SaveFile(caddy.CaddyFilePath(), req.Content, 0755)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err = service.NewCaddy().ReloadCaddy(); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func HttpDefaultRestart(c fiber.Ctx) error {
	err := service.NewCaddy().ReloadCaddy()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func HttpDefaultStop(c fiber.Ctx) error {
	err := service.NewCaddy().StopCaddy()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func HttpDefaultStatus(c fiber.Ctx) error {
	status := caddy.Server.Status
	return c.JSON(e.Succ(fiber.Map{"status": status}))
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
	added, err := service.NewCaddy().AddServerBlock(req.Domain, req.Proxy, req.OtherDomains, protocol)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"added": added,
	}))
}
