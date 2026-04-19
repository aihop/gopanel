package monitor

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

type DiskUsage struct {
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type Status struct {
	AtUnixMs      int64       `json:"at_unix_ms"`
	CPUPercent    float64     `json:"cpu_percent"`
	MemTotal      uint64      `json:"mem_total"`
	MemUsed       uint64      `json:"mem_used"`
	MemFree       uint64      `json:"mem_free"`
	MemUsedPercent float64    `json:"mem_used_percent"`
	Disks         []DiskUsage `json:"disks"`
}

func Collect(ctx context.Context) (Status, error) {
	out := Status{AtUnixMs: time.Now().UnixMilli()}

	cpuPct, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return Status{}, err
	}
	if len(cpuPct) > 0 {
		out.CPUPercent = cpuPct[0]
	}

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Status{}, err
	}
	out.MemTotal = vm.Total
	out.MemUsed = vm.Used
	out.MemFree = vm.Free
	out.MemUsedPercent = vm.UsedPercent

	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return Status{}, err
	}
	mountAllow := allowedMountpoints()
	for _, p := range parts {
		if !mountAllow(p.Mountpoint) {
			continue
		}
		u, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
			continue
		}
		out.Disks = append(out.Disks, DiskUsage{
			Mountpoint:  p.Mountpoint,
			Fstype:      p.Fstype,
			Total:       u.Total,
			Used:        u.Used,
			Free:        u.Free,
			UsedPercent: u.UsedPercent,
		})
	}

	return out, nil
}

