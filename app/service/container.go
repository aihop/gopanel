package service

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type ContainerService struct {
}

type IContainerService interface {
	Page(req *dto.PageContainer) (int64, interface{}, error)
	List() ([]string, error)
	PageNetwork(req *dto.SearchWithPage) (int64, interface{}, error)
	ListNetwork() ([]dto.Options, error)
	PageVolume(req *dto.SearchWithPage) (int64, interface{}, error)
	ListVolume() ([]dto.Options, error)
	PageCompose(req *dto.SearchWithPage) (int64, interface{}, error)
	CreateCompose(req *dto.ComposeCreate) (string, error)
	ComposeOperation(req *dto.ComposeOperation) error
	ContainerCreate(req *dto.ContainerOperate) error
	ContainerUpdate(req *dto.ContainerOperate) error
	ContainerUpgrade(req *dto.ContainerUpgrade) error
	ContainerInfo(req *dto.OperationWithName) (*dto.ContainerOperate, error)
	ContainerListStats() ([]dto.ContainerListStats, error)
	LoadResourceLimit() (*dto.ResourceLimit, error)
	ContainerRename(req *dto.ContainerRename) error
	ContainerCommit(req *dto.ContainerCommit) error
	ContainerLogClean(req *dto.OperationWithName) error
	ContainerOperation(req *dto.ContainerOperation) error
	ContainerLogs(wsConn *websocket.Conn, containerType, container, since, tail, runtimeHost string, follow bool) error
	DownloadContainerLogs(containerType, container, since, tail, runtimeHost string) (string, error)
	ContainerStats(id string) (*dto.ContainerStats, error)
	Inspect(req *dto.InspectReq) (string, error)
	DeleteNetwork(req *dto.BatchDelete) error
	CreateNetwork(req *dto.NetworkCreate) error
	DeleteVolume(req *dto.BatchDelete) error
	CreateVolume(req *dto.VolumeCreate) error
	TestCompose(req *dto.ComposeCreate) (bool, error)
	ComposeUpdate(req *dto.ComposeUpdate) error
	Prune(req *dto.ContainerPrune) (dto.ContainerPruneReport, error)

	LoadContainerLogs(req *dto.OperationWithNameAndType) string
}

func NewIContainerService() IContainerService {
	return &ContainerService{}
}

