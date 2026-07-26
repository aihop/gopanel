package service

import (
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/shirou/gopsutil/v4/disk"
)

// 伪文件系统与只读系统卷不参与磁盘清理，列出来只会干扰
var skipFsTypes = map[string]struct{}{
	"devfs": {}, "tmpfs": {}, "devtmpfs": {}, "overlay": {}, "squashfs": {},
	"autofs": {}, "proc": {}, "sysfs": {}, "cgroup": {}, "cgroup2": {},
	"nullfs": {}, "fdescfs": {}, "iso9660": {},
}

// DiskOverview 磁盘容量概览。
//
// 这里不复用 dashboard 的 loadDiskInfo：它靠解析 `df -hT -P` 的文本输出，
// 而 macOS 的 df 没有 Type 列（-T 在 macOS 上是「按文件系统类型筛选」），
// 列会整体错位，实测在 Mac 上返回的是这种东西：
//
//	path="12G 0% /System/Volumes/Data"  type="1.8Ti"  total=0
//
// 非空但全是垃圾，靠「为空则回退」判断不出来。更要命的是这个功能允许
// 点磁盘卡片把挂载点设为扫描根——错位的 path 是个不存在的字符串，
// 扫描直接扫不到任何东西。
//
// 改用 gopsutil：本来就是项目依赖，跨平台走 statfs 系统调用，
// 不依赖任何命令行输出格式。
func DiskOverview() []dto.DiskInfo {
	return diskInfoFromPartitions()
}

func diskInfoFromPartitions() []dto.DiskInfo {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil
	}
	seen := make(map[string]struct{}, len(parts))
	out := make([]dto.DiskInfo, 0, len(parts))
	for _, p := range parts {
		if _, skip := skipFsTypes[strings.ToLower(p.Fstype)]; skip {
			continue
		}
		mount := filepath.Clean(p.Mountpoint)
		if mount == "" {
			continue
		}
		// 同一挂载点可能被多次报告（bind mount / firmlink），只留一份
		if _, ok := seen[mount]; ok {
			continue
		}
		usage, err := disk.Usage(mount)
		if err != nil || usage == nil || usage.Total == 0 {
			continue
		}
		seen[mount] = struct{}{}
		out = append(out, dto.DiskInfo{
			Path:              mount,
			Type:              p.Fstype,
			Device:            p.Device,
			Total:             usage.Total,
			Free:              usage.Free,
			Used:              usage.Used,
			UsedPercent:       usage.UsedPercent,
			InodesTotal:       usage.InodesTotal,
			InodesUsed:        usage.InodesUsed,
			InodesFree:        usage.InodesFree,
			InodesUsedPercent: usage.InodesUsedPercent,
		})
	}
	return out
}
