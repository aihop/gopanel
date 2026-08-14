package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

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

type runtimeContainerSpec struct {
	Alias               string
	Image               string
	ContainerPort       string
	PublishedHostPort   string
	WorkingDir          string
	User                string
	Env                 []string
	Cmd                 []string
	Binds               []string
	Mounts              []mount.Mount
	NetworkMode         container.NetworkMode
	ExtraNetworks       []string
	PreviousContainerID string
	WaitForReady        bool
	// ReadyTimeout 等待容器就绪的预算。0 表示用默认值。
	// build_run 模式要在容器里装依赖 + 完整构建，慢机器上远超普通启动，
	// 由调用方按模式给足；给不够会把「还在构建」误判成「构建失败」。
	ReadyTimeout time.Duration
}

type runtimeContainerResult struct {
	HostPort    int
	ContainerID string
}

type runtimeContainerSpecBuilder func(image.InspectResponse) (runtimeContainerSpec, error)

func deployRuntimeContainer(ctx context.Context, imageName string, buildSpec runtimeContainerSpecBuilder, progress func(format string, a ...interface{})) (runtimeContainerResult, error) {
	cli, err := docker.NewDockerClient()
	if err != nil {
		return runtimeContainerResult{}, fmt.Errorf("failed to init docker client: %w", err)
	}
	if cli == nil {
		return runtimeContainerResult{}, nil
	}
	defer cli.Close()

	imageName = strings.TrimSpace(imageName)
	if imageName == "" {
		return runtimeContainerResult{}, fmt.Errorf("container image is empty")
	}
	if err := ensureRuntimeImage(ctx, cli, imageName, progress); err != nil {
		return runtimeContainerResult{}, err
	}
	imageInspect, err := cli.ImageInspect(ctx, imageName)
	if err != nil {
		return runtimeContainerResult{}, fmt.Errorf("failed to inspect image metadata %s: %w", imageName, err)
	}
	spec, err := buildSpec(imageInspect)
	if err != nil {
		return runtimeContainerResult{}, err
	}
	spec.Image = imageName
	return startRuntimeContainer(ctx, cli, imageInspect, spec, progress)
}

