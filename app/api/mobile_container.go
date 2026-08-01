package api

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/gormx"
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

type mobileContainerPublishPort struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
}

type mobileContainerPublishWebsite struct {
	ID            uint   `json:"id"`
	Alias         string `json:"alias"`
	PrimaryDomain string `json:"primaryDomain"`
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

func GetMobileContainerPublishOptions(c fiber.Ctx) error {
	containerID := strings.TrimSpace(c.Params("id"))
	if containerID == "" {
		return c.JSON(e.Fail(errors.New("容器参数无效")))
	}
	container, err := service.NewIContainerService().ContainerInfo(&dto.OperationWithName{
		Name:        containerID,
		RuntimeHost: strings.TrimSpace(c.Query("runtimeHost")),
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	websites, err := service.NewWebsite().List(gormx.NewContextx(200, "id desc"))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	websiteOptions := make([]mobileContainerPublishWebsite, 0, len(websites))
	for _, website := range websites {
		if website.Type != constant.Proxy || website.AppInstallID > 0 {
			continue
		}
		websiteOptions = append(websiteOptions, mobileContainerPublishWebsite{
			ID: website.ID, Alias: website.Alias, PrimaryDomain: website.PrimaryDomain,
		})
	}
	return c.JSON(e.Succ(fiber.Map{
		"ports":    mobilePublishedTCPPorts(container.ExposedPorts),
		"websites": websiteOptions,
	}))
}

func PublishMobileContainerWebsite(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ContainerWebsiteBind](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := service.NewIContainerService().BindWebsite(ctx, req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func mobilePublishedTCPPorts(ports []dto.PortHelper) []mobileContainerPublishPort {
	seen := make(map[int]struct{}, len(ports))
	result := make([]mobileContainerPublishPort, 0, len(ports))
	for _, port := range ports {
		if !strings.EqualFold(strings.TrimSpace(port.Protocol), "tcp") {
			continue
		}
		hostPort, err := strconv.Atoi(strings.TrimSpace(port.HostPort))
		if err != nil || hostPort < 1 || hostPort > 65535 {
			continue
		}
		if _, ok := seen[hostPort]; ok {
			continue
		}
		seen[hostPort] = struct{}{}
		result = append(result, mobileContainerPublishPort{
			HostPort: hostPort, ContainerPort: strings.TrimSpace(port.ContainerPort),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].HostPort < result[j].HostPort })
	return result
}

func isMobileContainerOperationAllowed(operation string) bool {
	return operation == "start" || operation == "stop" || operation == "restart"
}
