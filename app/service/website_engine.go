package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/random"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Removed DetectEngineEnv as we are shifting to pure containerized pipelines.

const runnerWorkspaceMountPath = "/gopanel/workspace"

func DeployWebsiteEngine(ctx context.Context, alias string, req *request.WebsiteCreate, progress func(format string, a ...interface{})) (int, string, string, error) {
	cli, err := docker.NewDockerClient()
	if err != nil {
		return 0, "", "", fmt.Errorf("failed to init docker client: %w", err)
	}
	defer cli.Close()

	var imageName string
	var containerPort string
	var cmd []string
	var envs []string
	rc := parseRunnerConfig(req.RunnerConfig)
	isRunnerPipeline := req.CodeSource == "pipeline" && req.RunnerConfig != nil

	if req.CodeSource == "git" || req.CodeSource == "pipeline" {
		imageName = strings.TrimSpace(req.GitRepo)
	} else {
		return 0, "", "", errors.New("unsupported container deployment source: " + req.CodeSource)
	}
	if req.CodeSource == "pipeline" && strings.TrimSpace(rc.BaseImage) != "" {
		imageName = strings.TrimSpace(rc.BaseImage)
	}
	if imageName == "" {
		imageName = "node:20-alpine"
	}
	// 检查镜像是否存在，本地存在就用本地镜像
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

	// 获取镜像元数据
	imageInspect, err := cli.ImageInspect(ctx, imageName)
	if err != nil {
		return 0, "", "", fmt.Errorf("failed to inspect image metadata %s: %w", imageName, err)
	}

	// 继承镜像自带的环境变量
	if len(imageInspect.Config.Env) > 0 {
		envs = append(envs, imageInspect.Config.Env...)
	}

	// 容器内部监听端口默认优先使用 Runner 配置，否则回退到镜像元数据探测。
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
		if strings.TrimSpace(req.PreviousContainerID) != "" {
			logEngineProgress(progress, "检测到固定端口模式且存在旧容器，新容器需要等待旧容器释放端口后才能完成切换，期间可能有短暂中断")
		}
	}

	containerName := fmt.Sprintf("%s-%s", alias, random.RandString(4))

	// 选择代码目录
	codeDir := req.CodeDir
	if codeDir == "" {
		codeDir = filepath.Join(global.CONF.System.BaseDir, "www", alias)
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "always"},
		PortBindings: nat.PortMap{
			nat.Port(containerPort + "/tcp"): []nat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: publishedHostPort,
				},
			},
		},
	}
	runnerExtraNetworks := make([]string, 0, len(rc.ExtraNetworks))
	selectedCodeDir := ""
	if req.CodeSource == "pipeline" {
		envs = mergeRunnerEnvs(envs, rc, containerPort)
		cmd = []string{"sh", "-lc", buildRunnerScript(rc, runnerWorkspaceMountPath)}
		logEngineProgress(progress, "Runner 配置: baseImage=%s, mode=%s", imageName, strings.TrimSpace(rc.Mode))
		logEngineProgress(progress, "Runner 启动脚本已生成")
		if isRunnerPipeline {
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
			selectedCodeDir = strings.TrimSpace(req.CodeDirFallback)
			if selectedCodeDir == "" {
				selectedCodeDir = strings.TrimSpace(codeDir)
			}
			if selectedCodeDir == "" {
				return 0, "", "", fmt.Errorf("runner 代码目录为空")
			}
			logEngineProgress(progress, "Runner 项目类型: %s", detectRunnerProjectKind(selectedCodeDir, rc))
			hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s:ro", selectedCodeDir, runnerWorkspaceMountPath))
			logEngineProgress(progress, "强制挂载 Runner 工作目录: %s -> %s (ro)", selectedCodeDir, runnerWorkspaceMountPath)
			persistentBinds, err := buildRunnerPersistentBinds(req.PipelineKey, rc, workingDir)
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
		} else {
			runtimeTemplate := detectReusableRuntimeTemplate(ctx, cli, imageName, workingDir, req.PreviousContainerID)

			// 如果我们通过模板探测到了历史容器，更新 PreviousContainerID，这样后面的深度继承和清理逻辑才会生效
			if runtimeTemplate.ContainerID != "" && req.PreviousContainerID == "" {
				req.PreviousContainerID = runtimeTemplate.ContainerID
			}

			if len(runtimeTemplate.Binds) > 0 {
				hostConfig.Binds = append(hostConfig.Binds, runtimeTemplate.Binds...)
				if runtimeTemplate.NetworkMode != "" && runtimeTemplate.NetworkMode != "default" {
					hostConfig.NetworkMode = container.NetworkMode(runtimeTemplate.NetworkMode)
				}
				if len(runtimeTemplate.ExtraHosts) > 0 {
					hostConfig.ExtraHosts = append(hostConfig.ExtraHosts, runtimeTemplate.ExtraHosts...)
				}
				if len(runtimeTemplate.Env) > 0 {
					// 合并继承自旧容器的环境变量。
					// 但 PORT/HOST 这种决定容器监听行为的关键变量必须以镜像为准，
					// 不能被历史容器再次覆盖，否则会出现“这里探测到 3100 / 最终又带回 3000”这类撕裂问题。
					envMap := make(map[string]string)
					for _, e := range envs {
						parts := strings.SplitN(e, "=", 2)
						if len(parts) == 2 {
							envMap[parts[0]] = parts[1]
						}
					}
					for _, e := range runtimeTemplate.Env {
						parts := strings.SplitN(e, "=", 2)
						if len(parts) == 2 {
							if parts[0] == "PORT" || parts[0] == "HOST" {
								continue
							}
							envMap[parts[0]] = parts[1]
						}
					}
					envs = []string{}
					for k, v := range envMap {
						envs = append(envs, fmt.Sprintf("%s=%s", k, v))
					}
				}
				selectedCodeDir = runtimeTemplate.RuntimeDir
				logEngineProgress(progress, "复用历史成功容器模板(%s): mounts=%d, networkMode=%s", runtimeTemplate.Source, len(runtimeTemplate.Binds), runtimeTemplate.NetworkMode)
			} else {
				previousMountDirs := detectPreviousContainerMountDirs(ctx, cli, req.PreviousContainerID, workingDir)
				selectedCodeDir, mountReason := resolveAutoMountCodeDir(
					imageInspect,
					workingDir,
					append([]string{codeDir}, append(previousMountDirs, req.CodeDirFallback)...)...,
				)
				if selectedCodeDir != "" {
					global.LOG.Infof("Auto mounting pipeline code dir %s -> %s", selectedCodeDir, workingDir)
					logEngineProgress(progress, "自动挂载流水线产物目录: %s -> %s", selectedCodeDir, workingDir)
					hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", selectedCodeDir, workingDir))
				} else if mountReason != "" {
					logEngineProgress(progress, "跳过自动挂载: %s", mountReason)
				}
			}
		}
	}

	config := &container.Config{
		Image:        imageName,
		Env:          envs,
		Cmd:          cmd,
		WorkingDir:   workingDir,
		ExposedPorts: imageInspect.Config.ExposedPorts, // 默认使用新镜像自带的所有暴露端口
	}
	if v := strings.TrimSpace(rc.RunnerUser); v != "" {
		config.User = v
		logEngineProgress(progress, "Runner 容器内运行用户: %s", v)
	}

	// 确保探测到的主端口一定被包含在内
	if config.ExposedPorts == nil {
		config.ExposedPorts = make(nat.PortSet)
	}
	config.ExposedPorts[nat.Port(containerPort+"/tcp")] = struct{}{}

	// 完整继承旧容器的配置 (如果存在)
	if req.CodeSource == "pipeline" && req.PreviousContainerID != "" && !isRunnerPipeline {
		if oldInspect, err := cli.ContainerInspect(ctx, req.PreviousContainerID); err == nil {
			if oldInspect.Config != nil {
				// 继承原有配置
				config = oldInspect.Config
				// 更新镜像和网络相关的必要字段
				config.Image = imageName

				// 重建暴露端口集合，只保留新镜像声明的端口和当前探测出的主监听端口。
				// 不能继续混入旧容器的 PortBindings，否则会把历史错误端口（如 3100/tcp）再次写回新容器。
				config.ExposedPorts = make(nat.PortSet)
				for port := range imageInspect.Config.ExposedPorts {
					config.ExposedPorts[port] = struct{}{}
				}
				config.ExposedPorts[nat.Port(containerPort+"/tcp")] = struct{}{}

				// 智能合并环境变量: 旧运行时的 Env 默认覆盖镜像的 Env，
				// 但像 PORT/HOST 这种直接决定容器监听行为的关键变量，必须以新镜像为准，
				// 否则会把旧容器里残留的错误端口（例如 3100）再次带回新容器。
				envMap := make(map[string]string)
				for _, e := range envs { // 从镜像读出的 env
					parts := strings.SplitN(e, "=", 2)
					if len(parts) == 2 {
						envMap[parts[0]] = parts[1]
					}
				}
				for _, e := range oldInspect.Config.Env { // 之前运行时存在的 env
					parts := strings.SplitN(e, "=", 2)
					if len(parts) == 2 {
						if parts[0] == "PORT" || parts[0] == "HOST" {
							continue
						}
						envMap[parts[0]] = parts[1]
					}
				}
				mergedEnvs := make([]string, 0, len(envMap))
				for k, v := range envMap {
					mergedEnvs = append(mergedEnvs, fmt.Sprintf("%s=%s", k, v))
				}
				config.Env = mergedEnvs
			}

			if oldInspect.HostConfig != nil {
				// 继承宿主机配置
				hostConfig = oldInspect.HostConfig

				// 对于网站容器，统一只发布当前主监听端口到本机随机端口。
				// 这一步必须覆盖旧容器的端口映射键，避免把历史错误的私有端口（如 3100/tcp）继续继承下来。
				hostConfig.PortBindings = nat.PortMap{
					nat.Port(containerPort + "/tcp"): []nat.PortBinding{
						{
							HostIP:   "127.0.0.1",
							HostPort: publishedHostPort,
						},
					},
				}
			}
			logEngineProgress(progress, "深度继承旧容器(%s)的完整运行参数和配置", req.PreviousContainerID)
		}
	}
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
		// 容错处理：即使我们使用了自动/固定 HostPort，在某些极端情况下（比如旧容器强行占用固定端口），
		// 依然可能启动失败。如果是端口分配冲突，我们退回“抢占旧容器”的安全策略。
		if req.PreviousContainerID != "" && strings.Contains(err.Error(), "port is already allocated") {
			logEngineProgress(progress, "检测到固定端口冲突，正在停止旧容器以释放端口...")
			_ = cli.ContainerStop(ctx, req.PreviousContainerID, container.StopOptions{})
			// 再次尝试启动
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

func logEngineProgress(progress func(format string, a ...interface{}), format string, a ...interface{}) {
	if progress != nil {
		progress(format, a...)
	}
}

func detectEngineContainerPort(imageInspect image.InspectResponse) string {
	for _, env := range imageInspect.Config.Env {
		if strings.HasPrefix(env, "PORT=") {
			port := strings.TrimSpace(strings.TrimPrefix(env, "PORT="))
			if port != "" {
				return port
			}
		}
	}

	var ports []string
	for port := range imageInspect.Config.ExposedPorts {
		if strings.HasSuffix(string(port), "/tcp") {
			ports = append(ports, strings.TrimSuffix(string(port), "/tcp"))
		}
	}
	sort.Strings(ports)
	if len(ports) > 0 {
		return ports[0]
	}
	return "80"
}

func detectEngineWorkingDir(imageInspect image.InspectResponse) string {
	if strings.TrimSpace(imageInspect.Config.WorkingDir) != "" {
		return strings.TrimSpace(imageInspect.Config.WorkingDir)
	}
	return "/app"
}

func shouldAutoMountCodeDir(imageInspect image.InspectResponse, workingDir, codeDir string) (bool, string) {
	if strings.TrimSpace(codeDir) == "" || strings.TrimSpace(workingDir) == "" {
		return false, "挂载源目录或工作目录为空"
	}
	relativeEntry := detectRelativeEntrypoint(imageInspect)
	if relativeEntry == "" {
		return false, ""
	}
	info, err := os.Stat(codeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Sprintf("挂载源目录不存在: %s", codeDir)
		}
		return false, fmt.Sprintf("挂载源目录不可访问: %s", err)
	}
	if !info.IsDir() {
		return false, fmt.Sprintf("挂载源路径不是目录: %s", codeDir)
	}
	targetFile := filepath.Join(codeDir, strings.TrimPrefix(relativeEntry, "./"))
	fileInfo, err := os.Stat(targetFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Sprintf("挂载目录中缺少启动文件 %s", relativeEntry)
		}
		return false, fmt.Sprintf("无法检查启动文件 %s: %s", relativeEntry, err)
	}
	if fileInfo.IsDir() {
		return false, fmt.Sprintf("启动文件 %s 实际是目录", relativeEntry)
	}
	return true, ""
}

