package api

import (
	"errors"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

type mobileContainerSummary struct {
	ContainerID   string   `json:"containerID"`
	Name          string   `json:"name"`
	ImageName     string   `json:"imageName"`
	State         string   `json:"state"`
	RunTime       string   `json:"runTime"`
	RuntimeHost   string   `json:"runtimeHost"`
	RuntimeKind   string   `json:"runtimeKind"`
	SourceType    string   `json:"sourceType"`
	Ports         []string `json:"ports"`
	CPUPercent    float64  `json:"cpuPercent"`
	MemoryUsage   uint64   `json:"memoryUsage"`
	MemoryLimit   uint64   `json:"memoryLimit"`
	MemoryPercent float64  `json:"memoryPercent"`
}

func GetMobileContainers(c fiber.Ctx) error {
	containerManager := service.NewIContainerService()
	total, result, err := containerManager.Page(&dto.PageContainer{
		PageInfo: dto.PageInfo{Page: 1, Limit: 1000},
		State:    "all",
		OrderBy:  "name",
		Order:    "ascending",
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	containers, ok := result.([]dto.ContainerInfo)
	if !ok {
		return c.JSON(e.Fail(errors.New("容器列表格式无效")))
	}
	stats, err := containerManager.ContainerListStats()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	statsByID := make(map[string]dto.ContainerListStats, len(stats))
	for _, stat := range stats {
		statsByID[stat.ContainerID] = stat
	}
	items := make([]mobileContainerSummary, 0, len(containers))
	running := 0
	for _, container := range containers {
		stat := statsByID[container.ContainerID]
		if container.State == "running" {
			running++
		}
		items = append(items, mobileContainerSummary{
			ContainerID:   container.ContainerID,
			Name:          container.Name,
			ImageName:     container.ImageName,
			State:         container.State,
			RunTime:       container.RunTime,
			RuntimeHost:   container.RuntimeHost,
			RuntimeKind:   container.RuntimeKind,
			SourceType:    container.SourceType,
			Ports:         container.Ports,
			CPUPercent:    stat.CPUPercent,
			MemoryUsage:   stat.MemoryUsage,
			MemoryLimit:   stat.MemoryLimit,
			MemoryPercent: stat.MemoryPercent,
		})
	}
	return c.JSON(e.Succ(fiber.Map{
		"items":   items,
		"total":   total,
		"running": running,
		"stopped": int(total) - running,
	}))
}

func OperateMobileContainer(c fiber.Ctx) error {
	var req struct {
		ContainerID string `json:"containerID"`
		Operation   string `json:"operation"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	req.ContainerID = strings.TrimSpace(req.ContainerID)
	req.Operation = strings.TrimSpace(req.Operation)
	if req.ContainerID == "" {
		return c.JSON(e.Fail(errors.New("容器参数无效")))
	}
	if !isMobileContainerOperationAllowed(req.Operation) {
		return c.JSON(e.Fail(errors.New("手机端不允许执行该容器操作")))
	}
	if err := service.NewIContainerService().ContainerOperation(&dto.ContainerOperation{
		Names:     []string{req.ContainerID},
		Operation: req.Operation,
	}); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func isMobileContainerOperationAllowed(operation string) bool {
	return operation == "start" || operation == "stop" || operation == "restart"
}
