package service

import (
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"os"
	"path/filepath"
	"strings"
)

func ProcessAppDeployment(website model.Website, pipelineRecordID uint, version, zipPath, releaseDir, runtimeDir, imageTag string) (*model.AppDeploy, error) {
	if err := global.DB.Preload("Domains").First(&website, website.ID).Error; err != nil {
		return nil, fmt.Errorf("加载网站信息失败: %w", err)
	}
	deploy := model.AppDeploy{WebsiteID: website.ID, PipelineRecordID: pipelineRecordID, Version: version, SourceType: "pipeline", SourceUrl: zipPath, ArchiveFile: zipPath, ReleaseDir: releaseDir, RuntimeDir: runtimeDir, ImageTag: imageTag, Status: "Building", LogText: "开始部署版本 " + version + "\n"}
	if err := global.DB.Create(&deploy).Error; err != nil {
		return nil, err
	}
	return runAppDeployment(&website, &deploy)
}
func ProcessReleaseAppDeployment(website model.Website, release *model.Release) (*model.AppDeploy, error) {
	if release == nil {
		return nil, fmt.Errorf("release 不存在")
	}
	if err := global.DB.Preload("Domains").First(&website, website.ID).Error; err != nil {
		return nil, fmt.Errorf("加载网站信息失败: %w", err)
	}
	version := strings.TrimSpace(release.Version)
	if version == "" {
		version = fmt.Sprintf("release-%d", release.ID)
	}
	releaseDir := strings.TrimSpace(release.ReleaseDir)
	if releaseDir == "" {
		releaseDir = filepath.Join(global.CONF.System.BaseDir, "www", website.Alias, "releases", version)
	}
	deploy := model.AppDeploy{WebsiteID: website.ID, ReleaseID: release.ID, PipelineRecordID: release.PipelineRecordID, Version: version, SourceType: "release", SourceUrl: strings.TrimSpace(release.ArchiveFile), ArchiveFile: strings.TrimSpace(release.ArchiveFile), ReleaseDir: releaseDir, ImageTag: strings.TrimSpace(release.ImageTag), Status: "Building", LogText: fmt.Sprintf("开始基于正式版本 %s 发布\n", version)}
	if err := global.DB.Create(&deploy).Error; err != nil {
		return nil, err
	}
	return runAppDeployment(&website, &deploy)
}
func ReuseAppDeployment(website model.Website, deploy *model.AppDeploy) (*model.AppDeploy, error) {
	if deploy == nil {
		return nil, fmt.Errorf("部署记录不存在")
	}
	if err := global.DB.Preload("Domains").First(&website, website.ID).Error; err != nil {
		return nil, fmt.Errorf("加载网站信息失败: %w", err)
	}
	deploy.Status = "Building"
	deploy.Port = 0
	deploy.ContainerID = ""
	deploy.IsActive = false
	deploy.LogText += fmt.Sprintf("\n重新切换并发布版本 %s\n", deploy.Version)
	if err := global.DB.Save(deploy).Error; err != nil {
		return nil, err
	}
	return runAppDeployment(&website, deploy)
}
func runAppDeployment(website *model.Website, deploy *model.AppDeploy) (*model.AppDeploy, error) {
	pipelineRecordID := deploy.PipelineRecordID
	appendLog := func(msg string) {
		deploy.LogText += msg + "\n"
		_ = global.DB.Save(deploy).Error
		appendPipelineDeployInfoLog(pipelineRecordID, website.Alias, msg)
	}
	failDeploy := func(err error) {
		deploy.Status = "Failed"
		errMsg := fmt.Sprintf("部署失败: %v", err)
		deploy.LogText += errMsg + "\n"
		_ = global.DB.Save(deploy).Error
		appendPipelineDeployErrorLog(pipelineRecordID, website.Alias, errMsg)
	}
	if deploy.ArchiveFile != "" {
		appendLog("正在解压产物代码...")
		if err := UnzipFile(deploy.ArchiveFile, deploy.ReleaseDir); err != nil {
			failDeploy(err)
			return deploy, err
		}
	} else {
		appendLog("无 ZIP 产物，跳过解压。")
		if err := os.MkdirAll(deploy.ReleaseDir, 0755); err != nil {
			failDeploy(err)
			return deploy, err
		}
	}
	var err error
	if website.Type == constant.Static {
		appendLog("静态网站类型，准备切换发布目录...")
		err = deployStaticWebsite(website, deploy.ReleaseDir)
	} else if website.Type == constant.Proxy {
		appendLog("反向代理类型，应用由流水线自行管理运行。更新代理指向...")
		err = deployProxyWebsite(website)
	} else if website.Type == constant.WebApp {
		appendLog("容器化应用类型，开始部署...")
		deploy.Port, deploy.ContainerID, deploy.RuntimeDir, err = deployWebAppWebsite(website, deploy.ReleaseDir, deploy.RuntimeDir, deploy.ImageTag, pipelineRecordID, deploy.ReleaseID == 0)
		if err == nil {
			appendLog(fmt.Sprintf("容器已启动，映射端口: %d", deploy.Port))
			if deploy.RuntimeDir != "" {
				appendLog(fmt.Sprintf("本次沿用运行目录: %s", deploy.RuntimeDir))
			}
		}
	} else {
		err = fmt.Errorf("暂不支持的网站类型: %s", website.Type)
	}
	if err != nil {
		failDeploy(err)
		return deploy, err
	}
	global.DB.Model(&model.AppDeploy{}).Where("website_id = ? AND id <> ?", website.ID, deploy.ID).Update("is_active", false)
	deploy.Status = "Running"
	deploy.IsActive = true
	appendLog("🎉 部署成功并已生效！")
	return deploy, nil
}