func resolveAutoMountCodeDir(imageInspect image.InspectResponse, workingDir string, candidates ...string) (string, string) {
	relativeEntry := detectRelativeEntrypoint(imageInspect)
	if relativeEntry == "" {
		return "", ""
	}

	seen := make(map[string]struct{})
	var reasons []string
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		ok, reason := shouldAutoMountCodeDir(imageInspect, workingDir, candidate)
		if ok {
			return candidate, ""
		}
		if reason != "" {
			reasons = append(reasons, fmt.Sprintf("%s (%s)", candidate, reason))
		}
	}

	if len(reasons) == 0 {
		return "", ""
	}
	return "", strings.Join(reasons, "; ")
}

func detectRelativeEntrypoint(imageInspect image.InspectResponse) string {
	commands := append([]string{}, imageInspect.Config.Entrypoint...)
	commands = append(commands, imageInspect.Config.Cmd...)
	for _, entry := range commands {
		entry = strings.TrimSpace(entry)
		if strings.HasPrefix(entry, "./") {
			return entry
		}
	}
	return ""
}

type runnerConfig struct {
	Mode            string
	BaseImage       string
	WorkingDir      string
	ContainerPort   string
	HostPort        string
	RunnerUser      string
	StartCommand    string
	BuildCommand    string
	PreStart        string
	Env             map[string]string
	PersistentPaths []string
	ExtraNetworks   []string
}

