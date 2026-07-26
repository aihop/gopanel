package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

// runAppDeployment 运行应用部署
// @param website 网站模型
// @param deploy 应用部署模型
// @param exposePort 暴露端口
// @return *model.AppDeploy 应用部署模型
// @return error 错误
func runAppDeployment(website *model.Website, deploy *model.AppDeploy, exposePort int) (*model.AppDeploy, error) {
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
	switch website.Type {
	case constant.Static:
		appendLog("静态网站类型，准备切换发布目录...")
		err = deployStaticWebsite(website, deploy.ReleaseDir)
	case constant.Proxy:
		appendLog("反向代理类型，应用由流水线自行管理运行。更新代理指向...")
		err = ApplyCaddyFromDB(context.Background())
	case constant.WebApp, constant.Container:
		appendLog("容器化应用类型，开始部署...")
		deploy.Port, deploy.ContainerID, deploy.RuntimeDir, err = deployWebAppWebsite(website, deploy.ReleaseDir, deploy.RuntimeDir, deploy.ImageTag, pipelineRecordID, deploy.ReleaseID == 0, exposePort, deploy.Version)
		if err == nil {
			appendLog(fmt.Sprintf("容器已启动，映射端口: %d", deploy.Port))
			if deploy.RuntimeDir != "" {
				appendLog(fmt.Sprintf("本次沿用运行目录: %s", deploy.RuntimeDir))
			}
		}
	default:
		err = fmt.Errorf("暂不支持的网站类型: %s", website.Type)
	}
	if err != nil {
		failDeploy(err)
		return deploy, err
	}
	global.DB.Model(&model.AppDeploy{}).Where("website_id = ? AND id <> ?", website.ID, deploy.ID).Update("is_active", false)
	deploy.Status = "Running"
	deploy.IsActive = true
	appendLog(appDeploySuccessMessage(deploy.SourceType))
	return deploy, nil
}

func appDeployStartMessage(sourceType, version string) string {
	switch strings.TrimSpace(sourceType) {
	case "pipeline_sync":
		return "开始同步构建结果 " + version + "\n"
	case "image":
		return "开始部署镜像版本 " + version + "\n"
	case "upload":
		return "开始部署归档版本 " + version + "\n"
	default:
		return "开始部署版本 " + version + "\n"
	}
}

func appDeploySuccessMessage(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case "pipeline_sync":
		return "🎉 构建结果已同步并生效！"
	default:
		return "🎉 部署成功并已生效！"
	}
}