func (u *ContainerService) Page(req *dto.PageContainer) (int64, interface{}, error) {
	var (
		records []types.Container
		list    []types.Container
	)
	ctx := context.Background()
	isPodman := docker.IsPodmanRuntime(ctx)
	options := container.ListOptions{
		All: true,
	}
	if len(req.Filters) != 0 && !isPodman {
		options.Filters = filters.NewArgs()
		options.Filters.Add("label", normalizeContainerLabelFilter(req.Filters, isPodman))
	}
	var containers []types.Container
	var sourceByID map[string]string
	if isPodman {
		list2, source2, err := listContainersMergedByHostWithSource(ctx, options)
		if err != nil {
			return 0, nil, err
		}
		containers = list2
		sourceByID = source2
	} else {
		client, err := docker.NewDockerClient()
		if err != nil {
			return 0, nil, err
		}
		defer client.Close()
		containers, err = client.ContainerList(ctx, options)
		if err != nil {
			return 0, nil, err
		}
	}
	if req.ExcludeAppStore {
		for _, item := range containers {
			if created, ok := item.Labels[composeCreatedBy]; ok && created == "Apps" {
				continue
			}
			list = append(list, item)
		}
	} else {
		list = containers
	}
	if isPodman && strings.TrimSpace(req.Filters) != "" {
		k, v, ok := splitLabelFilter(req.Filters)
		if ok {
			k = normalizeContainerLabelFilter(k, true)
			var filtered []types.Container
			for _, item := range list {
				if item.Labels == nil {
					continue
				}
				if lv, ok := item.Labels[k]; ok && (v == "" || lv == v) {
					filtered = append(filtered, item)
				}
			}
			list = filtered
		}
	}

	if len(req.Name) != 0 {
		length, count := len(list), 0
		for count < length {
			if !strings.Contains(containerPrimaryName(list[count]), req.Name) {
				list = append(list[:count], list[(count+1):]...)
				length--
			} else {
				count++
			}
		}
	}
	if req.State != "all" {
		length, count := len(list), 0
		for count < length {
			if list[count].State != req.State {
				list = append(list[:count], list[(count+1):]...)
				length--
			} else {
				count++
			}
		}
	}
	switch req.OrderBy {
	case "name":
		sort.Slice(list, func(i, j int) bool {
			if req.Order == constant.OrderAsc {
				return containerPrimaryName(list[i]) < containerPrimaryName(list[j])
			}
			return containerPrimaryName(list[i]) > containerPrimaryName(list[j])
		})
	case "state":
		sort.Slice(list, func(i, j int) bool {
			if req.Order == constant.OrderAsc {
				return list[i].State < list[j].State
			}
			return list[i].State > list[j].State
		})
	default:
		sort.Slice(list, func(i, j int) bool {
			if req.Order == constant.OrderAsc {
				return list[i].Created < list[j].Created
			}
			return list[i].Created > list[j].Created
		})
	}

	total, start, end := len(list), (req.Page-1)*req.Limit, req.Page*req.Limit
	if start > total {
		records = make([]types.Container, 0)
	} else {
		if end >= total {
			end = total
		}
		records = list[start:end]
	}

	backDatas := make([]dto.ContainerInfo, len(records))
	for i := 0; i < len(records); i++ {
		item := records[i]
		IsFromCompose := false
		if _, ok := firstLabel(item.Labels, composeProjectLabel, podmanComposeProjectLabel); ok {
			IsFromCompose = true
		}
		IsFromApp := false
		if created, ok := item.Labels[composeCreatedBy]; ok && created == "Apps" {
			IsFromApp = true
		}

		exposePorts := transPortToStr(records[i].Ports)
		name := ""
		if len(item.Names) > 0 {
			name = strings.TrimPrefix(item.Names[0], "/")
		}
		imageID := item.ImageID
		if parts := strings.Split(imageID, ":"); len(parts) > 1 {
			imageID = parts[len(parts)-1]
		}
		info := dto.ContainerInfo{
			ContainerID:   item.ID,
			CreateTime:    time.Unix(item.Created, 0).Format(constant.DateTimeLayout),
			Name:          name,
			ImageId:       imageID,
			ImageName:     item.Image,
			State:         item.State,
			RunTime:       item.Status,
			Ports:         exposePorts,
			IsFromApp:     IsFromApp,
			IsFromCompose: IsFromCompose,
		}
		if isPodman && sourceByID != nil {
			info.RuntimeHost = strings.TrimSpace(sourceByID[item.ID])
		}
		appInstallRepo := repo.NewIAppInstallRepo()
		websiteRepo := repo.NewWebsite()
		install, _ := appInstallRepo.GetFirst(appInstallRepo.WithContainerName(info.Name))
		if install.ID > 0 {
			info.AppInstallName = install.Name
			info.AppName = "namemem"
			websites, _ := websiteRepo.GetBy(websiteRepo.WithAppInstallId(install.ID))
			for _, website := range websites {
				info.Websites = append(info.Websites, website.PrimaryDomain)
			}
		}
		backDatas[i] = info
		if item.NetworkSettings != nil && len(item.NetworkSettings.Networks) > 0 {
			networks := make([]string, 0, len(item.NetworkSettings.Networks))
			for key := range item.NetworkSettings.Networks {
				networks = append(networks, item.NetworkSettings.Networks[key].IPAddress)
			}
			sort.Strings(networks)
			backDatas[i].Network = networks
		}
	}

	return int64(total), backDatas, nil
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

func (u *ContainerService) List() ([]string, error) {
	ctx := context.Background()
	var containers []types.Container
	if docker.IsPodmanRuntime(ctx) {
		list2, err := listContainersMergedByHost(ctx, container.ListOptions{All: true})
		if err != nil {
			return nil, err
		}
		containers = list2
	} else {
		client, err := docker.NewDockerClient()
		if err != nil {
			return nil, err
		}
		defer client.Close()
		list2, err := client.ContainerList(ctx, container.ListOptions{All: true})
		if err != nil {
			return nil, err
		}
		containers = list2
	}
	var datas []string
	for _, container := range containers {
		for _, name := range container.Names {
			if len(name) != 0 {
				datas = append(datas, strings.TrimPrefix(name, "/"))
			}
		}
	}

	return datas, nil
}

func (u *ContainerService) ContainerListStats() ([]dto.ContainerListStats, error) {
	ctx := context.Background()
	if docker.IsPodmanRuntime(ctx) {
		list, source, err := listContainersMergedByHostWithSource(ctx, container.ListOptions{All: true})
		if err != nil {
			return nil, err
		}
		datas := make([]dto.ContainerListStats, len(list))
		var wg sync.WaitGroup
		wg.Add(len(list))
		for i := 0; i < len(list); i++ {
			go func(index int, item types.Container) {
				host := strings.TrimSpace(source[item.ID])
				if host == "" || host == "podman-cli" {
					datas[index] = loadContainerListStatPodmanCLI(item.ID)
					wg.Done()
					return
				}
				if !strings.HasPrefix(host, "unix://") {
					wg.Done()
					return
				}
				cli, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
				if err != nil {
					wg.Done()
					return
				}
				datas[index] = loadCpuAndMem(cli, item.ID)
				_ = cli.Close()
				wg.Done()
			}(i, list[i])
		}
		wg.Wait()
		return datas, nil
	}

	client, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	list, err := client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	datas := make([]dto.ContainerListStats, len(list))
	var wg sync.WaitGroup
	wg.Add(len(list))
	for i := 0; i < len(list); i++ {
		go func(index int, item types.Container) {
			datas[index] = loadCpuAndMem(client, item.ID)
			wg.Done()
		}(i, list[i])
	}
	wg.Wait()
	return datas, nil
}

func loadContainerListStatPodmanCLI(containerID string) dto.ContainerListStats {
	c := exec.Command("podman", "stats", "--no-stream", "--format", "{{.ID}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}", containerID)
	out, err := c.CombinedOutput()
	if err != nil {
		return dto.ContainerListStats{ContainerID: containerID}
	}
	lines := splitNonEmptyLines(string(out))
	if len(lines) == 0 {
		return dto.ContainerListStats{ContainerID: containerID}
	}
	parts := strings.Split(lines[0], "\t")
	if len(parts) < 4 {
		return dto.ContainerListStats{ContainerID: containerID}
	}
	memUsage, _ := parseMemUsagePairToMB(parts[2])
	return dto.ContainerListStats{
		ContainerID:   strings.TrimSpace(containerID),
		CPUPercent:    parsePercent(parts[1]),
		MemoryPercent: parsePercent(parts[3]),
		MemoryUsage:   uint64(memUsage * 1024 * 1024),
	}
}

func podmanLinuxContainerHostMaps(ctx context.Context) (map[string]string, map[string]string, error) {
	list, source, err := listContainersMergedByHostWithSource(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, nil, err
	}
	idToHost := make(map[string]string)
	nameToHost := make(map[string]string)
	for _, it := range list {
		host := strings.TrimSpace(source[it.ID])
		if host != "" {
			idToHost[it.ID] = host
		}
		for _, n := range it.Names {
			name := strings.TrimPrefix(strings.TrimSpace(n), "/")
			if name == "" {
				continue
			}
			if _, ok := nameToHost[name]; ok {
				continue
			}
			nameToHost[name] = host
		}
	}
	return idToHost, nameToHost, nil
}

func resolveLinuxPodmanContainerHost(ctx context.Context, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil
	}
	idToHost, nameToHost, err := podmanLinuxContainerHostMaps(ctx)
	if err != nil {
		return "", err
	}
	if host := strings.TrimSpace(idToHost[key]); host != "" {
		return host, nil
	}
	if host := strings.TrimSpace(nameToHost[key]); host != "" {
		return host, nil
	}
	return "", nil
}

