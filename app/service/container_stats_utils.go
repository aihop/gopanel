package service

import (
	"context"
	"encoding/json"
	"github.com/aihop/gopanel/app/dto"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"io"
	"strings"
)

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
func loadCpuAndMem(client *client.Client, containerStr string) dto.ContainerListStats {
	data := dto.ContainerListStats{ContainerID: containerStr}
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