func parseRunnerConfig(raw map[string]interface{}) runnerConfig {
	rc := runnerConfig{
		Mode:          "build_run",
		BaseImage:     "node:20-alpine",
		WorkingDir:    "/var/www/app",
		ContainerPort: "3000",
		StartCommand:  "node .output/server/index.mjs",
		Env:           map[string]string{},
	}
	if raw == nil {
		return rc
	}
	if v := strings.TrimSpace(asString(raw["mode"])); v != "" {
		rc.Mode = v
	}
	if v := strings.TrimSpace(asString(raw["baseImage"])); v != "" {
		rc.BaseImage = v
	}
	if v := strings.TrimSpace(asString(raw["workingDir"])); v != "" {
		rc.WorkingDir = v
	}
	if v := strings.TrimSpace(asNumberString(raw["containerPort"])); v != "" {
		rc.ContainerPort = v
	}
	if v := strings.TrimSpace(asNumberString(raw["hostPort"])); v != "" {
		rc.HostPort = v
	}
	if v := strings.TrimSpace(asString(raw["runnerUser"])); v != "" {
		rc.RunnerUser = v
	}
	if v, ok := raw["startCommand"]; ok {
		rc.StartCommand = strings.TrimSpace(asString(v))
	}
	if v := strings.TrimSpace(asString(raw["buildCommand"])); v != "" {
		rc.BuildCommand = v
	}
	if v := asString(raw["preStart"]); strings.TrimSpace(v) != "" {
		rc.PreStart = v
	}
	if envRaw, ok := raw["env"].(map[string]interface{}); ok {
		for k, v := range envRaw {
			rc.Env[k] = asString(v)
		}
	} else if envRaw, ok := raw["env"].(map[string]string); ok {
		for k, v := range envRaw {
			rc.Env[k] = v
		}
	}
	if pathsRaw, ok := raw["persistentPaths"].([]interface{}); ok {
		for _, item := range pathsRaw {
			if v := strings.TrimSpace(asString(item)); v != "" {
				rc.PersistentPaths = append(rc.PersistentPaths, v)
			}
		}
	} else if pathsRaw, ok := raw["persistentPaths"].([]string); ok {
		for _, item := range pathsRaw {
			if v := strings.TrimSpace(item); v != "" {
				rc.PersistentPaths = append(rc.PersistentPaths, v)
			}
		}
	}
	if networksRaw, ok := raw["extraNetworks"].([]interface{}); ok {
		for _, item := range networksRaw {
			if v := strings.TrimSpace(asString(item)); v != "" {
				rc.ExtraNetworks = append(rc.ExtraNetworks, v)
			}
		}
	} else if networksRaw, ok := raw["extraNetworks"].([]string); ok {
		for _, item := range networksRaw {
			if v := strings.TrimSpace(item); v != "" {
				rc.ExtraNetworks = append(rc.ExtraNetworks, v)
			}
		}
	}
	rc.ExtraNetworks = normalizeRunnerExtraNetworks(rc.ExtraNetworks)
	return rc
}

