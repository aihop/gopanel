package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"io"
	"strconv"
	"strings"
)

func checkImageExist(client *client.Client, imageItem string) bool {
	images, err := client.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		return false
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageItem || tag == imageItem+":latest" {
				return true
			}
		}
	}
	return false
}
func pullImages(ctx context.Context, client *client.Client, imageName string) error {
	options := image.PullOptions{}
	repos, _ := repo.NewIImageRepoRepo().List()
	if len(repos) != 0 {
		for _, repo := range repos {
			if strings.HasPrefix(imageName, repo.DownloadUrl) && repo.Auth {
				authConfig := registry.AuthConfig{Username: repo.Username, Password: repo.Password}
				encodedJSON, err := json.Marshal(authConfig)
				if err != nil {
					return err
				}
				authStr := base64.URLEncoding.EncodeToString(encodedJSON)
				options.RegistryAuth = authStr
			}
		}
	} else {
		hasAuth, authStr := loadAuthInfo(imageName)
		if hasAuth {
			options.RegistryAuth = authStr
		}
	}
	out, err := client.ImagePull(ctx, imageName, options)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(io.Discard, out)
	if err != nil {
		return err
	}
	return nil
}
func checkPortStats(ports []dto.PortHelper) (nat.PortMap, error) {
	portMap := make(nat.PortMap)
	if len(ports) == 0 {
		return portMap, nil
	}
	for _, port := range ports {
		if strings.Contains(port.ContainerPort, "-") {
			if !strings.Contains(port.HostPort, "-") {
				return portMap, buserr.New(constant.ErrPortRules)
			}
			hostStart, _ := strconv.Atoi(strings.Split(port.HostPort, "-")[0])
			hostEnd, _ := strconv.Atoi(strings.Split(port.HostPort, "-")[1])
			containerStart, _ := strconv.Atoi(strings.Split(port.ContainerPort, "-")[0])
			containerEnd, _ := strconv.Atoi(strings.Split(port.ContainerPort, "-")[1])
			if (hostEnd-hostStart) <= 0 || (containerEnd-containerStart) <= 0 {
				return portMap, buserr.New(constant.ErrPortRules)
			}
			if (containerEnd - containerStart) != (hostEnd - hostStart) {
				return portMap, buserr.New(constant.ErrPortRules)
			}
			for i := 0; i <= hostEnd-hostStart; i++ {
				bindItem := nat.PortBinding{HostPort: strconv.Itoa(hostStart + i), HostIP: port.HostIP}
				portMap[nat.Port(fmt.Sprintf("%d/%s", containerStart+i, port.Protocol))] = []nat.PortBinding{bindItem}
			}
			for i := hostStart; i <= hostEnd; i++ {
				if common.ScanPort(i) {
					return portMap, buserr.WithDetail(constant.ErrPortInUsed, i, nil)
				}
			}
		} else {
			portItem := 0
			if strings.Contains(port.HostPort, "-") {
				portItem, _ = strconv.Atoi(strings.Split(port.HostPort, "-")[0])
			} else {
				portItem, _ = strconv.Atoi(port.HostPort)
			}
			if common.ScanPort(portItem) {
				return portMap, buserr.WithDetail(constant.ErrPortInUsed, portItem, nil)
			}
			bindItem := nat.PortBinding{HostPort: strconv.Itoa(portItem), HostIP: port.HostIP}
			portMap[nat.Port(fmt.Sprintf("%s/%s", port.ContainerPort, port.Protocol))] = []nat.PortBinding{bindItem}
		}
	}
	return portMap, nil
}
func loadConfigInfo(isCreate bool, req *dto.ContainerOperate, oldContainer *types.ContainerJSON) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	var config container.Config
	var hostConf container.HostConfig
	if !isCreate {
		config = *oldContainer.Config
		hostConf = *oldContainer.HostConfig
		normalizeHostConfigForRuntime(&hostConf)
	}
	var networkConf network.NetworkingConfig
	portMap, err := checkPortStats(req.ExposedPorts)
	if err != nil {
		return nil, nil, nil, err
	}
	exposed := make(nat.PortSet)
	for port := range portMap {
		exposed[port] = struct{}{}
	}
	config.Image = req.Image
	config.Cmd = req.Cmd
	config.Entrypoint = req.Entrypoint
	config.Env = req.Env
	config.Labels = stringsToMap(req.Labels)
	config.ExposedPorts = exposed
	config.OpenStdin = req.OpenStdin
	config.Tty = req.Tty
	if len(req.Network) != 0 {
		switch req.Network {
		case "host", "none", "bridge":
			hostConf.NetworkMode = container.NetworkMode(req.Network)
		}
		if req.Ipv4 != "" || req.Ipv6 != "" {
			networkConf.EndpointsConfig = map[string]*network.EndpointSettings{req.Network: {IPAMConfig: &network.EndpointIPAMConfig{IPv4Address: req.Ipv4, IPv6Address: req.Ipv6}}}
		} else {
			networkConf.EndpointsConfig = map[string]*network.EndpointSettings{req.Network: {}}
		}
	} else {
		if req.Ipv4 != "" || req.Ipv6 != "" {
			return nil, nil, nil, fmt.Errorf("please set up the network")
		}
		networkConf = network.NetworkingConfig{}
	}
	hostConf.Privileged = req.Privileged
	hostConf.AutoRemove = req.AutoRemove
	hostConf.CPUShares = req.CPUShares
	hostConf.PublishAllPorts = req.PublishAllPorts
	hostConf.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(req.RestartPolicy)}
	if req.RestartPolicy == "on-failure" {
		hostConf.RestartPolicy.MaximumRetryCount = 5
	}
	hostConf.NanoCPUs = int64(req.NanoCPUs * 1000000000)
	hostConf.Memory = int64(req.Memory * 1024 * 1024)
	hostConf.MemorySwap = 0
	hostConf.PortBindings = portMap
	hostConf.Binds = []string{}
	hostConf.Mounts = []mount.Mount{}
	config.Volumes = make(map[string]struct{})
	for _, volume := range req.Volumes {
		if volume.Type == "volume" {
			hostConf.Mounts = append(hostConf.Mounts, mount.Mount{Type: mount.Type(volume.Type), Source: volume.SourceDir, Target: volume.ContainerDir, ReadOnly: volume.Mode == "ro"})
			config.Volumes[volume.ContainerDir] = struct{}{}
		} else {
			hostConf.Binds = append(hostConf.Binds, fmt.Sprintf("%s:%s:%s", volume.SourceDir, volume.ContainerDir, volume.Mode))
		}
	}
	docker.EnsureContainerLogConfig(&hostConf)
	return &config, &hostConf, &networkConf, nil
}
func normalizeHostConfigForRuntime(hostConf *container.HostConfig) {
	if hostConf == nil {
		return
	}
	hostConf.MemorySwappiness = nil
	// Podman's specgen is stricter than Docker: empty mode strings
	// cause "invalid option type" errors. Set sensible defaults.
	if hostConf.CgroupnsMode == "" {
		hostConf.CgroupnsMode = "private"
	}
	if hostConf.IpcMode == "" {
		hostConf.IpcMode = "private"
	}
	if hostConf.UTSMode == "" {
		hostConf.UTSMode = "host"
	}
	if hostConf.RestartPolicy.Name == "" {
		// 默认 always：整机重启后配合已启用的 podman-restart.service 自动恢复容器。
		// 仅当未显式指定策略时生效；用户在 UI 里显式选择 no 会原样保留。
		hostConf.RestartPolicy.Name = container.RestartPolicyMode("always")
	}
}
func reCreateAfterUpdate(name string, client *client.Client, config *container.Config, hostConf *container.HostConfig, networkConf *types.NetworkSettings) {
	ctx := context.Background()
	normalizeHostConfigForRuntime(hostConf)
	docker.EnsureContainerLogConfig(hostConf)
	var oldNetworkConf network.NetworkingConfig
	if networkConf != nil {
		for networkKey := range networkConf.Networks {
			oldNetworkConf.EndpointsConfig = map[string]*network.EndpointSettings{networkKey: {}}
			break
		}
	}
	oldContainer, err := client.ContainerCreate(ctx, config, hostConf, &oldNetworkConf, &v1.Platform{}, name)
	if err != nil {
		global.LOG.Errorf("recreate after container update failed, err: %v", err)
		return
	}
	if err := client.ContainerStart(ctx, oldContainer.ID, container.StartOptions{}); err != nil {
		global.LOG.Errorf("restart after container update failed, err: %v", err)
	}
	global.LOG.Info("recreate after container update successful")
}
