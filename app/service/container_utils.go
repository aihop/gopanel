package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
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
)

func composeAvailable() bool {
	if _, err := exec.LookPath("podman"); err == nil {
		return true
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return true
	}
	if _, err := exec.LookPath("podman-compose"); err == nil {
		return true
	}
	return false
}

func hasDockerSockPathSetting() bool {
	var settingItem model.Setting
	if err := global.DB.Where("key = ?", "DockerSockPath").First(&settingItem).Error; err != nil {
		return false
	}
	return strings.TrimSpace(settingItem.Value) != ""
}

func isPodmanInstalled() bool {
	_, err := exec.LookPath("podman")
	return err == nil
}

func splitNonEmptyLines(s string) []string {
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func parsePercent(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parsePairToMB(s string) (float64, float64, bool) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0, false
	}
	a, ok1 := parseSizeToMB(parts[0])
	b, ok2 := parseSizeToMB(parts[1])
	return a, b, ok1 && ok2
}

func parseMemUsagePairToMB(s string) (float64, bool) {
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return 0, false
	}
	a, ok := parseSizeToMB(parts[0])
	return a, ok
}

func parseSizeToMB(s string) (float64, bool) {
	b, ok := parseSizeToBytes(s)
	if !ok {
		return 0, false
	}
	return b / 1024.0 / 1024.0, true
}

func parseSizeToBytes(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.ReplaceAll(s, "I", "")
	s = strings.ReplaceAll(s, "B", "")
	s = strings.TrimSpace(s)
	unit := ""
	switch {
	case strings.HasSuffix(s, "K"):
		unit = "K"
		s = strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "M"):
		unit = "M"
		s = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "G"):
		unit = "G"
		s = strings.TrimSuffix(s, "G")
	case strings.HasSuffix(s, "T"):
		unit = "T"
		s = strings.TrimSuffix(s, "T")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	switch unit {
	case "K":
		v *= 1024
	case "M":
		v *= 1024 * 1024
	case "G":
		v *= 1024 * 1024 * 1024
	case "T":
		v *= 1024 * 1024 * 1024 * 1024
	}
	return v, true
}

func stringsToMap(list []string) map[string]string {
	var labelMap = make(map[string]string)
	for _, label := range list {
		if strings.Contains(label, "=") {
			sps := strings.SplitN(label, "=", 2)
			labelMap[sps[0]] = sps[1]
		}
	}
	return labelMap
}

func calculateCPUPercentUnix(stats *container.Stats) float64 {
	cpuPercent := 0.0
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * 100.0
		if len(stats.CPUStats.CPUUsage.PercpuUsage) != 0 {
			cpuPercent = cpuPercent * float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
	}
	return cpuPercent
}

func calculateMemPercentUnix(memStats container.MemoryStats) float64 {
	memPercent := 0.0
	memUsage := calculateMemUsageUnixNoCache(memStats)
	memLimit := memStats.Limit
	if memUsage > 0.0 && memLimit > 0.0 {
		memPercent = (float64(memUsage) / float64(memLimit)) * 100.0
	}
	return memPercent
}

func calculateMemUsageUnixNoCache(mem container.MemoryStats) uint64 {
	if v, isCgroup1 := mem.Stats["total_inactive_file"]; isCgroup1 && v < mem.Usage {
		return mem.Usage - v
	}
	if v := mem.Stats["inactive_file"]; v < mem.Usage {
		return mem.Usage - v
	}
	return mem.Usage
}

func calculateBlockIO(blkio container.BlkioStats) (blkRead float64, blkWrite float64) {
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		switch strings.ToLower(bioEntry.Op) {
		case "read":
			blkRead = (blkRead + float64(bioEntry.Value)) / 1024 / 1024
		case "write":
			blkWrite = (blkWrite + float64(bioEntry.Value)) / 1024 / 1024
		}
	}
	return
}
func calculateNetwork(network map[string]container.NetworkStats) (float64, float64) {
	var rx, tx float64

	for _, v := range network {
		rx += float64(v.RxBytes) / 1024
		tx += float64(v.TxBytes) / 1024
	}
	return rx, tx
}

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
				authConfig := registry.AuthConfig{
					Username: repo.Username,
					Password: repo.Password,
				}
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

