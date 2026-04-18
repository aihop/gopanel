package api

import (
	"sort"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/gofiber/fiber/v3"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/net"
)

// @Tags Monitor
// @Summary Load monitor datas
// @Param request body dto.MonitorSearch true "request"
// @Success 200 {array} dto.MonitorData
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/monitor/search [post]
func LoadMonitor(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.MonitorSearch](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}

	loc, _ := time.LoadLocation(common.LoadTimeZoneByCmd())
	req.StartTime = req.StartTime.In(loc)
	req.EndTime = req.EndTime.In(loc)

	var backdatas []dto.MonitorData
	if global.MonitorDB == nil {
		return c.JSON(e.Succ(backdatas))
	}

	// 前端通常通过 param 来区分（cpu, memory, load, io, network）
	// 如果前端没传 StartTime/EndTime，给个默认的范围，比如最近1小时
	now := time.Now()
	if req.StartTime.IsZero() {
		oneHourAgo := now.Add(-1 * time.Hour)
		req.StartTime = oneHourAgo
	}
	if req.EndTime.IsZero() {
		req.EndTime = now
	}

	if req.Param == "all" || req.Param == "cpu" {
		var bases []model.MonitorBase
		if err := global.MonitorDB.
			Where("created_at > ? AND created_at < ?", req.StartTime, req.EndTime).
			Find(&bases).Error; err == nil && len(bases) > 0 {
			var itemData dto.MonitorData
			itemData.Param = "cpu"
			for _, base := range bases {
				itemData.Date = append(itemData.Date, base.CreatedAt)
				itemData.Value = append(itemData.Value, base.Cpu)
			}
			backdatas = append(backdatas, itemData)
		}
	}

	if req.Param == "all" || req.Param == "memory" {
		var bases []model.MonitorBase
		if err := global.MonitorDB.
			Where("created_at > ? AND created_at < ?", req.StartTime, req.EndTime).
			Find(&bases).Error; err == nil && len(bases) > 0 {
			var itemData dto.MonitorData
			itemData.Param = "memory"
			for _, base := range bases {
				itemData.Date = append(itemData.Date, base.CreatedAt)
				itemData.Value = append(itemData.Value, base.Memory)
			}
			backdatas = append(backdatas, itemData)
		}
	}

	if req.Param == "all" || req.Param == "load" {
		var bases []model.MonitorBase
		if err := global.MonitorDB.
			Where("created_at > ? AND created_at < ?", req.StartTime, req.EndTime).
			Find(&bases).Error; err == nil && len(bases) > 0 {
			var itemData dto.MonitorData
			itemData.Param = "load"
			for _, base := range bases {
				itemData.Date = append(itemData.Date, base.CreatedAt)
				// Load 返回对象
				itemData.Value = append(itemData.Value, map[string]interface{}{
					"cpuLoad1":  base.CpuLoad1,
					"cpuLoad5":  base.CpuLoad5,
					"cpuLoad15": base.CpuLoad15,
					"loadUsage": base.LoadUsage,
				})
			}
			backdatas = append(backdatas, itemData)
		}
	}

	if req.Param == "all" || req.Param == "io" {
		var ios []model.MonitorIO
		query := global.MonitorDB.Where("created_at > ? AND created_at < ?", req.StartTime, req.EndTime)
		if req.Info != "" && req.Info != "all" {
			query = query.Where("name = ?", req.Info)
		}
		if err := query.Find(&ios).Error; err == nil && len(ios) > 0 {
			var itemData dto.MonitorData
			itemData.Param = "io"
			if req.Info == "all" || req.Info == "" {
				type ioSum struct {
					Read  uint64
					Write uint64
					Count uint64
					Time  uint64
				}
				sums := make(map[time.Time]*ioSum)
				var times []time.Time
				for _, io := range ios {
					if _, exists := sums[io.CreatedAt]; !exists {
						sums[io.CreatedAt] = &ioSum{}
						times = append(times, io.CreatedAt)
					}
					sums[io.CreatedAt].Read += io.Read
					sums[io.CreatedAt].Write += io.Write
					sums[io.CreatedAt].Count += io.Count
					sums[io.CreatedAt].Time += io.Time
				}
				for _, t := range times {
					itemData.Date = append(itemData.Date, t)
					itemData.Value = append(itemData.Value, map[string]interface{}{
						"read":  sums[t].Read,
						"write": sums[t].Write,
						"count": sums[t].Count,
						"time":  sums[t].Time,
					})
				}
			} else {
				for _, io := range ios {
					itemData.Date = append(itemData.Date, io.CreatedAt)
					itemData.Value = append(itemData.Value, map[string]interface{}{
						"read":  io.Read,
						"write": io.Write,
						"count": io.Count,
						"time":  io.Time,
					})
				}
			}
			backdatas = append(backdatas, itemData)
		}
	}

	if req.Param == "all" || req.Param == "network" {
		var nets []model.MonitorNetwork
		query := global.MonitorDB.Where("created_at > ? AND created_at < ?", req.StartTime, req.EndTime)
		if req.Info != "" && req.Info != "all" {
			query = query.Where("name = ?", req.Info)
		}
		if err := query.Find(&nets).Error; err == nil && len(nets) > 0 {
			var itemData dto.MonitorData
			itemData.Param = "network"
			if req.Info == "all" || req.Info == "" {
				type netSum struct {
					Up   float64
					Down float64
				}
				sums := make(map[time.Time]*netSum)
				var times []time.Time
				for _, n := range nets {
					if _, exists := sums[n.CreatedAt]; !exists {
						sums[n.CreatedAt] = &netSum{}
						times = append(times, n.CreatedAt)
					}
					sums[n.CreatedAt].Up += n.Up
					sums[n.CreatedAt].Down += n.Down
				}
				for _, t := range times {
					itemData.Date = append(itemData.Date, t)
					itemData.Value = append(itemData.Value, map[string]interface{}{
						"up":   sums[t].Up,
						"down": sums[t].Down,
					})
				}
			} else {
				for _, n := range nets {
					itemData.Date = append(itemData.Date, n.CreatedAt)
					itemData.Value = append(itemData.Value, map[string]interface{}{
						"up":   n.Up,
						"down": n.Down,
					})
				}
			}
			backdatas = append(backdatas, itemData)
		}
	}

	return c.JSON(e.Succ(backdatas))
}

// @Tags Monitor
// @Summary Clean monitor datas
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /hosts/monitor/clean [post]
// @x-panel-log {"bodyKeys":[],"paramKeys":[],"BeforeFunctions":[],"formatZH":"清空监控数据","formatEN":"clean monitor datas"}
func CleanMonitor(c fiber.Ctx) error {
	if global.MonitorDB == nil {
		return c.JSON(e.Succ())
	}
	if err := global.MonitorDB.Exec("DELETE FROM monitor_bases").Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := global.MonitorDB.Exec("DELETE FROM monitor_ios").Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := global.MonitorDB.Exec("DELETE FROM monitor_networks").Error; err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ())
}

func GetNetworkOptions(c fiber.Ctx) error {
	netStat, _ := net.IOCounters(true)
	var options []string
	options = append(options, "all")
	for _, net := range netStat {
		options = append(options, net.Name)
	}
	sort.Strings(options)
	return c.JSON(e.Succ(options))
}

func GetIOOptions(c fiber.Ctx) error {
	diskStat, _ := disk.IOCounters()
	var options []string
	options = append(options, "all")
	for _, net := range diskStat {
		options = append(options, net.Name)
	}
	sort.Strings(options)
	return c.JSON(e.Succ(options))
}
