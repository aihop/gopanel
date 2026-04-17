package cron

import (
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

var (
	CronScheduler   *cron.Cron
	lastIOCounters  map[string]disk.IOCountersStat
	lastNetCounters []net.IOCountersStat
	lastMonitorTime time.Time
	monitorMutex    sync.Mutex
	isCronInit      bool
)

func Init() {
	if isCronInit {
		return
	}
	isCronInit = true

	CronScheduler = cron.New(cron.WithLocation(time.Local))

	// 每天凌晨 2:00 执行 SSL 自动续签任务
	_, err := CronScheduler.AddFunc("0 2 * * *", func() {
		global.LOG.Info("[Cron] 开始执行每日 SSL 自动续签检查任务")
		checkAndRenewSSL()
	})

	if err != nil {
		global.LOG.Errorf("[Cron] 添加 SSL 自动续签任务失败: %v", err)
	}

	// 2. 监控数据采集任务（每分钟执行一次）
	_, err = CronScheduler.AddFunc("* * * * *", func() {
		recordMonitorData()
	})
	if err != nil {
		global.LOG.Errorf("[Cron] 添加 Monitor 任务失败: %v", err)
	}

	CronScheduler.Start()
	global.LOG.Info("[Cron] task scheduler started")
}

func checkAndRenewSSL() {
	sslService := service.NewSSL()

	// 查找开启了自动续签的证书
	var certs []model.SSL
	if err := global.DB.Where("auto_renew = ?", true).Find(&certs).Error; err != nil {
		global.LOG.Errorf("[Cron] 查询需要自动续签的证书失败: %v", err)
		return
	}

	now := time.Now()
	for _, cert := range certs {
		// 只有通过 DNS API 签发的证书支持自动续签
		if cert.Type == "upload" || cert.Type == "caddy" {
			continue
		}

		// 如果过期时间减去7天在当前时间之前，说明距离过期不足7天了
		if cert.ExpireDate.AddDate(0, 0, -7).Before(now) {
			global.LOG.Infof("[Cron] 证书 %s (ID: %d) 将于 %s 过期，开始自动续签", cert.PrimaryDomain, cert.ID, cert.ExpireDate.Format("2006-01-02"))
			if err := sslService.Renew(cert.ID); err != nil {
				global.LOG.Errorf("[Cron] 证书 %s 自动续签请求失败: %v", cert.PrimaryDomain, err)
			}
		}
	}
}

// recordMonitorData 每分钟记录一次服务器的基础状态到 MonitorDB
func recordMonitorData() {
	if global.MonitorDB == nil {
		return
	}

	monitorMutex.Lock()
	defer monitorMutex.Unlock()

	now := time.Now()

	// 1. 记录基础信息 (CPU, 内存, 负载)
	var base model.MonitorBase
	base.CreatedAt = now

	// CPU
	if cpuPercents, err := cpu.Percent(0, false); err == nil && len(cpuPercents) > 0 {
		base.Cpu = cpuPercents[0]
	}
	// 内存
	if vMem, err := mem.VirtualMemory(); err == nil {
		base.Memory = vMem.UsedPercent
	}
	// 负载
	if l, err := load.Avg(); err == nil {
		base.CpuLoad1 = l.Load1
		base.CpuLoad5 = l.Load5
		base.CpuLoad15 = l.Load15
		// 简单计算负载使用率: Load1 / CPU核心数 * 100
		if cores, err := cpu.Counts(true); err == nil && cores > 0 {
			base.LoadUsage = (l.Load1 / float64(cores)) * 100
		}
	}

	global.MonitorDB.Create(&base)

	// 2. 记录磁盘 IO
	if ioCounters, err := disk.IOCounters(); err == nil {
		var ioRecords []model.MonitorIO
		for name, stat := range ioCounters {
			var readRate, writeRate, countRate, timeRate uint64
			if last, ok := lastIOCounters[name]; ok && !lastMonitorTime.IsZero() {
				duration := now.Sub(lastMonitorTime).Seconds()
				if duration > 0 {
					readRate = uint64(float64(stat.ReadBytes-last.ReadBytes) / duration)
					writeRate = uint64(float64(stat.WriteBytes-last.WriteBytes) / duration)
					countRate = uint64(float64((stat.ReadCount+stat.WriteCount)-(last.ReadCount+last.WriteCount)) / duration)
					timeRate = uint64(float64((stat.ReadTime+stat.WriteTime)-(last.ReadTime+last.WriteTime)) / duration)
				}
			}
			ioRecords = append(ioRecords, model.MonitorIO{
				BaseModel: model.BaseModel{CreatedAt: now},
				Name:      name,
				Read:      readRate,
				Write:     writeRate,
				Count:     countRate,
				Time:      timeRate,
			})
		}
		if len(ioRecords) > 0 && !lastMonitorTime.IsZero() {
			global.MonitorDB.Create(&ioRecords)
		}
		lastIOCounters = ioCounters
	}

	// 3. 记录网络流量
	if netCounters, err := net.IOCounters(true); err == nil {
		var netRecords []model.MonitorNetwork
		for _, stat := range netCounters {
			if stat.Name == "lo" {
				continue
			}
			var upRate, downRate float64
			if !lastMonitorTime.IsZero() {
				duration := now.Sub(lastMonitorTime).Seconds()
				if duration > 0 {
					for _, last := range lastNetCounters {
						if last.Name == stat.Name {
							upRate = float64(stat.BytesSent-last.BytesSent) / duration
							downRate = float64(stat.BytesRecv-last.BytesRecv) / duration
							break
						}
					}
				}
			}
			if upRate == 0 && downRate == 0 {
				continue
			}
			netRecords = append(netRecords, model.MonitorNetwork{
				BaseModel: model.BaseModel{CreatedAt: now},
				Name:      stat.Name,
				Up:        upRate,
				Down:      downRate,
			})
		}
		if len(netRecords) > 0 && !lastMonitorTime.IsZero() {
			global.MonitorDB.Create(&netRecords)
		}
		lastNetCounters = netCounters
	}

	lastMonitorTime = now
}
