package service

import (
	"context"
	"fmt"
	"io"
	"strings"

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
		RestartPolicy: container.RestartPolicy{Name: "always"},
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
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		if spec.PreviousContainerID != "" && strings.Contains(err.Error(), "port is already allocated") {
			logEngineProgress(progress, "检测到固定端口冲突，正在停止旧容器以释放端口...")
			_ = cli.ContainerStop(ctx, spec.PreviousContainerID, container.StopOptions{})
			if retryErr := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); retryErr != nil {
				return runtimeContainerResult{}, fmt.Errorf("停止旧容器后启动新容器仍失败: %w", retryErr)
			}
			logEngineProgress(progress, "已成功抢占固定端口并启动新容器")
		} else {
			return runtimeContainerResult{}, fmt.Errorf("failed to start engine container: %w", err)
		}
	}

	stopStreaming := startEngineContainerLogStreaming(ctx, cli, resp.ID, progress)
	defer stopStreaming()
	logEngineProgress(progress, "正在等待容器端口绑定: %s/tcp", containerPort)
	bindings, err := waitForEnginePortBinding(ctx, cli, resp.ID, containerPort)
	if err != nil {
		return runtimeContainerResult{}, err
	}
	var hostPort int
	_, _ = fmt.Sscanf(bindings[0].HostPort, "%d", &hostPort)
	return runtimeContainerResult{HostPort: hostPort, ContainerID: resp.ID}, nil
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