func resolveRunnerPublishedHostPort(rc runnerConfig) string {
	if v := strings.TrimSpace(rc.HostPort); v != "" {
		return v
	}
	return "0"
}

func asString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func asNumberString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case float32:
		return strconv.FormatInt(int64(t), 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case string:
		return t
	default:
		return ""
	}
}

func mergeRunnerEnvs(base []string, rc runnerConfig, containerPort string) []string {
	envMap := make(map[string]string)
	for _, e := range base {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for k, v := range rc.Env {
		ks := strings.TrimSpace(k)
		if ks == "" {
			continue
		}
		envMap[ks] = v
	}
	if strings.TrimSpace(containerPort) != "" {
		envMap["PORT"] = strings.TrimSpace(containerPort)
	}
	if _, ok := envMap["HOST"]; !ok {
		envMap["HOST"] = "0.0.0.0"
	}

	out := make([]string, 0, len(envMap))
	for k, v := range envMap {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}

func buildRunnerPersistentBinds(pipelineKey string, rc runnerConfig, workingDir string) ([]string, error) {
	if len(rc.PersistentPaths) == 0 {
		return nil, nil
	}
	wd := strings.TrimSpace(workingDir)
	if wd == "" {
		wd = "/var/www/app"
	}
	key := strings.TrimSpace(pipelineKey)
	if key == "" {
		key = "pipeline-runtime"
	}
	baseDir := filepath.Join(global.CONF.System.BaseDir, "apps", key)
	binds := make([]string, 0, len(rc.PersistentPaths))
	for _, raw := range rc.PersistentPaths {
		subPath, target, ok := normalizeRunnerPersistentPath(wd, raw)
		if !ok {
			continue
		}
		hostDir := filepath.Join(baseDir, filepath.FromSlash(subPath))
		if err := os.MkdirAll(hostDir, 0755); err != nil {
			return nil, err
		}
		binds = append(binds, fmt.Sprintf("%s:%s", hostDir, target))
	}
	return binds, nil
}

func normalizeRunnerPersistentPath(workingDir string, raw string) (string, string, bool) {
	wd := path.Clean(strings.TrimSpace(workingDir))
	if wd == "." || wd == "/" || wd == "" {
		wd = "/var/www/app"
	}
	candidate := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if candidate == "" {
		return "", "", false
	}
	if strings.HasPrefix(candidate, wd+"/") {
		candidate = strings.TrimPrefix(candidate, wd+"/")
	} else {
		candidate = strings.TrimPrefix(candidate, "/")
	}
	candidate = strings.TrimPrefix(path.Clean("/"+candidate), "/")
	if candidate == "" || candidate == "." {
		return "", "", false
	}
	target := path.Join(wd, candidate)
	return candidate, target, true
}

func buildRunnerEphemeralMounts(rc runnerConfig, workingDir, sourceDir string) []mount.Mount {
	if strings.ToLower(strings.TrimSpace(rc.Mode)) != "build_run" {
		return nil
	}
	if !runnerNeedsNodeModulesIsolation(rc, sourceDir) {
		return nil
	}
	wd := path.Clean(strings.TrimSpace(workingDir))
	if wd == "." || wd == "/" || wd == "" {
		wd = "/var/www/app"
	}
	for _, raw := range rc.PersistentPaths {
		subPath, _, ok := normalizeRunnerPersistentPath(wd, raw)
		if ok && subPath == "node_modules" {
			return nil
		}
	}
	return []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Target: path.Join(wd, "node_modules"),
		},
	}
}