func (u *ContainerService) Inspect(req *dto.InspectReq) (string, error) {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		return inspectPodman(req)
	}
	ctx := context.Background()
	isPodman := docker.IsPodmanRuntime(ctx)

	host := strings.TrimSpace(req.RuntimeHost)
	if host == "" && isPodman {
		if req.Type == "container" {
			host, _ = resolveLinuxPodmanContainerHost(ctx, req.ID)
		} else if req.Type == "image" {
			_, imgSource, _, err := listImagesMergedByHostWithSource(ctx)
			if err == nil {
				host = strings.TrimSpace(imgSource[req.ID])
			}
		}
	}

	if isPodman && (host == "podman-cli" || host == "") {
		return inspectPodman(req)
	}

	var cli *client.Client
	var err error
	if strings.HasPrefix(host, "unix://") {
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
	} else {
		cli, err = docker.NewDockerClient()
	}
	if err != nil {
		return "", err
	}
	defer cli.Close()
	var inspectInfo interface{}
	switch req.Type {
	case "container":
		inspectInfo, err = cli.ContainerInspect(ctx, req.ID)
	case "image":
		inspectInfo, _, err = cli.ImageInspectWithRaw(ctx, req.ID)
	case "network":
		inspectInfo, err = cli.NetworkInspect(ctx, req.ID, network.InspectOptions{})
	case "volume":
		inspectInfo, err = cli.VolumeInspect(ctx, req.ID)
	}
	if err != nil {
		if isPodman && strings.HasPrefix(host, "unix://") {
			if out, podErr := inspectPodman(&dto.InspectReq{ID: req.ID, Type: req.Type, RuntimeHost: host}); podErr == nil {
				return out, nil
			}
		}
		return "", err
	}
	bytes, err := json.Marshal(inspectInfo)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (u *ContainerService) Prune(req *dto.ContainerPrune) (dto.ContainerPruneReport, error) {
	report := dto.ContainerPruneReport{}
	if docker.IsPodmanRuntime(context.Background()) {
		return u.prunePodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return report, err
	}
	defer client.Close()
	pruneFilters := filters.NewArgs()
	if req.WithTagAll {
		if req.PruneType == "image" {
			// dangling 过滤器 仅对镜像生效
			pruneFilters.Add("dangling", "false")
		}
		if req.PruneType != "image" {
			pruneFilters.Add("until", "24h")
		}
	}
	switch req.PruneType {
	case "container":
		rep, err := client.ContainersPrune(context.Background(), pruneFilters)
		if err != nil {
			return report, err
		}
		report.DeletedNumber = len(rep.ContainersDeleted)
		report.SpaceReclaimed = int(rep.SpaceReclaimed)
	case "image":
		rep, err := client.ImagesPrune(context.Background(), pruneFilters)
		if err != nil {
			return report, err
		}
		report.DeletedNumber = len(rep.ImagesDeleted)
		report.SpaceReclaimed = int(rep.SpaceReclaimed)
	case "network":
		rep, err := client.NetworksPrune(context.Background(), pruneFilters)
		if err != nil {
			return report, err
		}
		report.DeletedNumber = len(rep.NetworksDeleted)
	case "volume":
		versions, err := client.ServerVersion(context.Background())
		if err != nil {
			return report, err
		}
		if common.ComparePanelVersion(versions.APIVersion, "1.42") {
			pruneFilters.Add("all", "true")
		}
		rep, err := client.VolumesPrune(context.Background(), pruneFilters)
		if err != nil {
			return report, err
		}
		report.DeletedNumber = len(rep.VolumesDeleted)
		report.SpaceReclaimed = int(rep.SpaceReclaimed)
	case "buildcache":
		opts := types.BuildCachePruneOptions{}
		opts.All = true
		rep, err := client.BuildCachePrune(context.Background(), opts)
		if err != nil {
			return report, err
		}
		report.DeletedNumber = len(rep.CachesDeleted)
		report.SpaceReclaimed = int(rep.SpaceReclaimed)
	}
	return report, nil
}

func (u *ContainerService) LoadResourceLimit() (*dto.ResourceLimit, error) {
	cpuCounts, err := cpu.Counts(true)
	if err != nil {
		return nil, fmt.Errorf("load cpu limit failed, err: %v", err)
	}
	memoryInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("load memory limit failed, err: %v", err)
	}

	data := dto.ResourceLimit{
		CPU:    cpuCounts,
		Memory: memoryInfo.Total,
	}
	return &data, nil
}

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

