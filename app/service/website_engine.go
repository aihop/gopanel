package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Removed DetectEngineEnv as we are shifting to pure containerized pipelines.

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

	if req.CodeSource == "git" || req.CodeSource == "pipeline" {
		imageName = req.GitRepo
	} else {
		return 0, "", "", errors.New("unsupported container deployment source: " + req.CodeSource)
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

	// 容器内部监听端口只认镜像自身元数据，不接受外部 Proxy、流水线端口或旧容器配置干预。
	containerPort = detectEngineContainerPort(imageInspect)

	workingDir := detectEngineWorkingDir(imageInspect)
	logEngineProgress(progress, "镜像运行配置: workingDir=%s, containerPort=%s", workingDir, containerPort)

	containerName := fmt.Sprintf("gopanel-engine-%s-%d", alias, time.Now().Unix())

	// 选择代码目录
	codeDir := req.CodeDir
	if codeDir == "" {
		codeDir = filepath.Join(global.CONF.System.BaseDir, "wwwroot", alias)
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "always"},
		PortBindings: nat.PortMap{
			nat.Port(containerPort + "/tcp"): []nat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: "0",
				},
			},
		},
	}
	selectedCodeDir := ""
	if req.CodeSource == "pipeline" {
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

	config := &container.Config{
		Image:        imageName,
		Env:          envs,
		Cmd:          cmd,
		WorkingDir:   workingDir,
		ExposedPorts: imageInspect.Config.ExposedPorts, // 默认使用新镜像自带的所有暴露端口
	}

	// 确保探测到的主端口一定被包含在内
	if config.ExposedPorts == nil {
		config.ExposedPorts = make(nat.PortSet)
	}
	config.ExposedPorts[nat.Port(containerPort+"/tcp")] = struct{}{}

	// 完整继承旧容器的配置 (如果存在)
	if req.CodeSource == "pipeline" && req.PreviousContainerID != "" {
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
							HostPort: "0",
						},
					},
				}
			}
			logEngineProgress(progress, "深度继承旧容器(%s)的完整运行参数和配置", req.PreviousContainerID)
		}
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return 0, "", "", fmt.Errorf("failed to create engine container: %w", err)
	}
	logEngineProgress(progress, "已创建容器: %s", containerName)

	logEngineProgress(progress, "正在启动容器...")
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// 容错处理：即使我们改用了 HostPort: "0"，但在某些极端情况下（比如旧容器强行绑定了某个系统级独占资源，或是 IP 被占用），
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
