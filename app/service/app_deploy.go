package service

import (
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// ProcessAppDeployment 处理应用部署
func ProcessAppDeployment(website model.Website, version, zipPath, releaseDir, runtimeDir, imageTag, sourceType string) (*model.AppDeploy, error) {
	if err := global.DB.Preload("Domains").First(&website, website.ID).Error; err != nil {
		return nil, fmt.Errorf("加载网站信息失败: %w", err)
	}
	deploy := model.AppDeploy{
		WebsiteID:   website.ID,
		Version:     version,
		SourceType:  strings.TrimSpace(sourceType),
		SourceUrl:   zipPath,
		ArchiveFile: zipPath,
		ReleaseDir:  releaseDir,
		RuntimeDir:  runtimeDir,
		ImageTag:    imageTag,
		Status:      "Building",
		LogText:     appDeployStartMessage(sourceType, version),
	}
	if err := global.DB.Create(&deploy).Error; err != nil {
		return nil, err
	}
	return runAppDeployment(&website, &deploy)
}

// ReuseAppDeployment 处理重新部署
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
