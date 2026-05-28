package service

import (
	"context"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"runtime"
	"strings"
)

func (u *ContainerService) ContainerCreate(req *dto.ContainerOperate) error {
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	ctx := context.Background()
	newContainer, _ := client.ContainerInspect(ctx, req.Name)
	if newContainer.ContainerJSONBase != nil {
		return buserr.New(constant.ErrContainerName)
	}
	if !checkImageExist(client, req.Image) || req.ForcePull {
		if err := pullImages(ctx, client, req.Image); err != nil {
			if !req.ForcePull {
				return err
			}
			global.LOG.Errorf("force pull image %s failed, err: %v", req.Image, err)
		}
	}
	imageInfo, _, err := client.ImageInspectWithRaw(ctx, req.Image)
	if err != nil {
		return err
	}
	if len(req.Entrypoint) == 0 {
		req.Entrypoint = imageInfo.Config.Entrypoint
	}
	if len(req.Cmd) == 0 {
		req.Cmd = imageInfo.Config.Cmd
	}
	sanitizeStaticIPForContainerNetwork(ctx, client, req)
	config, hostConf, networkConf, err := loadConfigInfo(true, req, nil)
	if err != nil {
		return err
	}
	global.LOG.Infof("new container info %s has been made, now start to create", req.Name)
	con, err := client.ContainerCreate(ctx, config, hostConf, networkConf, &v1.Platform{}, req.Name)
	if err != nil {
		if retriedCon, retryErr, handled := retryCreateWithoutStaticIPOnSubnetError(ctx, client, req.Name, config, hostConf, networkConf, err); handled {
			con = retriedCon
			err = retryErr
		}
	}
	if err != nil {
		_ = client.ContainerRemove(ctx, req.Name, container.RemoveOptions{RemoveVolumes: true, Force: true})
		return err
	}
	global.LOG.Infof("create container %s successful! now check if the container is started and delete the container information if it is not.", req.Name)
	if err := client.ContainerStart(ctx, con.ID, container.StartOptions{}); err != nil {
		if retryErr, handled := retryStartWithoutStaticIPOnSubnetError(ctx, client, req.Name, con.ID, config, hostConf, networkConf, err); handled {
			return retryErr
		}
		_ = client.ContainerRemove(ctx, req.Name, container.RemoveOptions{RemoveVolumes: true, Force: true})
		return fmt.Errorf("create successful but start failed, err: %v", err)
	}
	invalidateContainerListCaches()
	return nil
}
func (u *ContainerService) ContainerInfo(req *dto.OperationWithName) (*dto.ContainerOperate, error) {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		return containerInfoPodman(req)
	}
	ctx := context.Background()
	isPodman := docker.IsPodmanRuntime(ctx)
	host := strings.TrimSpace(req.RuntimeHost)
	if host == "" && isPodman {
		host, _ = resolveLinuxPodmanContainerHost(ctx, req.Name)
	}
	if isPodman && (host == "podman-cli" || host == "") {
		req2 := &dto.OperationWithName{Name: req.Name, RuntimeHost: host}
		return containerInfoPodman(req2)
	}
	var cli *client.Client
	var err error
	if strings.HasPrefix(host, "unix://") {
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
	} else {
		cli, err = docker.NewDockerClient()
	}
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	oldContainer, err := cli.ContainerInspect(ctx, req.Name)
	if err != nil {
		if isPodman {
			req2 := &dto.OperationWithName{Name: req.Name, RuntimeHost: host}
			return containerInfoPodman(req2)
		}
		return nil, err
	}
	var data dto.ContainerOperate
	data.ContainerID = oldContainer.ID
	data.Name = strings.ReplaceAll(oldContainer.Name, "/", "")
	data.Image = oldContainer.Config.Image
	if oldContainer.NetworkSettings != nil {
		for network := range oldContainer.NetworkSettings.Networks {
			data.Network = network
			break
		}
	}
	exposePorts, _ := loadPortByInspect(oldContainer.ID, cli)
	data.ExposedPorts = loadContainerPortForInfo(exposePorts)
	if oldContainer.NetworkSettings != nil && data.Network != "" {
		if bridgeNetworkSettings, ok := oldContainer.NetworkSettings.Networks[data.Network]; ok && bridgeNetworkSettings != nil {
			if bridgeNetworkSettings.IPAMConfig != nil {
				ipv4Address := bridgeNetworkSettings.IPAMConfig.IPv4Address
				data.Ipv4 = ipv4Address
				ipv6Address := bridgeNetworkSettings.IPAMConfig.IPv6Address
				data.Ipv6 = ipv6Address
			} else {
				data.Ipv4 = bridgeNetworkSettings.IPAddress
			}
		}
	}
	data.Cmd = oldContainer.Config.Cmd
	data.OpenStdin = oldContainer.Config.OpenStdin
	data.Tty = oldContainer.Config.Tty
	data.Entrypoint = oldContainer.Config.Entrypoint
	data.Env = oldContainer.Config.Env
	data.CPUShares = oldContainer.HostConfig.CPUShares
	for key, val := range oldContainer.Config.Labels {
		data.Labels = append(data.Labels, fmt.Sprintf("%s=%s", key, val))
	}
	data.AutoRemove = oldContainer.HostConfig.AutoRemove
	data.Privileged = oldContainer.HostConfig.Privileged
	data.PublishAllPorts = oldContainer.HostConfig.PublishAllPorts
	data.RestartPolicy = string(oldContainer.HostConfig.RestartPolicy.Name)
	if oldContainer.HostConfig.NanoCPUs != 0 {
		data.NanoCPUs = float64(oldContainer.HostConfig.NanoCPUs) / 1000000000
	}
	if oldContainer.HostConfig.Memory != 0 {
		data.Memory = float64(oldContainer.HostConfig.Memory) / 1024 / 1024
	}
	data.Volumes = loadVolumeBinds(oldContainer.Mounts)
	return &data, nil
}
func (u *ContainerService) ContainerUpdate(req *dto.ContainerOperate) error {
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	ctx := context.Background()
	newContainer, _ := client.ContainerInspect(ctx, req.Name)
	if newContainer.ContainerJSONBase != nil && newContainer.ID != req.ContainerID {
		return buserr.New(constant.ErrContainerName)
	}
	oldContainer, err := client.ContainerInspect(ctx, req.ContainerID)
	if err != nil {
		return err
	}
	if !checkImageExist(client, req.Image) || req.ForcePull {
		if err := pullImages(ctx, client, req.Image); err != nil {
			if !req.ForcePull {
				return err
			}
			return fmt.Errorf("pull image %s failed, err: %v", req.Image, err)
		}
	}
	if err := client.ContainerRemove(ctx, req.ContainerID, container.RemoveOptions{Force: true}); err != nil {
		return err
	}
	sanitizeStaticIPForContainerNetwork(ctx, client, req)
	config, hostConf, networkConf, err := loadConfigInfo(false, req, &oldContainer)
	if err != nil {
		reCreateAfterUpdate(req.Name, client, oldContainer.Config, oldContainer.HostConfig, oldContainer.NetworkSettings)
		return err
	}
	global.LOG.Infof("new container info %s has been update, now start to recreate", req.Name)
	con, err := client.ContainerCreate(ctx, config, hostConf, networkConf, &v1.Platform{}, req.Name)
	if err != nil {
		if retriedCon, retryErr, handled := retryCreateWithoutStaticIPOnSubnetError(ctx, client, req.Name, config, hostConf, networkConf, err); handled {
			con = retriedCon
			err = retryErr
		}
	}
	if err != nil {
		reCreateAfterUpdate(req.Name, client, oldContainer.Config, oldContainer.HostConfig, oldContainer.NetworkSettings)
		return fmt.Errorf("update container failed, err: %v", err)
	}
	global.LOG.Infof("update container %s successful! now check if the container is started.", req.Name)
	if err := client.ContainerStart(ctx, con.ID, container.StartOptions{}); err != nil {
		if retryErr, handled := retryStartWithoutStaticIPOnSubnetError(ctx, client, req.Name, con.ID, config, hostConf, networkConf, err); handled {
			return retryErr
		}
		return fmt.Errorf("update successful but start failed, err: %v", err)
	}
	invalidateContainerListCaches()
	return nil
}
func (u *ContainerService) ContainerUpgrade(req *dto.ContainerUpgrade) error {
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	ctx := context.Background()
	oldContainer, err := client.ContainerInspect(ctx, req.Name)
	if err != nil {
		return err
	}
	if !checkImageExist(client, req.Image) || req.ForcePull {
		if err := pullImages(ctx, client, req.Image); err != nil {
			if !req.ForcePull {
				return err
			}
			return fmt.Errorf("pull image %s failed, err: %v", req.Image, err)
		}
	}
	config := oldContainer.Config
	config.Image = req.Image
	hostConf := oldContainer.HostConfig
	var networkConf network.NetworkingConfig
	if oldContainer.NetworkSettings != nil {
		for networkKey := range oldContainer.NetworkSettings.Networks {
			networkConf.EndpointsConfig = map[string]*network.EndpointSettings{networkKey: {}}
			break
		}
	}
	if err := client.ContainerRemove(ctx, req.Name, container.RemoveOptions{Force: true}); err != nil {
		return err
	}
	global.LOG.Infof("new container info %s has been update, now start to recreate", req.Name)
	con, err := client.ContainerCreate(ctx, config, hostConf, &networkConf, &v1.Platform{}, req.Name)
	if err != nil {
		reCreateAfterUpdate(req.Name, client, oldContainer.Config, oldContainer.HostConfig, oldContainer.NetworkSettings)
		return fmt.Errorf("upgrade container failed, err: %v", err)
	}
	global.LOG.Infof("upgrade container %s successful! now check if the container is started.", req.Name)
	if err := client.ContainerStart(ctx, con.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("upgrade successful but start failed, err: %v", err)
	}
	invalidateContainerListCaches()
	return nil
}
func (u *ContainerService) ContainerRename(req *dto.ContainerRename) error {
	ctx := context.Background()
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	newContainer, _ := client.ContainerInspect(ctx, req.NewName)
	if newContainer.ContainerJSONBase != nil {
		return buserr.New(constant.ErrContainerName)
	}
	if err := client.ContainerRename(ctx, req.Name, req.NewName); err != nil {
		return err
	}
	invalidateContainerListCaches()
	return nil
}
func (u *ContainerService) ContainerCommit(req *dto.ContainerCommit) error {
	ctx := context.Background()
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	options := container.CommitOptions{Reference: req.NewImageName, Comment: req.Comment, Author: req.Author, Changes: nil, Pause: req.Pause, Config: nil}
	_, err = client.ContainerCommit(ctx, req.ContainerId, options)
	if err != nil {
		return fmt.Errorf("failed to commit container, err: %v", err)
	}
	return nil
}
func (u *ContainerService) ContainerOperation(req *dto.ContainerOperation) error {
	ctx := context.Background()
	runtimeHost := strings.TrimSpace(req.RuntimeHost)
	if docker.IsPodmanRuntime(ctx) {
		if runtimeHost != "" {
			return containerOperationPodmanCLI(ctx, runtimeHost, req.Names, req.Operation)
		}
		idToHost, nameToHost, err := podmanLinuxContainerHostMaps(ctx)
		if err != nil {
			return containerOperationPodmanCLI(ctx, "", req.Names, req.Operation)
		}
		group := make(map[string][]string)
		for _, n := range req.Names {
			key := strings.TrimSpace(n)
			if key == "" {
				continue
			}
			host := strings.TrimSpace(idToHost[key])
			if host == "" {
				host = strings.TrimSpace(nameToHost[key])
			}
			if host == "" {
				host = "podman-cli"
			}
			group[host] = append(group[host], key)
		}
		for host, items := range group {
			if err := containerOperationPodmanCLI(ctx, host, items, req.Operation); err != nil {
				return err
			}
		}
		return nil
	}
	var cli *client.Client
	var err error
	if strings.HasPrefix(runtimeHost, "unix://") {
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithHost(runtimeHost), client.WithAPIVersionNegotiation())
	} else {
		cli, err = docker.NewDockerClient()
	}
	if err != nil {
		return err
	}
	defer cli.Close()
	for _, item := range req.Names {
		global.LOG.Infof("start container %s operation %s", item, req.Operation)
		switch req.Operation {
		case constant.ContainerOpStart:
			err = cli.ContainerStart(ctx, item, container.StartOptions{})
		case constant.ContainerOpStop:
			err = cli.ContainerStop(ctx, item, container.StopOptions{})
		case constant.ContainerOpRestart:
			err = cli.ContainerRestart(ctx, item, container.StopOptions{})
		case constant.ContainerOpKill:
			err = cli.ContainerKill(ctx, item, "SIGKILL")
		case constant.ContainerOpPause:
			err = cli.ContainerPause(ctx, item)
		case constant.ContainerOpUnpause:
			err = cli.ContainerUnpause(ctx, item)
		case constant.ContainerOpRemove:
			err = cli.ContainerRemove(ctx, item, container.RemoveOptions{RemoveVolumes: true, Force: true})
		}
		if err != nil {
			return err
		}
	}
	invalidateContainerListCaches()
	return nil
}
