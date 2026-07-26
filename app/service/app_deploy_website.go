package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
)

func deployStaticWebsite(website *model.Website, releaseDir string) error {
	prevSiteDir := website.SiteDir
	website.SiteDir = releaseDir
	if err := global.DB.Save(website).Error; err != nil {
		return err
	}
	if err := ApplyCaddyFromDB(context.Background()); err != nil {
		website.SiteDir = prevSiteDir
		_ = global.DB.Save(website).Error
		return fmt.Errorf("应用静态站点配置失败: %w", err)
	}
	return nil
}

// deployWebAppWebsite 部署Web应用网站
// @param website 网站模型
// @param releaseDir 发布目录
// @param runtimeDir 运行时目录
// @param imageTag 镜像标签
// @param pipelineRecordID 流水线记录ID
// @param allowPipelineBridge 是否允许流水线桥接
// @param exposePort 暴露端口
// @param version 本次部署的版本号，注入容器环境变量供应用读取
// @return int, string, string, error
// @return exposePort 暴露端口
// @return containerID 容器ID
// @return runtimeDir 运行时目录
// @return error 错误
func deployWebAppWebsite(website *model.Website, releaseDir, runtimeDir, imageTag string, pipelineRecordID uint, allowPipelineBridge bool, exposePort int, version string) (int, string, string, error) {
	if website.PipelineID > 0 && allowPipelineBridge {
		if hostPort, containerID, actualRuntimeDir, ok, err := resolvePipelineRunnerBridge(website, pipelineRecordID); err != nil {
			return 0, "", "", err
		} else if ok {
			oldContainerID := strings.TrimSpace(website.ContainerID)
			if err := switchWebsiteRuntimeTarget(website, fmt.Sprintf("127.0.0.1:%d", hostPort), containerID, actualRuntimeDir); err != nil {
				return 0, "", "", err
			}
			appendPipelineDeployInfoLog(pipelineRecordID, website.Alias, fmt.Sprintf("检测到流水线 Runner 结果，已桥接代理到 127.0.0.1:%d", hostPort))
			cleanupPreviousWebsiteContainer(oldContainerID, containerID, pipelineRecordID, website.Alias)
			return hostPort, containerID, actualRuntimeDir, nil
		}
		if hostPort, containerID, ok, err := resolvePipelineScriptProxyTarget(website, pipelineRecordID, exposePort); err != nil {
			return 0, "", "", err
		} else if ok {
			oldContainerID := strings.TrimSpace(website.ContainerID)
			runtimeDir = strings.TrimSpace(runtimeDir)
			if runtimeDir == "" {
				runtimeDir = strings.TrimSpace(website.RuntimeDir)
			}
			if err := switchWebsiteRuntimeTarget(website, fmt.Sprintf("127.0.0.1:%d", hostPort), containerID, runtimeDir); err != nil {
				return 0, "", "", err
			}
			if imageRef := strings.TrimSpace(imageTag); imageRef != "" {
				website.EngineEnv = imageRef
				if err := global.DB.Save(website).Error; err != nil {
					return 0, "", "", err
				}
			}
			appendPipelineDeployInfoLog(pipelineRecordID, website.Alias, fmt.Sprintf("检测到纯脚本自管运行，已切换代理到 127.0.0.1:%d", hostPort))
			cleanupPreviousWebsiteContainer(oldContainerID, containerID, pipelineRecordID, website.Alias)
			return hostPort, containerID, runtimeDir, nil
		}
	}
	imageRef := strings.TrimSpace(imageTag)
	if imageRef == "" {
		imageRef = strings.TrimSpace(website.EngineEnv)
	}
	if strings.EqualFold(imageRef, "pipeline") {
		imageRef = ""
	}
	if imageRef == "" {
		return 0, "", "", fmt.Errorf("缺少可部署的镜像标签，请先为流水线配置产出镜像名并重新构建")
	}
	previousContainerID := website.ContainerID
	preferredRuntimeDir := strings.TrimSpace(runtimeDir)
	if preferredRuntimeDir == "" {
		preferredRuntimeDir = strings.TrimSpace(website.RuntimeDir)
	}
	if website.PipelineID > 0 {
		_, err := repo.NewPipeline(global.DB).Get(website.PipelineID)
		if err != nil {
			return 0, "", "", fmt.Errorf("读取流水线配置失败: %w", err)
		}
	}
	req := &request.WebsiteCreate{CodeSource: "pipeline", GitRepo: imageRef, CodeDir: preferredRuntimeDir, CodeDirFallback: releaseDir, PreviousContainerID: previousContainerID, Proxy: "", PipelineVersion: strings.TrimSpace(version)}
	hostPort, containerID, actualRuntimeDir, err := DeployWebsiteEngine(context.Background(), website.Alias, req, func(format string, a ...interface{}) {
		appendPipelineDeployInfoLog(pipelineRecordID, website.Alias, fmt.Sprintf(format, a...))
	})
	if err != nil {
		return 0, "", "", fmt.Errorf("启动容器失败: %w", err)
	}
	oldContainerID := strings.TrimSpace(previousContainerID)
	if err := switchWebsiteRuntimeTarget(website, fmt.Sprintf("127.0.0.1:%d", hostPort), containerID, actualRuntimeDir); err != nil {
		return 0, "", "", err
	}
	website.EngineEnv = imageRef
	if err := global.DB.Save(website).Error; err != nil {
		return 0, "", "", err
	}
	cleanupPreviousWebsiteContainer(oldContainerID, containerID, pipelineRecordID, website.Alias)
	return hostPort, containerID, actualRuntimeDir, nil
}

// switchWebsiteRuntimeTarget 切换网站运行时目标
// @param website 网站模型
// @param proxy 代理
// @param containerID 容器ID
// @param runtimeDir 运行时目录
// @return error 错误
func switchWebsiteRuntimeTarget(website *model.Website, proxy, containerID, runtimeDir string) error {
	if website == nil {
		return fmt.Errorf("website is nil")
	}
	prevProxy := website.Proxy
	prevContainerID := website.ContainerID
	prevRuntimeDir := website.RuntimeDir
	prevStatus := website.Status
	website.Proxy = proxy
	website.ContainerID = containerID
	website.RuntimeDir = runtimeDir
	website.Status = "Running"
	if err := global.DB.Save(website).Error; err != nil {
		return err
	}
	if err := ApplyCaddyFromDB(context.Background()); err != nil {
		website.Proxy = prevProxy
		website.ContainerID = prevContainerID
		website.RuntimeDir = prevRuntimeDir
		website.Status = prevStatus
		_ = global.DB.Save(website).Error
		return fmt.Errorf("切换网站代理失败: %w", err)
	}
	return nil
}

func cleanupPreviousWebsiteContainer(oldContainerID, newContainerID string, pipelineRecordID uint, websiteAlias string) {
	oldContainerID = strings.TrimSpace(oldContainerID)
	newContainerID = strings.TrimSpace(newContainerID)
	if oldContainerID == "" || oldContainerID == newContainerID {
		return
	}
	if err := cleanupPreviousContainer(oldContainerID); err != nil {
		appendPipelineDeployErrorLog(pipelineRecordID, websiteAlias, fmt.Sprintf("清理旧容器 %s 失败: %v", oldContainerID, err))
		return
	}
	appendPipelineDeployInfoLog(pipelineRecordID, websiteAlias, fmt.Sprintf("已在切换成功后清理旧容器 %s", oldContainerID))
}
