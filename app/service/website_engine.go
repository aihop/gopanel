package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/random"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"path/filepath"
	"strings"
)

const runnerWorkspaceMountPath = "/gopanel/workspace"

type websiteEngineDeployOptions struct {
	CodeSource          string
	Image               string
	CodeDir             string
	CodeDirFallback     string
	PreviousContainerID string
	PipelineKey         string
	PipelineVersion     string
	RunnerConfig        map[string]interface{}
}

func deployWebsiteEngine(ctx context.Context, alias string, options websiteEngineDeployOptions, progress func(format string, a ...interface{})) (int, string, string, error) {
	cli, err := docker.NewDockerClient()
	if err != nil {
		return 0, "", "", fmt.Errorf("failed to init docker client: %w", err)
	}
	if cli == nil {
		// Docker API not available — skip deployment
		// (e.g. podman on macOS without running machine).
		return 0, "", "", nil
	}
	defer cli.Close()
	var imageName string
	var containerPort string
	var cmd []string
	var envs []string
	rc := parseRunnerConfig(options.RunnerConfig)
	if options.CodeSource == "git" || options.CodeSource == "pipeline" {
		imageName = strings.TrimSpace(options.Image)
	} else {
		return 0, "", "", errors.New("unsupported container deployment source: " + options.CodeSource)
	}
	if options.CodeSource == "pipeline" && strings.TrimSpace(rc.BaseImage) != "" {
		imageName = strings.TrimSpace(rc.BaseImage)
	}
	if imageName == "" {
		imageName = "node:20-alpine"
	}
	if _, _, err := cli.ImageInspectWithRaw(ctx, imageName); err == nil {
		global.LOG.Infof("Using local engine image: %s", imageName)
		logEngineProgress(progress, "正在使用本地镜像: %s", imageName)
	} else {
		if !dockerclient.IsErrNotFound(err) {
			return 0, "", "", fmt.Errorf("failed to inspect image %s: %w", imageName, err)
		}
		global.LOG.Infof("Local image not found, pulling engine image: %s", imageName)
		logEngineProgress(progress, "本地未找到镜像，正在拉取: %s", imageName)
		reader, pullErr := cli.ImagePull(ctx, imageName, image.PullOptions{})
		if pullErr != nil {
			return 0, "", "", fmt.Errorf("failed to pull image %s: %w", imageName, pullErr)
		}
		defer reader.Close()
		_, _ = io.Copy(io.Discard, reader)
	}
	imageInspect, err := cli.ImageInspect(ctx, imageName)
	if err != nil {
		return 0, "", "", fmt.Errorf("failed to inspect image metadata %s: %w", imageName, err)
	}
	if len(imageInspect.Config.Env) > 0 {
		envs = append(envs, imageInspect.Config.Env...)
	}
	containerPort = strings.TrimSpace(rc.ContainerPort)
	if containerPort == "" {
		containerPort = detectEngineContainerPort(imageInspect)
	}
	workingDir := strings.TrimSpace(rc.WorkingDir)
	if workingDir == "" {
		workingDir = detectEngineWorkingDir(imageInspect)
	}
	publishedHostPort := resolveRunnerPublishedHostPort(rc)
	logEngineProgress(progress, "镜像运行配置: workingDir=%s, containerPort=%s", workingDir, containerPort)
	if publishedHostPort == "0" {
		logEngineProgress(progress, "Runner 发布端口策略: 自动分配宿主机端口")
	} else {
		logEngineProgress(progress, "Runner 发布端口策略: 固定宿主机端口 %s", publishedHostPort)
		if strings.TrimSpace(options.PreviousContainerID) != "" {
			logEngineProgress(progress, "检测到固定端口模式且存在旧容器，新容器需要等待旧容器释放端口后才能完成切换，期间可能有短暂中断")
		}
	}
	containerName := fmt.Sprintf("%s-%s", alias, random.RandString(4))
	codeDir := options.CodeDir
	if codeDir == "" {
		codeDir = filepath.Join(global.CONF.System.BaseDir, "www", alias)
	}
	hostConfig := &container.HostConfig{RestartPolicy: container.RestartPolicy{Name: "always"}, PortBindings: nat.PortMap{nat.Port(containerPort + "/tcp"): []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: publishedHostPort}}}}
	runnerExtraNetworks := make([]string, 0, len(rc.ExtraNetworks))
	selectedCodeDir := ""
	if options.CodeSource == "pipeline" {
		envs = mergeRunnerEnvs(envs, rc, containerPort, options.PipelineVersion)
		runnerSourceMountDir := resolveRunnerSourceMountDir(rc, workingDir)
		cmd = []string{"sh", "-lc", buildRunnerScript(rc, runnerSourceMountDir)}
		logEngineProgress(progress, "Runner 配置: baseImage=%s, mode=%s", imageName, strings.TrimSpace(rc.Mode))
		logEngineProgress(progress, "Runner 目录语义: sourceMountDir=%s, workingDir=%s", runnerSourceMountDir, workingDir)
		logEngineProgress(progress, "Runner 启动脚本已生成")
		const runnerNetworkName = "gopanel-network"
		if err := docker.EnsureNetwork(runnerNetworkName); err != nil {
			return 0, "", "", fmt.Errorf("failed to ensure runner network %s: %w", runnerNetworkName, err)
		}
		hostConfig.NetworkMode = container.NetworkMode(runnerNetworkName)
		logEngineProgress(progress, "Runner 默认接入网络: %s", runnerNetworkName)
		for _, networkName := range rc.ExtraNetworks {
			if networkName == runnerNetworkName {
				continue
			}
			if _, err := cli.NetworkInspect(ctx, networkName, network.InspectOptions{}); err != nil {
				return 0, "", "", fmt.Errorf("runner extra network %s 不存在或不可用: %w", networkName, err)
			}
			runnerExtraNetworks = append(runnerExtraNetworks, networkName)
		}
		if len(runnerExtraNetworks) > 0 {
			logEngineProgress(progress, "Runner 额外接入网络: %s", strings.Join(runnerExtraNetworks, ", "))
		}
		selectedCodeDir = strings.TrimSpace(options.CodeDirFallback)
		if selectedCodeDir == "" {
			selectedCodeDir = strings.TrimSpace(codeDir)
		}
		if selectedCodeDir == "" {
			return 0, "", "", fmt.Errorf("runner 代码目录为空")
		}
		logEngineProgress(progress, "Runner 项目类型: %s", detectRunnerProjectKind(selectedCodeDir, rc))
		if rc.HasCustomWorkingDir {
			hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", selectedCodeDir, runnerSourceMountDir))
			logEngineProgress(progress, "Runner 使用自定义 workingDir，代码源直接挂载到最终运行目录: %s -> %s", selectedCodeDir, runnerSourceMountDir)
		} else {
			hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s:ro", selectedCodeDir, runnerSourceMountDir))
			logEngineProgress(progress, "Runner 未自定义 workingDir，代码源先挂到只读中间目录: %s -> %s (ro)", selectedCodeDir, runnerSourceMountDir)
		}
		persistentBinds, err := buildRunnerPersistentBinds(options.PipelineKey, rc, workingDir)
		if err != nil {
			return 0, "", "", fmt.Errorf("failed to prepare runner persistent dirs: %w", err)
		}
		if len(persistentBinds) > 0 {
			hostConfig.Binds = append(hostConfig.Binds, persistentBinds...)
			logEngineProgress(progress, "已追加 %d 个持久化目录映射", len(persistentBinds))
		}
		ephemeralMounts := buildRunnerEphemeralMounts(rc, workingDir, selectedCodeDir)
		if len(ephemeralMounts) > 0 {
			hostConfig.Mounts = append(hostConfig.Mounts, ephemeralMounts...)
			logEngineProgress(progress, "Runner 依赖隔离: 已启用 node_modules 临时卷（Node 项目）")
		} else {
			logEngineProgress(progress, "Runner 依赖隔离: 未启用（非 Node 项目）")
		}
	}
	config := &container.Config{Image: imageName, Env: envs, Cmd: cmd, WorkingDir: workingDir, ExposedPorts: imageInspect.Config.ExposedPorts}
	if v := strings.TrimSpace(rc.RunnerUser); v != "" {
		config.User = v
		logEngineProgress(progress, "Runner 容器内运行用户: %s", v)
	}
	if config.ExposedPorts == nil {
		config.ExposedPorts = make(nat.PortSet)
	}
	config.ExposedPorts[nat.Port(containerPort+"/tcp")] = struct{}{}
	docker.EnsureContainerLogConfig(hostConfig)
	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return 0, "", "", fmt.Errorf("failed to create engine container: %w", err)
	}
	logEngineProgress(progress, "已创建容器: %s", containerName)
	for _, networkName := range runnerExtraNetworks {
		if err := cli.NetworkConnect(ctx, networkName, resp.ID, &network.EndpointSettings{}); err != nil {
			_ = cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			return 0, "", "", fmt.Errorf("runner 接入额外网络 %s 失败: %w", networkName, err)
		}
		logEngineProgress(progress, "已接入额外网络: %s", networkName)
	}
	logEngineProgress(progress, "正在启动容器...")
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		if options.PreviousContainerID != "" && strings.Contains(err.Error(), "port is already allocated") {
			logEngineProgress(progress, "检测到固定端口冲突，正在停止旧容器以释放端口...")
			_ = cli.ContainerStop(ctx, options.PreviousContainerID, container.StopOptions{})
			if retryErr := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); retryErr != nil {
				return 0, "", "", fmt.Errorf("停止旧容器后启动新容器仍失败: %w", retryErr)
			}
			logEngineProgress(progress, "已成功抢占固定端口并启动新容器")
		} else {
			return 0, "", "", fmt.Errorf("failed to start engine container: %w", err)
		}
	}
	stopStreaming := startEngineContainerLogStreaming(ctx, cli, resp.ID, progress)
	defer stopStreaming()
	logEngineProgress(progress, "正在等待容器端口绑定: %s/tcp", containerPort)
	bindings, err := waitForEnginePortBinding(ctx, cli, resp.ID, containerPort)
	if err != nil {
		return 0, "", "", err
	}
	var hostPort int
	fmt.Sscanf(bindings[0].HostPort, "%d", &hostPort)
	return hostPort, resp.ID, selectedCodeDir, nil
}

type runnerConfig struct {
	Mode                string
	BaseImage           string
	WorkingDir          string
	HasCustomWorkingDir bool
	ContainerPort       string
	HostPort            string
	RunnerUser          string
	StartCommand        string
	BuildCommand        string
	PreStart            string
	Env                 map[string]string
	PersistentPaths     []string
	ExtraNetworks       []string
}
