package service

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
)

func ProcessWebsiteDeployment(website model.Website, pipelineRecordID uint, version, zipPath, releaseDir, runtimeDir, imageTag string) (*model.WebsiteDeploy, error) {
	if err := global.DB.Preload("Domains").First(&website, website.ID).Error; err != nil {
		return nil, fmt.Errorf("加载网站信息失败: %w", err)
	}

	deploy := model.WebsiteDeploy{
		WebsiteID:        website.ID,
		PipelineRecordID: pipelineRecordID,
		Version:          version,
		SourceType:       "pipeline",
		SourceUrl:        zipPath,
		ArchiveFile:      zipPath,
		ReleaseDir:       releaseDir,
		RuntimeDir:       runtimeDir,
		ImageTag:         imageTag,
		Status:           "Building",
		LogText:          "开始部署版本 " + version + "\n",
	}
	if err := global.DB.Create(&deploy).Error; err != nil {
		return nil, err
	}

	return runWebsiteDeployment(&website, &deploy)
}

func ReuseWebsiteDeployment(website model.Website, deploy *model.WebsiteDeploy) (*model.WebsiteDeploy, error) {
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

	return runWebsiteDeployment(&website, deploy)
}

func runWebsiteDeployment(website *model.Website, deploy *model.WebsiteDeploy) (*model.WebsiteDeploy, error) {
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
		deploy.Port, deploy.ContainerID, deploy.RuntimeDir, err = deployWebAppWebsite(website, deploy.ReleaseDir, deploy.RuntimeDir, deploy.ImageTag, pipelineRecordID)
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

	global.DB.Model(&model.WebsiteDeploy{}).Where("website_id = ? AND id <> ?", website.ID, deploy.ID).Update("is_active", false)
	deploy.Status = "Running"
	deploy.IsActive = true
	appendLog("🎉 部署成功并已生效！")
	return deploy, nil
}

func BuildOtherDomains(w model.Website) string {
	var domains []string
	if w.Domains != nil {
		for _, d := range w.Domains {
			if normalizeWebsiteDomainForCompare(d.Domain) != normalizeWebsiteDomainForCompare(w.PrimaryDomain) {
				domains = append(domains, d.Domain)
			}
		}
	}
	return strings.Join(domains, "\n")
}

func buildDeployCaddyDomain(website model.Website) string {
	return BuildWebsiteCaddyDomain(website.PrimaryDomain, website.Protocol)
}

func appendPipelineDeployInfoLog(pipelineRecordID uint, websiteAlias, msg string) {
	if pipelineRecordID == 0 || !IsPipelineLoggerActive(pipelineRecordID) {
		return
	}
	GetPipelineLogger(pipelineRecordID).Info("[%s] %s", websiteAlias, msg)
}

func appendPipelineDeployErrorLog(pipelineRecordID uint, websiteAlias, msg string) {
	if pipelineRecordID == 0 || !IsPipelineLoggerActive(pipelineRecordID) {
		return
	}
	GetPipelineLogger(pipelineRecordID).Error("[%s] %s", websiteAlias, msg)
}

func deployStaticWebsite(website *model.Website, releaseDir string) error {
	website.SiteDir = releaseDir
	if err := global.DB.Save(website).Error; err != nil {
		return err
	}
	_, err := NewCaddy().ReplaceStaticServerBlock(buildDeployCaddyDomain(*website), releaseDir, BuildOtherDomains(*website), website.Protocol)
	if err != nil {
		return fmt.Errorf("配置 Caddy 失败: %w", err)
	}
	return nil
}

func deployProxyWebsite(website *model.Website) error {
	_, err := NewCaddy().ReplaceServerBlock(buildDeployCaddyDomain(*website), website.Proxy, BuildOtherDomains(*website), website.Protocol)
	if err != nil {
		return fmt.Errorf("更新代理失败: %w", err)
	}
	return nil
}

func deployWebAppWebsite(website *model.Website, releaseDir, runtimeDir, imageTag string, pipelineRecordID uint) (int, string, string, error) {
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
	req := &request.WebsiteCreate{
		CodeSource:          "pipeline",
		GitRepo:             imageRef,
		CodeDir:             preferredRuntimeDir,
		CodeDirFallback:     releaseDir,
		PreviousContainerID: previousContainerID,
		// 流水线部署时不再把 ExposePort 塞进 Proxy。
		// 容器内部监听端口应从镜像 EXPOSE / PORT 环境变量自动识别，
		// 否则会把用户配置的“外部访问端口”误当成容器私有端口。
		Proxy: "",
	}

	hostPort, containerID, actualRuntimeDir, err := DeployWebsiteEngine(context.Background(), website.Alias, req, func(format string, a ...interface{}) {
		appendPipelineDeployInfoLog(pipelineRecordID, website.Alias, fmt.Sprintf(format, a...))
	})
	if err != nil {
		return 0, "", "", fmt.Errorf("启动容器失败: %w", err)
	}

	website.Proxy = fmt.Sprintf("127.0.0.1:%d", hostPort)
	website.ContainerID = containerID
	website.EngineEnv = imageRef
	website.RuntimeDir = actualRuntimeDir
	website.Status = "Running"
	if err := global.DB.Save(website).Error; err != nil {
		return 0, "", "", err
	}

	if _, err := NewCaddy().ReplaceServerBlock(buildDeployCaddyDomain(*website), website.Proxy, BuildOtherDomains(*website), website.Protocol); err != nil {
		return 0, "", "", fmt.Errorf("更新代理失败: %w", err)
	}

	if req.PreviousContainerID != "" && req.PreviousContainerID != containerID {
		err := cleanupPreviousContainer(req.PreviousContainerID)
		if err != nil {
			appendPipelineDeployErrorLog(pipelineRecordID, website.Alias, fmt.Sprintf("清理旧容器 %s 失败: %v", req.PreviousContainerID, err))
		} else {
			appendPipelineDeployInfoLog(pipelineRecordID, website.Alias, fmt.Sprintf("旧容器 %s 已成功清理", req.PreviousContainerID))
		}
	}

	return hostPort, containerID, actualRuntimeDir, nil
}

func cleanupPreviousContainer(containerID string) error {
	cli, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return RemoveEngineContainer(context.Background(), cli, containerID)
}

func UnzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}
	cleanDest := filepath.Clean(dest) + string(os.PathSeparator)

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		cleanPath := filepath.Clean(fpath)
		if !strings.HasPrefix(cleanPath, cleanDest) {
			return fmt.Errorf("非法压缩包路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(cleanPath, os.ModePerm)
			continue
		}
		if err = os.MkdirAll(filepath.Dir(cleanPath), os.ModePerm); err != nil {
			return err
		}
		outFile, err := os.OpenFile(cleanPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
