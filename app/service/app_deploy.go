package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// ProcessAppDeployment 处理应用部署
func ProcessAppDeployment(website model.Website, pipelineRecordID uint, version, zipPath, releaseDir, runtimeDir, imageTag, sourceType string, exposePort int) (*model.AppDeploy, error) {
	if err := global.DB.Preload("Domains").First(&website, website.ID).Error; err != nil {
		return nil, fmt.Errorf("加载网站信息失败: %w", err)
	}
	deploy := model.AppDeploy{
		WebsiteID:        website.ID,
		PipelineRecordID: pipelineRecordID,
		Version:          version,
		SourceType:       strings.TrimSpace(sourceType),
		SourceUrl:        zipPath,
		ArchiveFile:      zipPath,
		ReleaseDir:       releaseDir,
		RuntimeDir:       runtimeDir,
		ImageTag:         imageTag,
		Status:           "Building",
		LogText:          appDeployStartMessage(sourceType, version),
	}
	if err := global.DB.Create(&deploy).Error; err != nil {
		return nil, err
	}
	return runAppDeployment(&website, &deploy, exposePort)
}

// ProcessReleaseAppDeployment 处理正式版本部署
func ProcessReleaseAppDeployment(website model.Website, release *model.Release, exposePort int) (*model.AppDeploy, error) {
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
	return runAppDeployment(&website, &deploy, exposePort)
}

// ReuseAppDeployment 处理重新部署
func ReuseAppDeployment(website model.Website, deploy *model.AppDeploy, exposePort int) (*model.AppDeploy, error) {
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
	return runAppDeployment(&website, deploy, exposePort)
}