func runnerNeedsNodeModulesIsolation(rc runnerConfig, sourceDir string) bool {
	baseImage := strings.ToLower(strings.TrimSpace(rc.BaseImage))
	startCmd := strings.ToLower(strings.TrimSpace(rc.StartCommand))
	buildCmd := strings.ToLower(strings.TrimSpace(rc.BuildCommand))

	if strings.Contains(baseImage, "node") {
		return true
	}
	for _, token := range []string{"npm", "pnpm", "yarn", ".output/server/index.mjs", "node "} {
		if strings.Contains(startCmd, token) || strings.Contains(buildCmd, token) {
			return true
		}
	}
	if strings.TrimSpace(sourceDir) != "" {
		if _, err := os.Stat(filepath.Join(sourceDir, "package.json")); err == nil {
			return true
		}
	}
	return false
}

func detectRunnerProjectKind(sourceDir string, rc runnerConfig) string {
	if strings.TrimSpace(sourceDir) != "" {
		switch {
		case runnerDirHasAny(sourceDir, "go.mod", "main.go"):
			return "Go"
		case runnerDirHasAny(sourceDir, "requirements.txt", "pyproject.toml", "app.py", "manage.py"):
			return "Python"
		case runnerDirHasAny(sourceDir, "composer.json", "artisan", "public/index.php"):
			return "PHP"
		case runnerDirHasAny(sourceDir, ".output/server/index.mjs", ".next", "package.json", "server.js"):
			return "Node"
		case runnerDirHasAny(sourceDir, "dist/index.html", "index.html"):
			return "Static"
		}
	}
	baseImage := strings.ToLower(strings.TrimSpace(rc.BaseImage))
	startCmd := strings.ToLower(strings.TrimSpace(rc.StartCommand))
	buildCmd := strings.ToLower(strings.TrimSpace(rc.BuildCommand))
	switch {
	case strings.Contains(baseImage, "golang"), strings.Contains(buildCmd, "go build"), strings.Contains(startCmd, "./app"):
		return "Go"
	case strings.Contains(baseImage, "python"), strings.Contains(buildCmd, "pip "), strings.Contains(startCmd, "python "), strings.Contains(startCmd, "gunicorn"), strings.Contains(startCmd, "uvicorn"):
		return "Python"
	case strings.Contains(baseImage, "php"), strings.Contains(buildCmd, "composer "), strings.Contains(startCmd, "php "):
		return "PHP"
	case strings.Contains(baseImage, "node"), strings.Contains(buildCmd, "npm"), strings.Contains(buildCmd, "pnpm"), strings.Contains(buildCmd, "yarn"), strings.Contains(startCmd, "node "):
		return "Node"
	}
	return "Custom"
}