func ensureRuntimeImage(ctx context.Context, cli *dockerclient.Client, imageName string, progress func(format string, a ...interface{})) error {
	if _, _, err := cli.ImageInspectWithRaw(ctx, imageName); err == nil {
		global.LOG.Infof("Using local engine image: %s", imageName)
		logEngineProgress(progress, "正在使用本地镜像: %s", imageName)
		return nil
	} else if !dockerclient.IsErrNotFound(err) {
		return fmt.Errorf("failed to inspect image %s: %w", imageName, err)
	}

	global.LOG.Infof("Local image not found, pulling engine image: %s", imageName)
	logEngineProgress(progress, "本地未找到镜像，正在拉取: %s", imageName)
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func startRuntimeContainer(ctx context.Context, cli *dockerclient.Client, imageInspect image.InspectResponse, spec runtimeContainerSpec, progress func(format string, a ...interface{})) (runtimeContainerResult, error) {
	containerPort := strings.TrimSpace(spec.ContainerPort)
	if containerPort == "" {
		return runtimeContainerResult{}, fmt.Errorf("container port is empty")
	}
	publishedHostPort := strings.TrimSpace(spec.PublishedHostPort)
	if publishedHostPort == "" {
		publishedHostPort = "0"
	}
	if publishedHostPort == "0" {
		logEngineProgress(progress, "发布端口策略: 自动分配宿主机端口")
	} else {
		logEngineProgress(progress, "发布端口策略: 固定宿主机端口 %s", publishedHostPort)
		if strings.TrimSpace(spec.PreviousContainerID) != "" {
			logEngineProgress(progress, "检测到固定端口模式且存在旧容器，新容器需要等待旧容器释放端口后才能完成切换，期间可能有短暂中断")
		}
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: initialRuntimeRestartPolicy(spec.WaitForReady),
		PortBindings: nat.PortMap{
			nat.Port(containerPort + "/tcp"): []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: publishedHostPort}},
		},
		Binds:       spec.Binds,
		Mounts:      spec.Mounts,
		NetworkMode: spec.NetworkMode,
	}
	docker.EnsureContainerLogConfig(hostConfig)

	exposedPorts := cloneRuntimeExposedPorts(imageInspect)
	exposedPorts[nat.Port(containerPort+"/tcp")] = struct{}{}
	config := &container.Config{
		Image:        spec.Image,
		Env:          spec.Env,
		Cmd:          spec.Cmd,
		WorkingDir:   spec.WorkingDir,
		User:         spec.User,
		ExposedPorts: exposedPorts,
	}
	containerName := fmt.Sprintf("%s-%s", spec.Alias, random.RandString(4))
	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return runtimeContainerResult{}, fmt.Errorf("failed to create engine container: %w", err)
	}
	logEngineProgress(progress, "已创建容器: %s", containerName)

	for _, networkName := range spec.ExtraNetworks {
		if _, err := cli.NetworkInspect(ctx, networkName, network.InspectOptions{}); err != nil {
			_ = cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			return runtimeContainerResult{}, fmt.Errorf("container extra network %s 不存在或不可用: %w", networkName, err)
		}
		if err := cli.NetworkConnect(ctx, networkName, resp.ID, &network.EndpointSettings{}); err != nil {
			_ = cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			return runtimeContainerResult{}, fmt.Errorf("container 接入额外网络 %s 失败: %w", networkName, err)
		}
		logEngineProgress(progress, "已接入额外网络: %s", networkName)
	}

	logEngineProgress(progress, "正在启动容器...")
	previousContainerStopped := false
	restorePreviousContainer := func(startErr error) error {
		_ = cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
		if !previousContainerStopped {
			return startErr
		}
		if restoreErr := cli.ContainerStart(context.Background(), spec.PreviousContainerID, container.StartOptions{}); restoreErr != nil {
			return fmt.Errorf("%w；恢复旧容器失败: %v", startErr, restoreErr)
		}
		logEngineProgress(progress, "新容器启动失败，已恢复旧容器")
		return startErr
	}
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		if spec.PreviousContainerID != "" && strings.Contains(err.Error(), "port is already allocated") {
			logEngineProgress(progress, "检测到固定端口冲突，正在停止旧容器以释放端口...")
			if stopErr := cli.ContainerStop(ctx, spec.PreviousContainerID, container.StopOptions{}); stopErr != nil {
				return runtimeContainerResult{}, restorePreviousContainer(fmt.Errorf("停止旧容器释放固定端口失败: %w", stopErr))
			}
			previousContainerStopped = true
			if retryErr := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); retryErr != nil {
				return runtimeContainerResult{}, restorePreviousContainer(fmt.Errorf("停止旧容器后启动新容器仍失败: %w", retryErr))
			}
			logEngineProgress(progress, "已成功抢占固定端口并启动新容器")
		} else {
			return runtimeContainerResult{}, restorePreviousContainer(fmt.Errorf("failed to start engine container: %w", err))
		}
	}

	stopStreaming := startEngineContainerLogStreaming(ctx, cli, resp.ID, progress)
	defer stopStreaming()
	logEngineProgress(progress, "正在等待容器端口绑定: %s/tcp", containerPort)
	bindings, err := waitForEnginePortBinding(ctx, cli, resp.ID, containerPort)
	if err != nil {
		return runtimeContainerResult{}, restorePreviousContainer(err)
	}
	var hostPort int
	_, _ = fmt.Sscanf(bindings[0].HostPort, "%d", &hostPort)
	if spec.WaitForReady {
		readyTimeout := resolveRuntimeReadyTimeout(spec.ReadyTimeout)
		logEngineProgress(progress, "端口已绑定，正在等待 Runner 构建完成并开始监听: 127.0.0.1:%d（最长等待 %s）", hostPort, readyTimeout)
		if err := waitForRuntimeContainerReady(ctx, cli, resp.ID, containerPort, hostPort, readyTimeout); err != nil {
			return runtimeContainerResult{}, restorePreviousContainer(err)
		}
		if _, err := cli.ContainerUpdate(ctx, resp.ID, readyRuntimeUpdateConfig()); err != nil {
			return runtimeContainerResult{}, restorePreviousContainer(fmt.Errorf("Runner 就绪后启用自动重启失败: %w", err))
		}
		logEngineProgress(progress, "Runner 服务已就绪: 127.0.0.1:%d", hostPort)
		logEngineProgress(progress, "Runner 已启用运行期自动重启策略")
	}
	return runtimeContainerResult{HostPort: hostPort, ContainerID: resp.ID}, nil
}

