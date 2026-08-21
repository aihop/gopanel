package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/i18n"
	"github.com/aihop/gopanel/utils/env"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/gofiber/fiber/v3"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

func AppsValidate(c fiber.Ctx) error {
	v, err := mem.VirtualMemory()
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	d, err := disk.Usage("/")
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	availableMemMB := v.Available / 1024 / 1024
	freeDiskGB := d.Free / 1024 / 1024 / 1024

	isWarning := false
	var messages []string

	if availableMemMB < 300 {
		isWarning = true
		messages = append(messages, i18n.GetMsgWithMap(constant.ErrAppsResourceInsufficientMemory, map[string]interface{}{"available": availableMemMB}))
	}

	if freeDiskGB < 1 {
		isWarning = true
		messages = append(messages, i18n.GetMsgWithMap(constant.ErrAppsResourceInsufficientDisk, map[string]interface{}{"available": freeDiskGB}))
	}

	return c.JSON(e.Succ(map[string]interface{}{
		"isWarning": isWarning,
		"message":   strings.Join(messages, "，") + i18n.GetMsg(constant.ErrAppsResourceInsufficientHint),
	}))
}

func RepairCompose(c fiber.Ctx) error {
	if runtime.GOOS != "linux" {
		return c.JSON(e.Error(buserr.New(constant.ErrAppsUnsupportedPlatform)))
	}
	resp, err := gpc.Do(context.Background(), "COMPOSE_INSTALL", map[string]interface{}{})
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unknown action") {
			return c.JSON(e.Error(buserr.New(constant.ErrAppsGpcOutdatedComposeInstall)))
		}
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(map[string]any{
		"output": strings.TrimSpace(resp.Output),
	}))
}

func RepairPodmanSubuid(c fiber.Ctx) error {
	if runtime.GOOS != "linux" {
		return c.JSON(e.Error(buserr.New(constant.ErrAppsUnsupportedPlatform)))
	}

	// Default to gopanel user or get from process env
	username := os.Getenv("USER")
	if username == "" {
		username = "gopanel"
	}
	if username == "root" {
		return c.JSON(e.Error(buserr.New(constant.ErrAppsSubUIDRepairUnnecessary)))
	}

	resp, err := gpc.Do(context.Background(), "REPAIR_PODMAN_SUBUID", map[string]interface{}{
		"username": username,
	})

	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unknown action") {
			return c.JSON(e.Error(buserr.New(constant.ErrAppsGpcOutdatedPodmanSubUID)))
		}
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(map[string]any{
		"output": strings.TrimSpace(resp.Output),
	}))
}

func isPortAvailable(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

func findAvailablePort(startPort int) int {
	for port := startPort + 1; port < 65535; port++ {
		if isPortAvailable(port) {
			return port
		}
	}
	return startPort
}

func RepairPortConflict(c fiber.Ctx) error {
	var req struct {
		InstallId uint `json:"installId"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.JSON(e.Fail(err))
	}

	installRepo := repo.NewIAppInstallRepo()
	install, err := installRepo.GetFirst(repo.NewCommonRepo().WithByID(req.InstallId))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	envMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(install.Env), &envMap); err != nil {
		return c.JSON(e.Fail(err))
	}

	changed := false
	for k, v := range envMap {
		if strings.Contains(k, "PORT") {
			var port int
			switch val := v.(type) {
			case float64:
				port = int(val)
			case string:
				port, _ = strconv.Atoi(val)
			}

			if port > 0 {
				if !isPortAvailable(port) {
					newPort := findAvailablePort(port)
					envMap[k] = newPort
					changed = true
					if k == "PANEL_APP_PORT_HTTP" {
						install.HttpPort = newPort
					}
					if k == "PANEL_APP_PORT_HTTPS" {
						install.HttpsPort = newPort
					}
				}
			}
		}
	}

	if !changed {
		return c.JSON(e.Error(buserr.New(constant.ErrAppsNoPortConflict)))
	}

	envBytes, _ := json.Marshal(envMap)
	install.Env = string(envBytes)
	if err := installRepo.Save(context.Background(), &install); err != nil {
		return c.JSON(e.Fail(err))
	}

	// Convert envMap to map[string]string for env.Write
	envStringMap := make(map[string]string)
	for k, v := range envMap {
		switch val := v.(type) {
		case string:
			envStringMap[k] = val
		case float64:
			envStringMap[k] = strconv.FormatFloat(val, 'f', -1, 64)
		case int:
			envStringMap[k] = strconv.Itoa(val)
		default:
			envStringMap[k] = fmt.Sprintf("%v", val)
		}
	}

	if err := env.Write(envStringMap, install.GetEnvPath()); err != nil {
		return c.JSON(e.Error(buserr.WithDetail(constant.ErrAppsEnvUpdateFailed, err.Error())))
	}

	return c.JSON(e.Succ(map[string]interface{}{
		"msg": i18n.GetMsg(constant.ErrAppsPortConflictResolved),
	}))
}

func RepairPodmanShortName(c fiber.Ctx) error {
	if runtime.GOOS == "darwin" {
		out, err := service.RepairPodmanShortNameOnDarwin(context.Background())
		if err != nil {
			return c.JSON(e.Error(err))
		}
		return c.JSON(e.Succ(map[string]any{
			"output": strings.TrimSpace(out),
		}))
	}
	if runtime.GOOS != "linux" {
		return c.JSON(e.Error(buserr.New(constant.ErrAppsUnsupportedPlatform)))
	}
	resp, err := gpc.Do(context.Background(), "REPAIR_PODMAN_SHORT_NAME", map[string]interface{}{})
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unknown action") {
			return c.JSON(e.Error(buserr.New(constant.ErrAppsGpcOutdatedPodmanShortName)))
		}
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(map[string]any{
		"output": strings.TrimSpace(resp.Output),
	}))
}
