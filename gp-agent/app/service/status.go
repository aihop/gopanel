package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aihop/gopanel/gp-agent/global"
	"github.com/aihop/gopanel/gp-agent/init/caddy"
	"github.com/aihop/gopanel/gp-agent/init/daemon"
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

type LocalStatus struct {
	AtUnixMs       int64       `json:"at_unix_ms"`
	CPUPercent     float64     `json:"cpu_percent"`
	MemTotal       uint64      `json:"mem_total"`
	MemUsed        uint64      `json:"mem_used"`
	MemFree        uint64      `json:"mem_free"`
	MemUsedPercent float64     `json:"mem_used_percent"`
	Disks          []DiskUsage `json:"disks"`
}

type AgentStatus struct {
	Version          string `json:"version"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
	BaseDir          string `json:"base_dir"`
	SocketPath       string `json:"socket"`
	CaddyStatus      string `json:"caddy_status"`
	DaemonStatus     string `json:"daemon_status"`
	ManagedAppsCount int    `json:"managed_apps_count"`
	LastError        string `json:"last_error"`
}

var Version = "dev"

func GetLocalStatus(ctx context.Context) (LocalStatus, error) {
	out := LocalStatus{AtUnixMs: time.Now().UnixMilli()}

	cpuPct, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return LocalStatus{}, err
	}
	if len(cpuPct) > 0 {
		out.CPUPercent = cpuPct[0]
	}

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return LocalStatus{}, err
	}
	out.MemTotal = vm.Total
	out.MemUsed = vm.Used
	out.MemFree = vm.Free
	out.MemUsedPercent = vm.UsedPercent

	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return LocalStatus{}, err
	}
	for _, p := range parts {
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

func GetAgentStatus() AgentStatus {
	uptime := int64(0)
	if !global.CONF.StartedAt.IsZero() {
		uptime = int64(time.Since(global.CONF.StartedAt).Seconds())
	}
	caddyStatus := "disabled"
	if global.CONF.EnableCaddy {
		if caddy.Server.Status {
			caddyStatus = "running"
		} else {
			caddyStatus = "failed"
		}
	}
	daemonStatus := "disabled"
	if global.CONF.EnableDaemon {
		if daemon.Supervisor != nil {
			daemonStatus = "running"
		} else {
			daemonStatus = "failed"
		}
	}
	return AgentStatus{
		Version:          Version,
		UptimeSeconds:    uptime,
		BaseDir:          global.CONF.BaseDir,
		SocketPath:       global.CONF.SocketPath,
		CaddyStatus:      caddyStatus,
		DaemonStatus:     daemonStatus,
		ManagedAppsCount: 0,
		LastError:        "",
	}
}

func GetAgentStatusJSON() (string, error) {
	b, err := json.Marshal(GetAgentStatus())
	if err != nil {
		return "", err
	}
	return string(b), nil
}