func initialRuntimeRestartPolicy(waitForReady bool) container.RestartPolicy {
	if waitForReady {
		return container.RestartPolicy{Name: "no"}
	}
	return container.RestartPolicy{Name: "always"}
}

func readyRuntimeUpdateConfig() container.UpdateConfig {
	return container.UpdateConfig{RestartPolicy: container.RestartPolicy{Name: "always"}}
}

const (
	defaultRuntimeReadyTimeout = 15 * time.Minute
	// 运维逃生口：机器特别慢（低配 ARM + swap）时不必改代码重装面板。
	runtimeReadyTimeoutEnv = "GOPANEL_RUNNER_READY_TIMEOUT"
)

func resolveRuntimeReadyTimeout(specTimeout time.Duration) time.Duration {
	if raw := strings.TrimSpace(os.Getenv(runtimeReadyTimeoutEnv)); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			return parsed
		}
		if global.LOG != nil {
			global.LOG.Warnf("ignored invalid %s=%q, expected a duration like 45m", runtimeReadyTimeoutEnv, raw)
		}
	}
	if specTimeout > 0 {
		return specTimeout
	}
	return defaultRuntimeReadyTimeout
}

func waitForRuntimeContainerReady(ctx context.Context, cli *dockerclient.Client, containerID, containerPort string, hostPort int, readyTimeout time.Duration) error {
	if hostPort <= 0 {
		return fmt.Errorf("runner host port is invalid: %d", hostPort)
	}
	if readyTimeout <= 0 {
		readyTimeout = defaultRuntimeReadyTimeout
	}
	readyCtx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	address := fmt.Sprintf("127.0.0.1:%d", hostPort)
	var lastDialErr error
	for {
		inspect, err := cli.ContainerInspect(readyCtx, containerID)
		if err != nil {
			return fmt.Errorf("failed to inspect runner container readiness: %w", err)
		}
		if inspect.ContainerJSONBase == nil || inspect.ContainerJSONBase.State == nil || !inspect.ContainerJSONBase.State.Running || inspect.ContainerJSONBase.State.Restarting {
			return buildRunnerNotReadyDiagError(readyCtx, cli, containerID, containerPort, hostPort,
				"Runner 在构建/启动完成前退出", inspect)
		}
		if inspect.RestartCount > 0 {
			return buildRunnerNotReadyDiagError(readyCtx, cli, containerID, containerPort, hostPort,
				"Runner 在构建/启动完成前被重启（构建会从头重跑，常见原因是内存不足或启动命令崩溃）", inspect)
		}
		requestCtx, requestCancel := context.WithTimeout(readyCtx, 3*time.Second)
		checkErr := checkContainerWebsiteEndpoint(requestCtx, "http", address)
		requestCancel()
		if checkErr == nil {
			return nil
		}
		lastDialErr = checkErr
		select {
		case <-readyCtx.Done():
			return buildRuntimeContainerReadinessError(context.Background(), cli, containerID, address, readyTimeout, lastDialErr)
		case <-ticker.C:
		}
	}
}

func buildRuntimeContainerReadinessError(ctx context.Context, cli *dockerclient.Client, containerID, address string, readyTimeout time.Duration, lastErr error) error {
	// 把预算和调大的办法写进报错：超时最常见的原因是机器慢、构建还没跑完，
	// 而不是构建失败——不写清楚，用户会去查一个并不存在的构建错误。
	detail := fmt.Sprintf("Runner 在 %s 内未就绪 (http://%s/)，若构建本身仍在进行可用 %s 调大预算",
		readyTimeout, address, runtimeReadyTimeoutEnv)
	if lastErr != nil {
		detail += fmt.Sprintf(": %v", lastErr)
	}
	rawLogs := readEngineContainerLogTail(ctx, cli, containerID)
	if hint := detectEngineContainerKillHint(nil, rawLogs); hint != "" {
		detail += fmt.Sprintf(", %s", hint)
	}
	if logs := formatEngineContainerLogs(rawLogs); logs != "" {
		detail += fmt.Sprintf(", logs=%s", logs)
	}
	return errors.New(detail)
}

func cloneRuntimeExposedPorts(imageInspect image.InspectResponse) nat.PortSet {
	ports := make(nat.PortSet)
	if imageInspect.Config == nil {
		return ports
	}
	for port := range imageInspect.Config.ExposedPorts {
		ports[port] = struct{}{}
	}
	return ports
}
