package api

import (
	"fmt"
	"strconv"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

func WebsiteReleasePage(c fiber.Ctx) error {
	websiteID, _ := strconv.Atoi(c.Query("websiteId"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if websiteID <= 0 {
		return c.JSON(e.Fail(fmt.Errorf("无效的网站 id")))
	}

	appSvc := service.NewAppDeployApplication(global.DB)
	total, list, err := appSvc.PageWebsiteReleases(uint(websiteID), page, limit)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(fiber.Map{
		"total": total,
		"items": list,
	}))
}

type AppDeployListReq struct {
	WebsiteID uint `json:"websiteId" validate:"required"`
}

type AppDeploySwitchReq struct {
	DeployID uint `json:"deployId" validate:"required"`
}

type AppDeployDeleteReq struct {
	DeployID uint `json:"deployId" validate:"required"`
}

func AppDeployList(c fiber.Ctx) error {
	R, err := e.BodyToStruct[AppDeployListReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	appSvc := service.NewAppDeployApplication(global.DB)
	list, err := appSvc.ListByWebsite(R.WebsiteID)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(list))
}

func AppDeploySwitch(c fiber.Ctx) error {
	R, err := e.BodyToStruct[AppDeploySwitchReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	appSvc := service.NewAppDeployApplication(global.DB)
	if err := appSvc.Switch(R.DeployID); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func AppDeployDelete(c fiber.Ctx) error {
	R, err := e.BodyToStruct[AppDeployDeleteReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	appSvc := service.NewAppDeployApplication(global.DB)
	if err := appSvc.Delete(R.DeployID); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

type AppDeployTriggerReq struct {
	WebsiteID uint   `json:"websiteId" validate:"required"`
	ZipPath   string `json:"zipPath"`
	ImageTag  string `json:"imageTag"` // 新增字段，用于触发自定义镜像部署
	ReleaseID uint   `json:"releaseId"`
}

func AppDeployTrigger(c fiber.Ctx) error {
	R, err := e.BodyToStruct[AppDeployTriggerReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	appSvc := service.NewAppDeployApplication(global.DB)
	if err := appSvc.Trigger(service.AppDeployTriggerOptions{
		WebsiteID: R.WebsiteID,
		ZipPath:   R.ZipPath,
		ImageTag:  R.ImageTag,
		ReleaseID: R.ReleaseID,
	}); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

type AppDeploySnapshotReq struct {
	WebsiteID uint `json:"websiteId" validate:"required"`
}

func AppDeploySnapshot(c fiber.Ctx) error {
	req, err := e.BodyToStruct[AppDeploySnapshotReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	appSvc := service.NewAppDeployApplication(global.DB)
	if err := appSvc.Snapshot(req.WebsiteID); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}