func containerInfoPodman(req *dto.OperationWithName) (*dto.ContainerOperate, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return nil, err
	}
	args := make([]string, 0, 8)
	if host := strings.TrimSpace(req.RuntimeHost); strings.HasPrefix(host, "unix://") {
		args = append(args, "--url", host)
	}
	args = append(args, "container", "inspect", req.Name, "--format", "json")
	out, err := exec.Command("podman", args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return nil, err
		}
		return nil, errors.New(msg)
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("container not found")
	}
	m := items[0]

	var data dto.ContainerOperate
	data.ContainerID = strings.TrimSpace(fmt.Sprint(m["Id"]))
	if data.ContainerID == "" {
		data.ContainerID = strings.TrimSpace(fmt.Sprint(m["ID"]))
	}
	data.Name = strings.TrimPrefix(strings.TrimSpace(fmt.Sprint(m["Name"])), "/")
	if data.Name == "" {
		data.Name = strings.TrimSpace(req.Name)
	}

	if cfg, ok := m["Config"].(map[string]interface{}); ok {
		data.Image = strings.TrimSpace(fmt.Sprint(cfg["Image"]))
		data.Cmd = toStringSlice(cfg["Cmd"])
		data.Entrypoint = toStringSlice(cfg["Entrypoint"])
		data.Env = toStringSlice(cfg["Env"])
		data.OpenStdin = toBool(cfg["OpenStdin"])
		data.Tty = toBool(cfg["Tty"])
		if labels, ok := cfg["Labels"].(map[string]interface{}); ok {
			for k, v := range labels {
				data.Labels = append(data.Labels, fmt.Sprintf("%s=%v", k, v))
			}
		}
	}

	if hostCfg, ok := m["HostConfig"].(map[string]interface{}); ok {
		data.AutoRemove = toBool(hostCfg["AutoRemove"])
		data.Privileged = toBool(hostCfg["Privileged"])
		data.PublishAllPorts = toBool(hostCfg["PublishAllPorts"])
		if rp, ok := hostCfg["RestartPolicy"].(map[string]interface{}); ok {
			data.RestartPolicy = strings.TrimSpace(fmt.Sprint(rp["Name"]))
		} else {
			data.RestartPolicy = strings.TrimSpace(fmt.Sprint(hostCfg["RestartPolicy"]))
		}
		if n := toInt64(hostCfg["NanoCpus"]); n != 0 {
			data.NanoCPUs = float64(n) / 1000000000
		}
		if n := toInt64(hostCfg["NanoCPUs"]); n != 0 {
			data.NanoCPUs = float64(n) / 1000000000
		}
		data.CPUShares = toInt64(hostCfg["CpuShares"])
		if data.CPUShares == 0 {
			data.CPUShares = toInt64(hostCfg["CPUShares"])
		}
		if mem := toInt64(hostCfg["Memory"]); mem != 0 {
			data.Memory = float64(mem) / 1024 / 1024
		}
	}

	if net, ok := m["NetworkSettings"].(map[string]interface{}); ok {
		if networks, ok := net["Networks"].(map[string]interface{}); ok {
			for name, v := range networks {
				data.Network = name
				if nm, ok := v.(map[string]interface{}); ok {
					data.Ipv4 = strings.TrimSpace(fmt.Sprint(nm["IPAddress"]))
					data.Ipv6 = strings.TrimSpace(fmt.Sprint(nm["GlobalIPv6Address"]))
				}
				break
			}
		}
	}

	data.ExposedPorts = podmanPortsForInfo(req.Name, req.RuntimeHost)
	data.Volumes = podmanMountsForInfo(m)
	return &data, nil
}

func podmanPortsForInfo(container string, runtimeHost string) []dto.PortHelper {
	args := make([]string, 0, 6)
	if host := strings.TrimSpace(runtimeHost); strings.HasPrefix(host, "unix://") {
		args = append(args, "--url", host)
	}
	args = append(args, "port", container)
	out, err := exec.Command("podman", args...).CombinedOutput()
	if err != nil {
		return nil
	}
	lines := splitNonEmptyLines(string(out))
	var res []dto.PortHelper
	for _, line := range lines {
		parts := strings.Split(line, "->")
		if len(parts) != 2 {
			continue
		}
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])

		cp := ""
		proto := "tcp"
		if strings.Contains(left, "/") {
			lp := strings.SplitN(left, "/", 2)
			cp = strings.TrimSpace(lp[0])
			proto = strings.TrimSpace(lp[1])
		} else {
			cp = left
		}

		hostIP := ""
		hostPort := ""
		if strings.Contains(right, ":") {
			rp := strings.SplitN(right, ":", 2)
			hostIP = strings.TrimSpace(rp[0])
			hostPort = strings.TrimSpace(rp[1])
		} else {
			hostPort = right
		}

		res = append(res, dto.PortHelper{
			HostIP:        hostIP,
			HostPort:      hostPort,
			ContainerPort: cp,
			Protocol:      proto,
		})
	}
	return res
}

