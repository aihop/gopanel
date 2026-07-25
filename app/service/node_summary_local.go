package service

import (
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/shirou/gopsutil/v4/host"
)

// certExpiringWindowDays 证书剩余天数低于该值即计入告警
const certExpiringWindowDays = 30

// LocalNodeSummary 生成本机摘要，供主控通过只读接口拉取。
// 复用 DashboardService.LoadCurrentInfo 采集 CPU/内存/磁盘，容器与证书部分尽力而为，
// 任一子项失败都不影响整体返回——主控宁可拿到部分数据，也不要因为 docker 挂了就把节点标成离线。
func LocalNodeSummary() model.NodeSummary {
	summary := model.NodeSummary{
		Version:  constant.AppVersion,
		ShotTime: time.Now(),
	}

	if hostInfo, err := host.Info(); err == nil && hostInfo != nil {
		summary.Hostname = hostInfo.Hostname
		summary.OS = hostInfo.Platform + " " + hostInfo.PlatformVersion
		summary.Uptime = hostInfo.Uptime
	}

	current := NewIDashboardService().LoadCurrentInfo(dto.DashboardReq{Scope: "basic"})
	if current != nil {
		summary.CPUPercent = current.CPUUsedPercent
		summary.CPUTotal = current.CPUTotal
		summary.Load1 = current.Load1
		summary.MemPercent = current.MemoryUsedPercent
		summary.MemTotal = current.MemoryTotal
		summary.MemUsed = current.MemoryUsed
		fillMaxDisk(&summary, current.DiskData)
	}

	fillContainerStat(&summary)
	fillCertStat(&summary)

	return summary
}

// fillMaxDisk 只取占用率最高的那块盘——细条上放不下多块盘，而“最满的那块”正是要告警的对象
func fillMaxDisk(summary *model.NodeSummary, disks []dto.DiskInfo) {
	for _, item := range disks {
		if item.UsedPercent > summary.DiskMaxPercent {
			summary.DiskMaxPercent = item.UsedPercent
			summary.DiskMaxPath = item.Path
		}
	}
}

func fillContainerStat(summary *model.NodeSummary) {
	client, err := docker.NewClient()
	if err != nil {
		global.LOG.Debugf("[Node] 采集容器摘要跳过，容器运行时不可用: %v", err)
		return
	}
	defer client.Close()

	containers, err := client.ListAllContainers()
	if err != nil {
		global.LOG.Debugf("[Node] 采集容器列表失败: %v", err)
		return
	}
	summary.ContainerTotal = len(containers)
	for _, item := range containers {
		switch item.State {
		case "running":
			summary.ContainerRunning++
		case "created", "exited", "removing":
			// 正常的停止状态，不计入异常
		default:
			// dead / restarting / paused 等需要人工关注
			summary.ContainerAbnormal++
		}
	}
}

func fillCertStat(summary *model.NodeSummary) {
	ssls, err := repo.NewSSL().ListBy()
	if err != nil {
		global.LOG.Debugf("[Node] 采集证书摘要失败: %v", err)
		return
	}
	// 用 hasCert 判断是否已取到第一张证书，不能拿 minDays 的初值当哨兵——
	// 已过期证书的剩余天数本身就是负数，会和哨兵值撞上
	hasCert := false
	minDays := 0
	for _, item := range ssls {
		if item.ExpireDate.IsZero() {
			continue
		}
		summary.CertTotal++
		days := int(time.Until(item.ExpireDate).Hours() / 24)
		if days <= certExpiringWindowDays {
			summary.CertExpiringCount++
		}
		if !hasCert || days < minDays {
			minDays = days
			hasCert = true
		}
	}
	if hasCert {
		summary.CertMinDays = minDays
	}
}
