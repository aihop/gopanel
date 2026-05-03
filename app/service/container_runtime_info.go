package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"io"
	"runtime"
	"strings"
)

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
	data := dto.ResourceLimit{CPU: cpuCounts, Memory: memoryInfo.Total}
	return &data, nil
}
func (u *ContainerService) ContainerStatsByID(id string) (*dto.ContainerStats, error) {
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