func podmanMountsForInfo(inspect map[string]interface{}) []dto.VolumeHelper {
	mountsAny, ok := inspect["Mounts"]
	if !ok || mountsAny == nil {
		return nil
	}
	var res []dto.VolumeHelper
	switch mounts := mountsAny.(type) {
	case []interface{}:
		for _, it := range mounts {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			res = append(res, dto.VolumeHelper{
				Type:         strings.TrimSpace(fmt.Sprint(m["Type"])),
				SourceDir:    strings.TrimSpace(fmt.Sprint(m["Source"])),
				ContainerDir: strings.TrimSpace(fmt.Sprint(m["Destination"])),
				Mode:         strings.TrimSpace(fmt.Sprint(m["Mode"])),
			})
		}
	}
	return res
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

	return nil
}

func sanitizeStaticIPForContainerNetwork(ctx context.Context, client *client.Client, req *dto.ContainerOperate) {
	if req == nil || strings.TrimSpace(req.Network) == "" {
		return
	}

	if ip := strings.TrimSpace(req.Ipv4); ip != "" {
		ok, subnet, err := ipv4BelongsToDockerNetwork(ctx, client, req.Network, ip)
		if err != nil {
			global.LOG.Warnf("inspect network %s for ipv4 validation failed: %v", req.Network, err)
		} else if !ok {
			global.LOG.Warnf("static ipv4 %s does not belong to network %s subnet %s, fallback to dynamic ip", ip, req.Network, subnet)
			req.Ipv4 = ""
		}
	}

	if ip := strings.TrimSpace(req.Ipv6); ip != "" {
		ok, subnet, err := ipv6BelongsToDockerNetwork(ctx, client, req.Network, ip)
		if err != nil {
			global.LOG.Warnf("inspect network %s for ipv6 validation failed: %v", req.Network, err)
		} else if !ok {
			global.LOG.Warnf("static ipv6 %s does not belong to network %s subnet %s, fallback to dynamic ip", ip, req.Network, subnet)
			req.Ipv6 = ""
		}
	}
}

func ipv4BelongsToDockerNetwork(ctx context.Context, client *client.Client, networkName, ip string) (bool, string, error) {
	networkInfo, err := client.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err != nil {
		return false, "", err
	}
	return ipBelongsToDockerNetwork(networkInfo, ip, false)
}

func ipv6BelongsToDockerNetwork(ctx context.Context, client *client.Client, networkName, ip string) (bool, string, error) {
	networkInfo, err := client.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err != nil {
		return false, "", err
	}
	return ipBelongsToDockerNetwork(networkInfo, ip, true)
}

func ipBelongsToDockerNetwork(networkInfo network.Inspect, ip string, wantIPv6 bool) (bool, string, error) {
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return false, "", fmt.Errorf("invalid ip: %s", ip)
	}

	var matchedSubnets []string
	for _, cfg := range networkInfo.IPAM.Config {
		subnet := strings.TrimSpace(cfg.Subnet)
		if subnet == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(subnet)
		if err != nil {
			continue
		}
		isIPv6Subnet := ipNet.IP.To4() == nil
		if isIPv6Subnet != wantIPv6 {
			continue
		}
		if ipNet.Contains(parsedIP) {
			return true, subnet, nil
		}
		matchedSubnets = append(matchedSubnets, subnet)
	}

	if len(matchedSubnets) > 0 {
		return false, strings.Join(matchedSubnets, ","), nil
	}
	return true, "", nil
}

func retryCreateWithoutStaticIPOnSubnetError(
	ctx context.Context,
	client *client.Client,
	containerName string,
	config *container.Config,
	hostConf *container.HostConfig,
	networkConf *network.NetworkingConfig,
	createErr error,
) (container.CreateResponse, error, bool) {
	if !isStaticIPSubnetError(createErr) || networkConf == nil || len(networkConf.EndpointsConfig) == 0 {
		return container.CreateResponse{}, createErr, false
	}

	clonedNetworkConf := cloneNetworkingConfigWithoutStaticIP(networkConf)
	if !networkConfigStaticIPChanged(clonedNetworkConf, networkConf) {
		return container.CreateResponse{}, createErr, false
	}

	global.LOG.Warnf("create container %s failed with static ip subnet mismatch, retry with dynamic ip: %v", containerName, createErr)
	con, retryErr := client.ContainerCreate(ctx, config, hostConf, clonedNetworkConf, &v1.Platform{}, containerName)
	if retryErr != nil {
		return container.CreateResponse{}, retryErr, true
	}
	return con, nil, true
}

func retryStartWithoutStaticIPOnSubnetError(
	ctx context.Context,
	client *client.Client,
	containerName string,
	containerID string,
	config *container.Config,
	hostConf *container.HostConfig,
	networkConf *network.NetworkingConfig,
	startErr error,
) (error, bool) {
	if !isStaticIPSubnetError(startErr) || networkConf == nil || len(networkConf.EndpointsConfig) == 0 {
		return startErr, false
	}

	clonedNetworkConf := cloneNetworkingConfigWithoutStaticIP(networkConf)
	if !networkConfigStaticIPChanged(clonedNetworkConf, networkConf) {
		return startErr, false
	}

	_ = client.ContainerRemove(ctx, containerID, container.RemoveOptions{RemoveVolumes: true, Force: true})
	global.LOG.Warnf("start container %s failed with static ip subnet mismatch, retry with dynamic ip: %v", containerName, startErr)
	con, err := client.ContainerCreate(ctx, config, hostConf, clonedNetworkConf, &v1.Platform{}, containerName)
	if err != nil {
		return fmt.Errorf("retry container create without static ip failed, err: %v", err), true
	}
	if err := client.ContainerStart(ctx, con.ID, container.StartOptions{}); err != nil {
		_ = client.ContainerRemove(ctx, con.ID, container.RemoveOptions{RemoveVolumes: true, Force: true})
		return fmt.Errorf("retry container start without static ip failed, err: %v", err), true
	}
	return nil, true
}

