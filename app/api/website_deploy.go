package api

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/constant"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

type DeployListReq struct {
	WebsiteID uint `json:"websiteId" validate:"required"`
}

type DeploySwitchReq struct {
	DeployID uint `json:"deployId" validate:"required"`
}

type DeployDeleteReq struct {
	DeployID uint `json:"deployId" validate:"required"`
}

func WebsiteDeployList(c fiber.Ctx) error {
	R, err := e.BodyToStruct[DeployListReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	var list []model.WebsiteDeploy
	global.DB.Where("website_id = ?", R.WebsiteID).Order("created_at desc").Find(&list)

	return c.JSON(e.Succ(list))
}

func WebsiteDeploySwitch(c fiber.Ctx) error {
	R, err := e.BodyToStruct[DeploySwitchReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	// 1. Get Target Deploy
	var targetDeploy model.WebsiteDeploy
	if err := global.DB.First(&targetDeploy, R.DeployID).Error; err != nil {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("部署记录不存在"))))
	}

	// 2. Get Website
	var website model.Website
	if err := global.DB.Preload("Domains").First(&website, targetDeploy.WebsiteID).Error; err != nil {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("网站不存在"))))
	}

	// 对流水线来源的版本，回滚本质上是“重新发布旧版本快照”
	if targetDeploy.SourceType == "pipeline" || targetDeploy.SourceType == "git" || targetDeploy.ArchiveFile != "" || targetDeploy.ImageTag != "" {
		releaseDir := targetDeploy.ReleaseDir
		if releaseDir == "" {
			releaseDir = filepath.Join(global.CONF.System.BaseDir, "wwwroot", website.Alias, "releases", targetDeploy.Version)
		}
		archiveFile := targetDeploy.ArchiveFile
		if archiveFile == "" {
			archiveFile = targetDeploy.SourceUrl
		}
		targetDeploy.ArchiveFile = archiveFile
		targetDeploy.SourceUrl = archiveFile
		targetDeploy.ReleaseDir = releaseDir
		if _, err := service.ReuseWebsiteDeployment(website, &targetDeploy); err != nil {
			return c.JSON(e.Fail(buserr.Err(fmt.Errorf("回滚发布失败: %w", err))))
		}
		return c.JSON(e.Succ())
	}

	// 兼容应用商店/compose 快照
	if website.Type == constant.Proxy {
		var appInstall model.AppInstall
		if err := global.DB.First(&appInstall, website.AppInstallID).Error; err == nil {
			envPath := appInstall.GetEnvPath()
			composePath := appInstall.GetComposePath()
			_ = os.WriteFile(envPath, []byte(targetDeploy.Env), 0644)
			_ = os.WriteFile(composePath, []byte(targetDeploy.DockerCompose), 0644)

			_ = service.NewAppInstall().Operate(request.AppInstalledOperate{
				InstallId: appInstall.ID,
				Operate:   constant.OperateUp,
			})

			appInstall.Env = targetDeploy.Env
			appInstall.DockerCompose = targetDeploy.DockerCompose
			global.DB.Save(&appInstall)

			if _, err = service.NewCaddy().ReplaceServerBlock(service.BuildWebsiteCaddyDomain(website.PrimaryDomain, website.Protocol), website.Proxy, service.BuildOtherDomains(website), website.Protocol); err != nil {
				return c.JSON(e.Fail(buserr.Err(fmt.Errorf("切换 Caddy 配置失败: %w", err))))
			}

			global.DB.Model(&model.WebsiteDeploy{}).Where("website_id = ?", website.ID).Update("is_active", false)
			targetDeploy.IsActive = true
			global.DB.Save(&targetDeploy)
			return c.JSON(e.Succ())
		}
	}

	return c.JSON(e.Fail(buserr.Err(fmt.Errorf("当前版本缺少可回滚快照"))))
}

func WebsiteDeployDelete(c fiber.Ctx) error {
	R, err := e.BodyToStruct[DeployDeleteReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	var deploy model.WebsiteDeploy
	if err := global.DB.First(&deploy, R.DeployID).Error; err != nil {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("部署记录不存在"))))
	}
	if deploy.IsActive {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("线上运行中的版本不允许删除"))))
	}
	if err := global.DB.Delete(&deploy).Error; err != nil {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("删除部署记录失败: %w", err))))
	}
	return c.JSON(e.Succ())
}

type DeployTriggerReq struct {
	WebsiteID uint   `json:"websiteId" validate:"required"`
	ZipPath   string `json:"zipPath"`
	ImageTag  string `json:"imageTag"` // 新增字段，用于触发自定义镜像部署
}

func WebsiteDeployTrigger(c fiber.Ctx) error {
	R, err := e.BodyToStruct[DeployTriggerReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	var website model.Website
	if err := global.DB.Preload("Domains").First(&website, R.WebsiteID).Error; err != nil {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("网站不存在"))))
	}

	version := fmt.Sprintf("v%d", time.Now().Unix())
	releaseDir := filepath.Join(global.CONF.System.BaseDir, "wwwroot", website.Alias, "releases", version)

	// Trigger async deployment
	go service.ProcessWebsiteDeployment(website, 0, version, R.ZipPath, releaseDir, "", R.ImageTag)

	return c.JSON(e.Succ())
}

type SnapshotReq struct {
	WebsiteID uint `json:"websiteId" validate:"required"`
}

func WebsiteDeploySnapshot(c fiber.Ctx) error {
	req, err := e.BodyToStruct[SnapshotReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	var website model.Website
	if err := global.DB.First(&website, req.WebsiteID).Error; err != nil {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("网站不存在"))))
	}

	if !(website.Type == constant.Proxy && website.AppInstallID > 0) {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("仅容器应用支持快照"))))
	}

	var appInstall model.AppInstall
	if err := global.DB.First(&appInstall, website.AppInstallID).Error; err != nil {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("关联应用不存在"))))
	}

	version := fmt.Sprintf("v%d", time.Now().Unix())

	// Create a new deploy record
	deploy := model.WebsiteDeploy{
		WebsiteID:     website.ID,
		Version:       version,
		SourceType:    "compose",
		Status:        "Running",
		LogText:       "已创建配置快照，记录了当时的 docker-compose 和环境变量配置。",
		DockerCompose: appInstall.DockerCompose,
		Env:           appInstall.Env,
		AppInstallID:  appInstall.ID,
		Port:          appInstall.HttpPort, // snapshot the current port
	}

	// Make it the active one
	global.DB.Model(&model.WebsiteDeploy{}).Where("website_id = ?", website.ID).Update("is_active", false)
	deploy.IsActive = true

	if err := global.DB.Create(&deploy).Error; err != nil {
		return c.JSON(e.Fail(buserr.Err(fmt.Errorf("创建快照失败: %v", err))))
	}

	return c.JSON(e.Succ())
}