func loadCpuAndMem(client *client.Client, containerStr string) dto.ContainerListStats {
	data := dto.ContainerListStats{
		ContainerID: containerStr,
	}
	res, err := client.ContainerStats(context.Background(), containerStr, false)
	if err != nil {
		return data
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return data
	}
	var stats *container.Stats
	if err := json.Unmarshal(body, &stats); err != nil {
		return data
	}

	data.CPUTotalUsage = stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage
	data.SystemUsage = stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage
	data.CPUPercent = calculateCPUPercentUnix(stats)
	data.PercpuUsage = len(stats.CPUStats.CPUUsage.PercpuUsage)

	data.MemoryCache = stats.MemoryStats.Stats["cache"]
	data.MemoryUsage = calculateMemUsageUnixNoCache(stats.MemoryStats)
	data.MemoryLimit = stats.MemoryStats.Limit

	data.MemoryPercent = calculateMemPercentUnix(stats.MemoryStats)
	return data
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
			networkConf.EndpointsConfig = map[string]*network.EndpointSettings{
				req.Network: {
					IPAMConfig: &network.EndpointIPAMConfig{
						IPv4Address: req.Ipv4,
						IPv6Address: req.Ipv6,
					},
				}}
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
			hostConf.Mounts = append(hostConf.Mounts, mount.Mount{
				Type:     mount.Type(volume.Type),
				Source:   volume.SourceDir,
				Target:   volume.ContainerDir,
				ReadOnly: volume.Mode == "ro",
			})
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
	// cgroup v2 runtimes such as crun reject swappiness during container create.
	hostConf.MemorySwappiness = nil
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

func loadVolumeBinds(binds []types.MountPoint) []dto.VolumeHelper {
	var datas []dto.VolumeHelper
	for _, bind := range binds {
		var volumeItem dto.VolumeHelper
		volumeItem.Type = string(bind.Type)
		if bind.Type == "volume" {
			volumeItem.SourceDir = bind.Name
		} else {
			volumeItem.SourceDir = bind.Source
		}
		volumeItem.ContainerDir = bind.Destination
		volumeItem.Mode = "ro"
		if bind.RW {
			volumeItem.Mode = "rw"
		}
		datas = append(datas, volumeItem)
	}
	return datas
}

func loadPortByInspect(id string, client *client.Client) ([]types.Port, error) {
	container, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil, err
	}
	var itemPorts []types.Port
	portBindings := container.NetworkSettings.Ports
	if len(portBindings) == 0 && container.ContainerJSONBase != nil && container.ContainerJSONBase.HostConfig != nil {
		portBindings = container.ContainerJSONBase.HostConfig.PortBindings
	}
	for key, val := range portBindings {
		if !strings.Contains(string(key), "/") {
			continue
		}
		item := strings.Split(string(key), "/")
		itemPort, _ := strconv.ParseUint(item[0], 10, 16)

		for _, itemVal := range val {
			publicPort, _ := strconv.ParseUint(itemVal.HostPort, 10, 16)
			itemPorts = append(itemPorts, types.Port{
				PrivatePort: uint16(itemPort),
				Type:        item[1],
				PublicPort:  uint16(publicPort),
				IP:          itemVal.HostIP,
			})
		}
	}
	return itemPorts, nil
}

func loadContainerPortForInfo(itemPorts []types.Port) []dto.PortHelper {
	var exposedPorts []dto.PortHelper
	samePortMap := make(map[string]dto.PortHelper)
	ports := transPortToStr(itemPorts)
	for _, item := range ports {
		itemStr := strings.Split(item, "->")
		if len(itemStr) < 2 {
			continue
		}
		var itemPort dto.PortHelper
		lastIndex := strings.LastIndex(itemStr[0], ":")
		if lastIndex == -1 {
			itemPort.HostPort = itemStr[0]
		} else {
			itemPort.HostIP = itemStr[0][0:lastIndex]
			itemPort.HostPort = itemStr[0][lastIndex+1:]
		}
		itemContainer := strings.Split(itemStr[1], "/")
		if len(itemContainer) != 2 {
			continue
		}
		itemPort.ContainerPort = itemContainer[0]
		itemPort.Protocol = itemContainer[1]
		keyItem := fmt.Sprintf("%s->%s/%s", itemPort.HostPort, itemPort.ContainerPort, itemPort.Protocol)
		if val, ok := samePortMap[keyItem]; ok {
			val.HostIP = ""
			samePortMap[keyItem] = val
		} else {
			samePortMap[keyItem] = itemPort
		}
	}
	for _, val := range samePortMap {
		exposedPorts = append(exposedPorts, val)
	}
	return exposedPorts
}