func isStaticIPSubnetError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "requested static ip") && strings.Contains(msg, "not in any subnet")
}

func cloneNetworkingConfigWithoutStaticIP(src *network.NetworkingConfig) *network.NetworkingConfig {
	if src == nil {
		return nil
	}
	dst := &network.NetworkingConfig{
		EndpointsConfig: make(map[string]*network.EndpointSettings, len(src.EndpointsConfig)),
	}
	for name, endpoint := range src.EndpointsConfig {
		if endpoint == nil {
			dst.EndpointsConfig[name] = nil
			continue
		}
		cloned := *endpoint
		if endpoint.IPAMConfig != nil {
			ipam := *endpoint.IPAMConfig
			ipam.IPv4Address = ""
			ipam.IPv6Address = ""
			cloned.IPAMConfig = &ipam
		}
		dst.EndpointsConfig[name] = &cloned
	}
	return dst
}

func networkConfigStaticIPChanged(current *network.NetworkingConfig, original *network.NetworkingConfig) bool {
	if current == nil || original == nil {
		return false
	}
	for name, endpoint := range original.EndpointsConfig {
		if endpoint == nil || endpoint.IPAMConfig == nil {
			continue
		}
		oldIPv4 := strings.TrimSpace(endpoint.IPAMConfig.IPv4Address)
		oldIPv6 := strings.TrimSpace(endpoint.IPAMConfig.IPv6Address)
		newEndpoint := current.EndpointsConfig[name]
		if newEndpoint == nil || newEndpoint.IPAMConfig == nil {
			if oldIPv4 != "" || oldIPv6 != "" {
				return true
			}
			continue
		}
		if oldIPv4 != strings.TrimSpace(newEndpoint.IPAMConfig.IPv4Address) || oldIPv6 != strings.TrimSpace(newEndpoint.IPAMConfig.IPv6Address) {
			return true
		}
	}
	return false
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
	return client.ContainerRename(ctx, req.Name, req.NewName)
}

func (u *ContainerService) ContainerCommit(req *dto.ContainerCommit) error {
	ctx := context.Background()
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	options := container.CommitOptions{
		Reference: req.NewImageName,
		Comment:   req.Comment,
		Author:    req.Author,
		Changes:   nil,
		Pause:     req.Pause,
		Config:    nil,
	}
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
	return nil
}

func containerOperationPodmanCLI(ctx context.Context, runtimeHost string, names []string, op string) error {
	if err := docker.PodmanEnsureReady(ctx); err != nil {
		return err
	}
	host := strings.TrimSpace(runtimeHost)
	for _, item := range names {
		global.LOG.Infof("start container %s operation %s", item, op)
		var args []string
		switch op {
		case constant.ContainerOpStart:
			args = []string{"start", item}
		case constant.ContainerOpStop:
			args = []string{"stop", item}
		case constant.ContainerOpRestart:
			args = []string{"restart", item}
		case constant.ContainerOpKill:
			args = []string{"kill", "-s", "SIGKILL", item}
		case constant.ContainerOpPause:
			args = []string{"pause", item}
		case constant.ContainerOpUnpause:
			args = []string{"unpause", item}
		case constant.ContainerOpRemove:
			args = []string{"rm", "-f", "-v", item}
		default:
			return errors.New("operation is empty")
		}
		c, err := docker.RuntimeCommandWithHost(ctx, host, args...)
		if err != nil {
			return err
		}
		out, err := c.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				return err
			}
			return errors.New(msg)
		}
	}
	return nil
}

