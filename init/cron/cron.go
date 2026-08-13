package cron

import (
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

var (
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

	global.Cron = cron.New(cron.WithLocation(time.Local))

	// 监控数据采集任务（每分钟执行一次），系统遥测，不是用户可配置的计划任务
	_, err := global.Cron.AddFunc("* * * * *", func() {
		recordMonitorData()
	})
	if err != nil {
		global.LOG.Errorf("[Cron] 添加 Monitor 任务失败: %v", err)
	}

	// 多节点摘要采集（每分钟一轮）。串行拉取，单节点 5 秒超时，只在配置了节点时才真正发请求。
	// 采集完立刻评估告警——两者必须同频，否则会拿上一轮的旧数据发新告警。
	_, err = global.Cron.AddFunc("* * * * *", func() {
		service.CollectAllNodes()
		service.EvaluateAlerts()
	})
	if err != nil {
		global.LOG.Errorf("[Cron] 添加节点摘要采集任务失败: %v", err)
	}

	_, err = global.Cron.AddFunc("* * * * *", func() {
		service.EvaluateSecurityRisks()
		service.RetrySecurityNotifications()
		service.AnalyzePendingSecurityEvents()
	})
	if err != nil {
		global.LOG.Errorf("[Cron] 添加 AI 安全监测任务失败: %v", err)
	}

	_, err = global.Cron.AddFunc("* * * * *", service.ReconcileFlowRuns)
	if err != nil {
		global.LOG.Errorf("[Cron] 添加 Flow 恢复任务失败: %v", err)
	}

	global.Cron.Start()
	service.ReconcileFlowRuns()
	global.LOG.Info("[Cron] task scheduler started")

	seedDefaultSSLRenewJob()
	loadCronjobs()
}

// seedDefaultSSLRenewJob 把原本硬编码的每日 SSL 续签任务收编成一条默认计划任务，
// 只在第一次启动（还没有任何 ssl_renew 类型任务）时插入一次，之后完全交给用户在计划任务页面管理
func seedDefaultSSLRenewJob() {
	jobs, err := repo.NewCronjob().ListByType("ssl_renew")
	if err != nil {
		global.LOG.Errorf("[Cron] 查询 SSL 续签计划任务失败: %v", err)
		return
	}
	if len(jobs) > 0 {
		return
	}
	job := &model.Cronjob{
		Name:   "SSL 证书自动续签",
		Type:   "ssl_renew",
		Spec:   "0 2 * * *",
		Status: constant.StatusEnable,
	}
	if err := repo.NewCronjob().Create(job); err != nil {
		global.LOG.Errorf("[Cron] 创建默认 SSL 续签计划任务失败: %v", err)
	}
}

// loadCronjobs 启动时把所有已启用的计划任务注册进调度器
func loadCronjobs() {
	jobs, err := repo.NewCronjob().ListEnabled()
	if err != nil {
		global.LOG.Errorf("[Cron] 加载计划任务失败: %v", err)
		return
	}
	cronjobRepo := repo.NewCronjob()
	for _, job := range jobs {
		jobID := job.ID
		entryID, err := global.Cron.AddFunc(job.Spec, func() {
			service.NewCronjobService().Run(jobID)
		})
		if err != nil {
			global.LOG.Errorf("[Cron] 注册计划任务 %d(%s) 失败: %v", job.ID, job.Name, err)
			continue
		}
		if err := cronjobRepo.UpdateEntryID(job.ID, int(entryID)); err != nil {
			global.LOG.Errorf("[Cron] 更新计划任务 %d EntryID 失败: %v", job.ID, err)
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