func runnerDirHasAny(dir string, names ...string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func normalizeRunnerExtraNetworks(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildRunnerScript(rc runnerConfig, sourceDir string) string {
	mode := strings.ToLower(strings.TrimSpace(rc.Mode))
	wd := strings.TrimSpace(rc.WorkingDir)
	if wd == "" {
		wd = "/var/www/app"
	}
	srcDir := strings.TrimSpace(sourceDir)
	if srcDir == "" {
		srcDir = runnerWorkspaceMountPath
	}
	startCmd := strings.TrimSpace(rc.StartCommand)
	installCmd := strings.TrimSpace(rc.BuildCommand)

	var b strings.Builder
	b.WriteString("set -e\n")
	b.WriteString("mkdir -p \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("\"\n")
	b.WriteString("if [ -d \"")
	b.WriteString(strings.ReplaceAll(srcDir, "\"", "\\\""))
	b.WriteString("\" ]; then\n")
	b.WriteString("  echo \"[RUNNER] syncing source into working dir\"\n")
	b.WriteString("  if command -v tar >/dev/null 2>&1; then\n")
	b.WriteString("    if tar --help 2>/dev/null | grep -q -- '--exclude'; then\n")
	b.WriteString("      echo \"[RUNNER] sync strategy: tar --exclude (skip node_modules/.git/.gopanel_artifact)\"\n")
	b.WriteString("      (cd \"")
	b.WriteString(strings.ReplaceAll(srcDir, "\"", "\\\""))
	b.WriteString("\" && tar --exclude='./node_modules' --exclude='./.git' --exclude='./.gopanel_artifact' --exclude='./__MACOSX' -cf - .) | (cd \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("\" && tar xpf -)\n")
	b.WriteString("    else\n")
	b.WriteString("      echo \"[RUNNER] sync strategy: tar fallback + cleanup transient dirs\"\n")
	b.WriteString("      (cd \"")
	b.WriteString(strings.ReplaceAll(srcDir, "\"", "\\\""))
	b.WriteString("\" && tar cf - .) | (cd \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("\" && tar xpf -)\n")
	b.WriteString("      rm -rf \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("/node_modules\" \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("/.git\" \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("/.gopanel_artifact\" \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("/__MACOSX\" 2>/dev/null || true\n")
	b.WriteString("    fi\n")
	b.WriteString("  else\n")
	b.WriteString("    echo \"[RUNNER] sync strategy: cp -a fallback + cleanup transient dirs\"\n")
	b.WriteString("    cp -a \"")
	b.WriteString(strings.ReplaceAll(srcDir, "\"", "\\\""))
	b.WriteString("\"/. \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("\"/\n")
	b.WriteString("    rm -rf \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("/node_modules\" \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("/.git\" \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("/.gopanel_artifact\" \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("/__MACOSX\" 2>/dev/null || true\n")
	b.WriteString("  fi\n")
	b.WriteString("fi\n")
	b.WriteString("cd \"")
	b.WriteString(strings.ReplaceAll(wd, "\"", "\\\""))
	b.WriteString("\"\n")
	if strings.TrimSpace(rc.PreStart) != "" {
		b.WriteString(rc.PreStart)
		if !strings.HasSuffix(rc.PreStart, "\n") {
			b.WriteString("\n")
		}
	}
	if mode == "build_run" {
		if installCmd != "" {
			b.WriteString("echo \"[BUILD+RUN] executing custom build command\"\n")
			b.WriteString(installCmd)
			b.WriteString("\n")
		} else {
			b.WriteString("if [ -f package.json ]; then\n")
			b.WriteString("  echo \"[BUILD+RUN] package.json detected, rebuilding app\"\n")
			b.WriteString("  if [ -f pnpm-lock.yaml ]; then pnpm install --frozen-lockfile; ")
			b.WriteString("elif [ -f yarn.lock ]; then yarn install --frozen-lockfile; ")
			b.WriteString("elif [ -f package-lock.json ]; then npm ci; else npm install; fi\n")
			b.WriteString("  npm run build\n")
			b.WriteString("elif [ -f .output/server/index.mjs ]; then\n")
			b.WriteString("  echo \"[RUN] detected standalone .output, start directly\"\n")
			b.WriteString("else\n")
			b.WriteString("  echo \"[BUILD+RUN] buildCommand 为空，且未识别到默认 Node 构建特征（package.json / .output）\" >&2\n")
			b.WriteString("  exit 1\n")
			b.WriteString("fi\n")
		}
	}
	if startCmd != "" {
		b.WriteString("exec ")
		b.WriteString(startCmd)
		b.WriteString("\n")
	} else {
		b.WriteString("echo \"[RUN] start command empty, skip auto start\"\n")
	}
	return b.String()
}