func (u *ContainerService) ContainerLogClean(req *dto.OperationWithName) error {
	if cmd.CheckIllegal(req.Name) {
		return buserr.New(constant.ErrCmdIllegal)
	}
	ctx := context.Background()
	if docker.IsPodmanRuntime(ctx) {
		if runtime.GOOS == "darwin" {
			return errors.New("podman on darwin does not support cleaning container log files (logs are stored inside podman machine); please restart/recreate the container to clear logs")
		}
		host := strings.TrimSpace(req.RuntimeHost)
		if host == "" {
			host, _ = resolveLinuxPodmanContainerHost(ctx, req.Name)
		}
		logPath, err := podmanContainerLogPath(ctx, req.Name, host)
		if err != nil {
			return err
		}
		if logPath == "" {
			return errors.New("container log path is empty")
		}
		return truncateContainerLogFiles(logPath)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	containerItem, err := client.ContainerInspect(ctx, req.Name)
	if err != nil {
		return err
	}
	if containerItem.LogPath == "" {
		return errors.New("container log path is empty")
	}
	return truncateContainerLogFiles(containerItem.LogPath)
}

func (u *ContainerService) ContainerLogs(wsConn *websocket.Conn, containerType, containerID, since, tail, runtimeHost string, follow bool) error {
	defer func() { wsConn.Close() }()
	if cmd.CheckIllegal(containerID, since, tail) {
		return buserr.New(constant.ErrCmdIllegal)
	}
	var cmdExec *exec.Cmd
	if containerType == "compose" {
		commandArg := []string{"-f", containerID, "logs"}
		if tail != "0" {
			commandArg = append(commandArg, "--tail", tail)
		}
		if since != "all" {
			commandArg = append(commandArg, "--since", since)
		}
		if follow {
			commandArg = append(commandArg, "-f")
		}
		c, err := compose.Command(context.Background(), commandArg...)
		if err != nil {
			return err
		}
		cmdExec = c
	} else {
		ctx := context.Background()
		isPodman := docker.IsPodmanRuntime(ctx)
		host := strings.TrimSpace(runtimeHost)
		if host == "" && isPodman {
			host, _ = resolveLinuxPodmanContainerHost(ctx, containerID)
		}

		commandArg := make([]string, 0, 12)
		commandArg = append(commandArg, "logs")
		if tail != "0" {
			commandArg = append(commandArg, "--tail", tail)
		}
		if since != "all" {
			commandArg = append(commandArg, "--since", since)
		}
		if follow {
			commandArg = append(commandArg, "-f")
		}
		commandArg = append(commandArg, containerID)
		c, err := docker.RuntimeCommandWithHost(ctx, host, commandArg...)
		if err != nil {
			return err
		}
		cmdExec = c
	}
	if !follow {
		cmdExec.Stderr = cmdExec.Stdout
		stdout, _ := cmdExec.CombinedOutput()
		if !utf8.Valid(stdout) {
			return errors.New("invalid utf8")
		}
		if err := wsConn.WriteMessage(websocket.TextMessage, stdout); err != nil {
			global.LOG.Errorf("send message with log to ws failed, err: %v", err)
		}
		return nil
	}

	stdout, err := cmdExec.StdoutPipe()
	if err != nil {
		_ = cmdExec.Process.Signal(syscall.SIGTERM)
		return err
	}
	cmdExec.Stderr = cmdExec.Stdout
	if err := cmdExec.Start(); err != nil {
		_ = cmdExec.Process.Signal(syscall.SIGTERM)
		return err
	}
	exitCh := make(chan struct{})
	go func() {
		_, wsData, _ := wsConn.ReadMessage()
		if string(wsData) == "close conn" {
			_ = cmdExec.Process.Signal(syscall.SIGTERM)
			exitCh <- struct{}{}
		}
	}()

	go func() {
		buffer := make([]byte, 1024)
		for {
			select {
			case <-exitCh:
				return
			default:
				n, err := stdout.Read(buffer)
				if err != nil {
					if err == io.EOF {
						return
					}
					global.LOG.Errorf("read bytes from log failed, err: %v", err)
					return
				}
				if !utf8.Valid(buffer[:n]) {
					continue
				}
				if err = wsConn.WriteMessage(websocket.TextMessage, buffer[:n]); err != nil {
					global.LOG.Errorf("send message with log to ws failed, err: %v", err)
					return
				}
			}
		}
	}()
	_ = cmdExec.Wait()
	return nil
}

func (u *ContainerService) DownloadContainerLogs(containerType, containerID, since, tail, runtimeHost string) (string, error) {
	if cmd.CheckIllegal(containerID, since, tail) {
		return "", buserr.New(constant.ErrCmdIllegal)
	}
	var cmdExec *exec.Cmd
	if containerType == "compose" {
		commandArg := []string{"-f", containerID, "logs"}
		if tail != "0" {
			commandArg = append(commandArg, "--tail", tail)
		}
		if since != "all" {
			commandArg = append(commandArg, "--since", since)
		}
		c, err := compose.Command(context.Background(), commandArg...)
		if err != nil {
			return "", err
		}
		cmdExec = c
	} else {
		ctx := context.Background()
		isPodman := docker.IsPodmanRuntime(ctx)
		host := strings.TrimSpace(runtimeHost)
		if host == "" && isPodman {
			host, _ = resolveLinuxPodmanContainerHost(ctx, containerID)
		}

		commandArg := make([]string, 0, 12)
		commandArg = append(commandArg, "logs")
		if tail != "0" {
			commandArg = append(commandArg, "--tail", tail)
		}
		if since != "all" {
			commandArg = append(commandArg, "--since", since)
		}
		commandArg = append(commandArg, containerID)
		c, err := docker.RuntimeCommandWithHost(ctx, host, commandArg...)
		if err != nil {
			return "", err
		}
		cmdExec = c
	}

	stdout, err := cmdExec.StdoutPipe()
	if err != nil {
		_ = cmdExec.Process.Signal(syscall.SIGTERM)
		return "", err
	}
	cmdExec.Stderr = cmdExec.Stdout
	if err := cmdExec.Start(); err != nil {
		_ = cmdExec.Process.Signal(syscall.SIGTERM)
		return "", err
	}

	tempFile, err := os.CreateTemp("", "cmd_output_*.txt")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	errCh := make(chan error)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if _, err := tempFile.WriteString(line + "\n"); err != nil {
				errCh <- err
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	select {
	case err := <-errCh:
		if err != nil {
			global.LOG.Errorf("Error: %v", err)
		}
	case <-time.After(3 * time.Second):
		global.LOG.Errorf("Timeout reached")
	}
	return tempFile.Name(), nil
}

func (u *ContainerService) ContainerStats(id string) (*dto.ContainerStats, error) {
	ctx := context.Background()
	if docker.IsPodmanRuntime(ctx) {
		host := ""
		host, _ = resolveLinuxPodmanContainerHost(ctx, id)
		return containerStatsPodman(id, host)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	res, err := client.ContainerStats(context.TODO(), id, false)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		res.Body.Close()
		return nil, err
	}
	res.Body.Close()
	var stats *container.Stats
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, err
	}
	var data dto.ContainerStats
	data.CPUPercent = calculateCPUPercentUnix(stats)
	data.IORead, data.IOWrite = calculateBlockIO(stats.BlkioStats)
	data.Memory = float64(stats.MemoryStats.Usage) / 1024 / 1024
	if cache, ok := stats.MemoryStats.Stats["cache"]; ok {
		data.Cache = float64(cache) / 1024 / 1024
	}
	data.NetworkRX, data.NetworkTX = calculateNetwork(stats.Networks)
	data.ShotTime = stats.Read
	return &data, nil
}

func inspectPodman(req *dto.InspectReq) (string, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return "", err
	}
	args := make([]string, 0, 8)
	if host := strings.TrimSpace(req.RuntimeHost); strings.HasPrefix(host, "unix://") {
		args = append(args, "--url", host)
	}
	switch req.Type {
	case "container":
		args = append(args, "container", "inspect", req.ID, "--format", "json")
	case "image":
		args = append(args, "image", "inspect", req.ID, "--format", "json")
	case "network":
		args = append(args, "network", "inspect", req.ID, "--format", "json")
	case "volume":
		args = append(args, "volume", "inspect", req.ID, "--format", "json")
	default:
		return "", errors.New("invalid inspect type")
	}
	c := exec.Command("podman", args...)
	out, err := c.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", err
		}
		return "", errors.New(msg)
	}
	return string(out), nil
}

