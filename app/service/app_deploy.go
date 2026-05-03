package service

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	container "github.com/docker/docker/api/types/container"
)

func ProcessAppDeployment(website model.Website, pipelineRecordID uint, version, zipPath, releaseDir, runtimeDir, imageTag string) (*model.AppDeploy, error) {
	if err := global.DB.Preload("Domains").First(&website, website.ID).Error; err != nil {
		return nil, fmt.Errorf("加载网站信息失败: %w", err)
	}

	deploy := model.AppDeploy{
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

	deploy := model.AppDeploy{
		WebsiteID:        website.ID,
		ReleaseID:        release.ID,
		PipelineRecordID: release.PipelineRecordID,
		Version:          version,
		SourceType:       "release",
		SourceUrl:        strings.TrimSpace(release.ArchiveFile),
		ArchiveFile:      strings.TrimSpace(release.ArchiveFile),
		ReleaseDir:       releaseDir,
		ImageTag:         strings.TrimSpace(release.ImageTag),
		Status:           "Building",
		LogText:          fmt.Sprintf("开始基于正式版本 %s 发布\n", version),
	}
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

func deployProxyWebsite(website *model.Website) error {
	if err := ApplyCaddyFromDB(context.Background()); err != nil {
		return fmt.Errorf("应用代理站点配置失败: %w", err)
	}
	return nil
}

func deployWebAppWebsite(website *model.Website, releaseDir, runtimeDir, imageTag string, pipelineRecordID uint, allowPipelineBridge bool) (int, string, string, error) {
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
		if hostPort, containerID, ok, err := resolvePipelineScriptProxyTarget(website, pipelineRecordID); err != nil {
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

	oldContainerID := strings.TrimSpace(previousContainerID)
	if err := switchWebsiteRuntimeTarget(website, fmt.Sprintf("127.0.0.1:%d", hostPort), containerID, actualRuntimeDir); err != nil {
		return 0, "", "", err
	}
	website.EngineEnv = imageRef
	if err := global.DB.Save(website).Error; err != nil {
		return 0, "", "", err
	}
	cleanupPreviousWebsiteContainer(oldContainerID, containerID, pipelineRecordID, website.Alias)

	// if _, err := NewCaddy().ReplaceServerBlock(buildDeployCaddyDomain(*website), website.Proxy, BuildOtherDomains(*website), website.Protocol); err != nil {
	// 	return 0, "", "", fmt.Errorf("更新代理失败: %w", err)
	// }

	// if req.PreviousContainerID != "" && req.PreviousContainerID != containerID {
	// 	err := cleanupPreviousContainer(req.PreviousContainerID)
	// 	if err != nil {
	// 		appendPipelineDeployErrorLog(pipelineRecordID, website.Alias, fmt.Sprintf("清理旧容器 %s 失败: %v", req.PreviousContainerID, err))
	// 	} else {
	// 		appendPipelineDeployInfoLog(pipelineRecordID, website.Alias, fmt.Sprintf("旧容器 %s 已成功清理", req.PreviousContainerID))
	// 	}
	// }

	return hostPort, containerID, actualRuntimeDir, nil
}

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

func resolvePipelineRunnerBridge(website *model.Website, pipelineRecordID uint) (int, string, string, bool, error) {
	if website == nil || website.PipelineID == 0 {
		return 0, "", "", false, nil
	}
	pipeline, err := repo.NewPipeline(global.DB).Get(website.PipelineID)
	if err != nil {
		return 0, "", "", false, fmt.Errorf("读取流水线配置失败: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(pipeline.RunnerMode), "runner") {
		return 0, "", "", false, nil
	}

	recordRepo := repo.NewPipelineRecord(global.DB)
	var record *model.PipelineRecord
	if pipelineRecordID > 0 {
		rec, err := recordRepo.Get(pipelineRecordID)
		if err == nil && rec != nil && rec.PipelineID == website.PipelineID {
			record = rec
		}
	}
	if record == nil {
		rec, err := recordRepo.LatestByPipelineID(website.PipelineID)
		if err != nil {
			return 0, "", "", false, nil
		}
		record = rec
	}
	if record == nil || record.RunnerHostPort <= 0 {
		return 0, "", "", false, nil
	}
	return record.RunnerHostPort, strings.TrimSpace(record.RunnerContainerID), strings.TrimSpace(record.RunnerReleaseDir), true, nil
}

func resolvePipelineScriptProxyTarget(website *model.Website, pipelineRecordID uint) (int, string, bool, error) {
	if website == nil || website.PipelineID == 0 {
		return 0, "", false, nil
	}
	pipeline, err := repo.NewPipeline(global.DB).Get(website.PipelineID)
	if err != nil {
		return 0, "", false, fmt.Errorf("读取流水线配置失败: %w", err)
	}
	if strings.EqualFold(strings.TrimSpace(pipeline.RunnerMode), "runner") {
		return 0, "", false, nil
	}

	containerName := strings.TrimSpace(pipeline.PipelineKey)
	if containerName == "" {
		containerName = strings.TrimSpace(website.Alias)
	}
	if containerName == "" {
		return 0, "", false, fmt.Errorf("纯脚本流水线缺少稳定容器名，无法自动识别运行端口；请为流水线设置 pipelineKey")
	}

	hostPort, containerID, err := detectScriptRuntimePortByContainerName(containerName, pipeline.ExposePort)
	if err != nil {
		return 0, "", false, fmt.Errorf("纯脚本流水线未检测到可用容器端口，请确认脚本已成功启动容器 %s: %w", containerName, err)
	}
	if pipelineRecordID > 0 {
		_ = repo.NewPipelineRecord(global.DB).UpdateRunnerResult(pipelineRecordID, "", containerID, hostPort)
	}
	return hostPort, containerID, true, nil
}

func detectScriptRuntimePortByContainerName(containerName string, preferredPort int) (int, string, error) {
	cli, err := docker.NewDockerClient()
	if err != nil {
		return 0, "", err
	}
	defer cli.Close()

	inspect, err := cli.ContainerInspect(context.Background(), containerName)
	if err != nil {
		return 0, "", err
	}
	if inspect.State == nil || !inspect.State.Running {
		return 0, "", fmt.Errorf("容器 %s 未在运行", containerName)
	}

	hostPort, err := choosePublishedHostPort(inspect, preferredPort)
	if err != nil {
		return 0, "", err
	}
	if err := verifyLocalProxyPortReachable(hostPort); err != nil {
		return 0, "", fmt.Errorf("容器 %s 已运行，但宿主机端口 %d 当前不可访问: %w", containerName, hostPort, err)
	}
	return hostPort, inspect.ID, nil
}

func choosePublishedHostPort(inspect container.InspectResponse, preferredPort int) (int, error) {
	portBindings := inspect.NetworkSettings.Ports
	if len(portBindings) == 0 && inspect.ContainerJSONBase != nil && inspect.ContainerJSONBase.HostConfig != nil {
		portBindings = inspect.ContainerJSONBase.HostConfig.PortBindings
	}
	if len(portBindings) == 0 {
		return 0, fmt.Errorf("容器没有可用的端口映射")
	}

	type portCandidate struct {
		hostPort    int
		privatePort int
	}
	seen := make(map[int]portCandidate)
	for key, bindings := range portBindings {
		privatePort, err := strconv.Atoi(key.Port())
		if err != nil {
			continue
		}
		for _, binding := range bindings {
			hostPort, err := strconv.Atoi(strings.TrimSpace(binding.HostPort))
			if err != nil || hostPort <= 0 {
				continue
			}
			hostIP := strings.TrimSpace(binding.HostIP)
			if hostIP != "" && hostIP != "127.0.0.1" && hostIP != "0.0.0.0" && hostIP != "::" && hostIP != "::1" {
				continue
			}
			if _, ok := seen[hostPort]; !ok {
				seen[hostPort] = portCandidate{hostPort: hostPort, privatePort: privatePort}
			}
		}
	}
	if len(seen) == 0 {
		return 0, fmt.Errorf("容器没有可识别的宿主机端口映射")
	}
	if preferredPort > 0 {
		for _, candidate := range seen {
			if candidate.privatePort == preferredPort || candidate.hostPort == preferredPort {
				return candidate.hostPort, nil
			}
		}
	}
	if len(seen) == 1 {
		for _, candidate := range seen {
			return candidate.hostPort, nil
		}
	}
	return 0, fmt.Errorf("容器存在多个端口映射，无法自动判断入口端口")
}

func verifyLocalProxyPortReachable(port int) error {
	target := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", target, 2*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
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
