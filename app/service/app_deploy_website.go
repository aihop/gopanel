package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
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
// @return int, string, string, error
// @return hostPort 宿主机端口
// @return containerID 容器ID
// @return runtimeDir 运行时目录
// @return error 错误
func deployWebAppWebsite(website *model.Website, releaseDir, runtimeDir, imageTag string) (int, string, string, error) {
	imageRef := strings.TrimSpace(imageTag)
	if imageRef == "" {
		imageRef = strings.TrimSpace(website.EngineEnv)
	}
	if strings.EqualFold(imageRef, "pipeline") {
		imageRef = ""
	}
	if imageRef == "" {
		return 0, "", "", fmt.Errorf("缺少可部署的镜像标签")
	}
	previousContainerID := website.ContainerID
	preferredRuntimeDir := strings.TrimSpace(runtimeDir)
	if preferredRuntimeDir == "" {
		preferredRuntimeDir = strings.TrimSpace(website.RuntimeDir)
	}
	options := websiteEngineDeployOptions{CodeSource: "git", Image: imageRef, CodeDir: preferredRuntimeDir, CodeDirFallback: releaseDir, PreviousContainerID: previousContainerID}
	hostPort, containerID, actualRuntimeDir, err := deployWebsiteEngine(context.Background(), website.Alias, options, nil)
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
	cleanupPreviousWebsiteContainer(oldContainerID, containerID)
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

func cleanupPreviousWebsiteContainer(oldContainerID, newContainerID string) {
	oldContainerID = strings.TrimSpace(oldContainerID)
	newContainerID = strings.TrimSpace(newContainerID)
	if oldContainerID == "" || oldContainerID == newContainerID {
		return
	}
	if err := cleanupPreviousContainer(oldContainerID); err != nil {
		global.LOG.Errorf("清理旧容器 %s 失败: %v", oldContainerID, err)
		return
	}
	global.LOG.Infof("已在切换成功后清理旧容器 %s", oldContainerID)
}

func cleanupPreviousContainer(containerID string) error {
	cli, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return RemoveEngineContainer(context.Background(), cli, containerID)
}
