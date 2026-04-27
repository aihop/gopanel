package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/aihop/gopanel/utils/common"
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
		records      []types.Container
		list         []types.Container
		dockerClient *client.Client
	)
	ctx := context.Background()
	resolved := docker.ResolveRuntime(ctx)
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
		dockerClient = client
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
	appInstallRepo := repo.NewIAppInstallRepo()
	websiteRepo := repo.NewWebsite()
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
			RuntimeKind:   string(resolved.Kind),
			RuntimeMode:   inferContainerRuntimeMode(string(resolved.Kind), ""),
			SourceType:    "manual",
		}
		if isPodman && sourceByID != nil {
			info.RuntimeHost = strings.TrimSpace(sourceByID[item.ID])
		}
		info.RuntimeMode = inferContainerRuntimeMode(info.RuntimeKind, info.RuntimeHost)
		install, _ := appInstallRepo.GetFirst(appInstallRepo.WithContainerName(info.Name))
		if install.ID > 0 {
			info.AppInstallName = install.Name
			info.SourceType = "app"
			websites, _ := websiteRepo.GetBy(websiteRepo.WithAppInstallId(install.ID))
			for _, website := range websites {
				info.Websites = append(info.Websites, website.PrimaryDomain)
			}
		}
		if info.SourceType == "manual" {
			if website, err := websiteRepo.GetFirst(websiteRepo.WithContainerID(item.ID)); err == nil && website.ID > 0 {
				if strings.TrimSpace(website.PrimaryDomain) != "" {
					info.Websites = append(info.Websites, website.PrimaryDomain)
				}
				if website.PipelineID > 0 {
					info.SourceType = "pipeline"
				} else {
					info.SourceType = "website"
				}
			} else if strings.HasPrefix(strings.ToLower(info.Name), "gopanel-engine-pipeline-") {
				info.SourceType = "pipeline"
			}
		}
		if info.SourceType == "manual" && info.IsFromCompose {
			info.SourceType = "compose"
		}
		backDatas[i] = info
		if item.NetworkSettings != nil && len(item.NetworkSettings.Networks) > 0 {
			networks := make([]string, 0, len(item.NetworkSettings.Networks))
			for key := range item.NetworkSettings.Networks {
				if ip := strings.TrimSpace(item.NetworkSettings.Networks[key].IPAddress); ip != "" {
					networks = append(networks, ip)
				}
			}
			backDatas[i].Network = normalizeContainerIPList(networks)
		}
		if len(backDatas[i].Network) == 0 {
			runtimeHost := ""
			if isPodman && sourceByID != nil {
				runtimeHost = strings.TrimSpace(sourceByID[item.ID])
			}
			backDatas[i].Network = inspectContainerIPsForList(ctx, dockerClient, item.ID, runtimeHost)
		}
		backDatas[i].RunUser = inspectContainerRunUserForList(ctx, dockerClient, item.ID, backDatas[i].RuntimeHost)
	}

	return int64(total), backDatas, nil
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

func (u *ContainerService) Inspect(req *dto.InspectReq) (string, error) {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		return inspectPodman(req)
	}
	ctx := context.Background()
	isPodman := docker.IsPodmanRuntime(ctx)

	host := strings.TrimSpace(req.RuntimeHost)
	if host == "" && isPodman {
		switch req.Type {
		case "container":
			host, _ = resolveLinuxPodmanContainerHost(ctx, req.ID)
		case "image":
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