func transPortToStr(ports []types.Port) []string {
	var (
		ipv4Ports []types.Port
		ipv6Ports []types.Port
	)
	for _, port := range ports {
		if strings.Contains(port.IP, ":") {
			ipv6Ports = append(ipv6Ports, port)
		} else {
			ipv4Ports = append(ipv4Ports, port)
		}
	}
	list1 := simplifyPort(ipv4Ports)
	list2 := simplifyPort(ipv6Ports)
	return append(list1, list2...)
}
func simplifyPort(ports []types.Port) []string {
	var datas []string
	if len(ports) == 0 {
		return datas
	}
	if len(ports) == 1 {
		ip := ""
		if len(ports[0].IP) != 0 {
			ip = ports[0].IP + ":"
		}
		itemPortStr := fmt.Sprintf("%s%v/%s", ip, ports[0].PrivatePort, ports[0].Type)
		if ports[0].PublicPort != 0 {
			itemPortStr = fmt.Sprintf("%s%v->%v/%s", ip, ports[0].PublicPort, ports[0].PrivatePort, ports[0].Type)
		}
		datas = append(datas, itemPortStr)
		return datas
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].PrivatePort < ports[j].PrivatePort
	})
	start := ports[0]

	for i := 1; i < len(ports); i++ {
		if ports[i].PrivatePort != ports[i-1].PrivatePort+1 || ports[i].IP != ports[i-1].IP || ports[i].PublicPort != ports[i-1].PublicPort+1 || ports[i].Type != ports[i-1].Type {
			if ports[i-1].PrivatePort == start.PrivatePort {
				itemPortStr := fmt.Sprintf("%s:%v/%s", start.IP, start.PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v->%v/%s", start.IP, start.PublicPort, start.PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			} else {
				itemPortStr := fmt.Sprintf("%s:%v-%v/%s", start.IP, start.PrivatePort, ports[i-1].PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v-%v->%v-%v/%s", start.IP, start.PublicPort, ports[i-1].PublicPort, start.PrivatePort, ports[i-1].PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			}
			start = ports[i]
		}
		if i == len(ports)-1 {
			if ports[i].PrivatePort == start.PrivatePort {
				itemPortStr := fmt.Sprintf("%s:%v/%s", start.IP, start.PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v->%v/%s", start.IP, start.PublicPort, start.PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			} else {
				itemPortStr := fmt.Sprintf("%s:%v-%v/%s", start.IP, start.PrivatePort, ports[i].PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v-%v->%v-%v/%s", start.IP, start.PublicPort, ports[i].PublicPort, start.PrivatePort, ports[i].PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			}
		}
	}
	return datas
}

func normalizeContainerIPList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		ip := strings.TrimSpace(item)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func inferContainerRuntimeMode(runtimeKind string, runtimeHost string) string {
	host := strings.TrimSpace(runtimeHost)
	if runtime.GOOS != "linux" {
		return "default"
	}
	if host != "" && strings.Contains(host, "/run/user/") {
		return "rootless"
	}
	switch strings.TrimSpace(runtimeKind) {
	case string(docker.RuntimeDocker), string(docker.RuntimePodman):
		return "rootful"
	default:
		return "default"
	}
}

func inspectContainerIPsForList(ctx context.Context, dockerClient *client.Client, containerID string, runtimeHost string) []string {
	if dockerClient != nil {
		inspect, err := dockerClient.ContainerInspect(ctx, containerID)
		if err == nil && inspect.NetworkSettings != nil {
			networks := make([]string, 0, len(inspect.NetworkSettings.Networks))
			for _, endpoint := range inspect.NetworkSettings.Networks {
				if endpoint == nil {
					continue
				}
				if ip := strings.TrimSpace(endpoint.IPAddress); ip != "" {
					networks = append(networks, ip)
				}
				if ip := strings.TrimSpace(endpoint.GlobalIPv6Address); ip != "" {
					networks = append(networks, ip)
				}
			}
			return normalizeContainerIPList(networks)
		}
	}
	if strings.TrimSpace(runtimeHost) == "" {
		return nil
	}
	raw, err := inspectPodman(&dto.InspectReq{ID: containerID, Type: "container", RuntimeHost: runtimeHost})
	if err != nil {
		return nil
	}
	return extractContainerIPsFromInspectJSON(raw)
}

func extractContainerIPsFromInspectJSON(raw string) []string {
	type endpointView struct {
		IPAddress         string `json:"IPAddress"`
		GlobalIPv6Address string `json:"GlobalIPv6Address"`
	}
	type networkSettingsView struct {
		Networks map[string]endpointView `json:"Networks"`
	}
	type inspectView struct {
		NetworkSettings networkSettingsView `json:"NetworkSettings"`
	}
	parse := func(items []inspectView) []string {
		if len(items) == 0 {
			return nil
		}
		networks := make([]string, 0)
		for _, item := range items {
			for _, endpoint := range item.NetworkSettings.Networks {
				if ip := strings.TrimSpace(endpoint.IPAddress); ip != "" {
					networks = append(networks, ip)
				}
				if ip := strings.TrimSpace(endpoint.GlobalIPv6Address); ip != "" {
					networks = append(networks, ip)
				}
			}
		}
		return normalizeContainerIPList(networks)
	}

	var list []inspectView
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return parse(list)
	}

	var single inspectView
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return parse([]inspectView{single})
	}
	return nil
}

func inspectContainerRunUserForList(ctx context.Context, dockerClient *client.Client, containerID string, runtimeHost string) string {
	if dockerClient != nil {
		inspect, err := dockerClient.ContainerInspect(ctx, containerID)
		if err == nil && inspect.Config != nil {
			return strings.TrimSpace(inspect.Config.User)
		}
	}
	if strings.TrimSpace(runtimeHost) == "" {
		return ""
	}
	raw, err := inspectPodman(&dto.InspectReq{ID: containerID, Type: "container", RuntimeHost: runtimeHost})
	if err != nil {
		return ""
	}
	return extractContainerUserFromInspectJSON(raw)
}

func extractContainerUserFromInspectJSON(raw string) string {
	type configView struct {
		User string `json:"User"`
	}
	type inspectView struct {
		Config configView `json:"Config"`
	}
	parse := func(items []inspectView) string {
		for _, item := range items {
			if v := strings.TrimSpace(item.Config.User); v != "" {
				return v
			}
		}
		return ""
	}

	var list []inspectView
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return parse(list)
	}

	var single inspectView
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return parse([]inspectView{single})
	}
	return ""
}

func normalizeContainerLabelFilter(filter string, isPodman bool) string {
	f := strings.TrimSpace(filter)
	if !isPodman || f == "" {
		return f
	}
	f = strings.ReplaceAll(f, composeProjectLabel, podmanComposeProjectLabel)
	f = strings.ReplaceAll(f, composeConfigLabel, podmanComposeConfigLabel)
	f = strings.ReplaceAll(f, composeWorkdirLabel, podmanComposeWorkdirLabel)
	return f
}

func containerPrimaryName(item types.Container) string {
	if len(item.Names) == 0 {
		return ""
	}
	return strings.TrimPrefix(item.Names[0], "/")
}

func toStringSlice(v interface{}) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []interface{}:
		var out []string
		for _, it := range x {
			s := strings.TrimSpace(fmt.Sprint(it))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		return []string{s}
	default:
		return nil
	}
}

func toBool(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		b, _ := strconv.ParseBool(strings.TrimSpace(x))
		return b
	default:
		return false
	}
}

func toInt64(v interface{}) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	default:
		return 0
	}
}