func containerStatsPodman(id string, runtimeHost string) (*dto.ContainerStats, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return nil, err
	}
	args := make([]string, 0, 10)
	if host := strings.TrimSpace(runtimeHost); strings.HasPrefix(host, "unix://") {
		args = append(args, "--url", host)
	}
	args = append(args, "stats", "--no-stream", "--format", "{{.ID}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}", id)
	c := exec.Command("podman", args...)
	out, err := c.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return nil, err
		}
		return nil, errors.New(msg)
	}
	lines := splitNonEmptyLines(string(out))
	if len(lines) == 0 {
		return nil, errors.New("no stats")
	}
	parts := strings.Split(lines[0], "\t")
	if len(parts) < 6 {
		return nil, errors.New("invalid stats output")
	}
	memUsage, _ := parseMemUsagePairToMB(parts[2])
	netRx, netTx, _ := parsePairToMB(parts[4])
	ioRead, ioWrite, _ := parsePairToMB(parts[5])
	return &dto.ContainerStats{
		CPUPercent: parsePercent(parts[1]),
		Memory:     memUsage,
		Cache:      0,
		IORead:     ioRead,
		IOWrite:    ioWrite,
		NetworkRX:  netRx,
		NetworkTX:  netTx,
		ShotTime:   time.Now(),
	}, nil
}

func podmanContainerLogPath(ctx context.Context, containerName string, runtimeHost string) (string, error) {
	c, err := docker.RuntimeCommandWithHost(ctx, runtimeHost, "container", "inspect", containerName, "--format", "{{.LogPath}}")
	if err != nil {
		return "", err
	}
	out, err := c.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", err
		}
		return "", errors.New(msg)
	}
	return strings.TrimSpace(string(out)), nil
}

func truncateContainerLogFiles(logPath string) error {
	file, err := os.OpenFile(logPath, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	if err = file.Truncate(0); err != nil {
		return err
	}
	files, _ := filepath.Glob(fmt.Sprintf("%s.*", logPath))
	for _, file := range files {
		_ = os.Remove(file)
	}
	return nil
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

func (u *ContainerService) LoadContainerLogs(req *dto.OperationWithNameAndType) string {
	filePath := ""
	if req.Type == "compose-detail" {
		ctx := context.Background()
		options := container.ListOptions{All: true}
		isPodman := docker.IsPodmanRuntime(ctx)
		if !isPodman {
			options.Filters = filters.NewArgs()
			options.Filters.Add("label", fmt.Sprintf("%s=%s", composeProjectLabel, req.Name))
		}
		var (
			containers []types.Container
			err        error
		)
		if isPodman {
			containers, err = docker.ListContainersMerged(ctx, options)
		} else {
			cli, err := docker.NewDockerClient()
			if err != nil {
				return ""
			}
			defer cli.Close()
			containers, err = cli.ContainerList(ctx, options)
		}
		if err != nil {
			return ""
		}
		for _, container := range containers {
			if isPodman {
				name, ok := firstLabel(container.Labels, composeProjectLabel, podmanComposeProjectLabel)
				if !ok || name != req.Name {
					continue
				}
			}
			config, _ := firstLabel(container.Labels, composeConfigLabel, podmanComposeConfigLabel)
			workdir, _ := firstLabel(container.Labels, composeWorkdirLabel, podmanComposeWorkdirLabel)
			if len(config) != 0 && len(workdir) != 0 && strings.Contains(config, workdir) {
				filePath = config
				break
			} else {
				filePath = workdir
				break
			}
		}
		if len(containers) == 0 {
			composeItem, _ := repo.NewIComposeTemplateRepo().GetRecord(repo.NewCommonRepo().WithByName(req.Name))
			filePath = composeItem.Path
		}
	}
	if _, err := os.Stat(filePath); err != nil {
		return ""
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return string(content)
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

func checkImageLike(imageName string) bool {
	cli, err := docker.NewDockerClient()
	if err != nil {
		return false
	}
	images, err := cli.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		return false
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if strings.Contains(tag, imageName) {
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
