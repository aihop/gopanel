package api

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/common"
	"github.com/gofiber/fiber/v3"
)

type SettingApiTokenReq struct {
	ApiInterfaceStatus string `json:"apiInterfaceStatus"`
	ApiKey             string `json:"apiKey"`
}

// 更新API Token配置
func SettingSystemApiTokenUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[SettingApiTokenReq](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}

	updateConfYamlFile(map[string]interface{}{
		"system.api_interface_status": req.ApiInterfaceStatus,
		"system.api_key":              req.ApiKey,
	})

	global.CONF.System.ApiInterfaceStatus = req.ApiInterfaceStatus
	global.CONF.System.ApiKey = req.ApiKey

	return c.JSON(e.Succ("API Token settings updated"))
}

func SettingSystemBaseDir(c fiber.Ctx) error {
	return c.JSON(e.Succ(global.CONF.System.BaseDir))
}

// 清理临时文件目录
func SettingSystemClearDir(c fiber.Ctx) error {
	req, err := e.BodyToStruct[SettingClearDirReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var dir string
	switch req.Key {
	case "tmp":
		dir = global.CONF.System.TmpDir
	case "log":
		dir = global.CONF.System.LogPath
	case "cache":
		dir = global.CONF.System.Cache
	default:
		return c.JSON(e.Fail(fmt.Errorf("invalid key")))
	}
	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return c.JSON(e.Fail(fmt.Errorf("directory %s does not exist", dir)))
	}

	// 清理目录中的内容
	result, err := clearFilesWithStats(dir)
	if err != nil {
		return c.JSON(e.Fail(fmt.Errorf("failed to clear directory %s: %v", dir, err)))
	}

	global.LOG.Info("Cache cleared successfully")

	// 返回成功消息
	return c.JSON(e.Succ(result))
}

func SettingSystemRestart(c fiber.Ctx) error {
	operation := strings.TrimSpace(c.Params("operation"))
	if operation == "" {
		operation = "panel"
	}

	var err error
	switch operation {
	case "panel":
		err = cmd.RestartGoPanel()
	case "server":
		err = cmd.RestartServer()
	default:
		err = fmt.Errorf("unsupported restart operation: %s", operation)
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}

// 设置系统运行端口
// 1. 接收端口参数
// 2. 验证端口是否被占用
// 3. 更新配置文件
// 4. 返回成功消息
// 4. 然后重启系统
func SettingSystemPort(c fiber.Ctx) error {
	req, err := e.BodyToStruct[SettingPortReq](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}

	// 检查端口占用
	pid, err := processService.CheckProcessPort(req.ServerPort)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if pid != 0 {
		return c.JSON(e.Fail(fmt.Errorf("port %d is already in use by process %d", req.ServerPort, pid)))
	}

	// 更新端口配置
	updateConfYamlFile(map[string]interface{}{
		"system.port": req.ServerPort,
		"http.listen": fmt.Sprintf(":%d", req.ServerPort),
		"rpc.listen":  fmt.Sprintf(":%d", req.ServerPort+1),
	})

	global.CONF.System.Port = fmt.Sprintf("%d", req.ServerPort)

	return c.JSON(e.Succ(fmt.Sprintf("System port updated to %d", req.ServerPort)))
}

// 获取系统设置
func SettingSystemConfig(c fiber.Ctx) error {
	return c.JSON(e.Succ(global.CONF))
}

// 更新安全入口
func SettingSystemEntrance(c fiber.Ctx) error {
	req, err := e.BodyToStruct[SettingEntranceReq](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	req.Entrance = strings.TrimSpace(req.Entrance)
	if req.Entrance == "" {
		return c.JSON(e.Fail(fmt.Errorf("entrance cannot be empty")))
	}

	// 定义已存在的路由组前缀，防止冲突
	existingGroups := []string{"api", "web"}
	// 检查是否与现有路由组冲突
	for _, group := range existingGroups {
		if req.Entrance == group {
			return c.JSON(e.Fail(fmt.Errorf("entrance '%s' conflicts with an existing route group", req.Entrance)))
		}
	}

	updateConfYamlFile(map[string]interface{}{
		"system.entrance": req.Entrance,
	})

	// 返回当前配置
	return c.JSON(e.Succ())
}

// 获取当前版本信息参数
func SettingSystemVersion(c fiber.Ctx) error {
	version, err := appVersionService.GoPanelVersion()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(version))
}

// 系统更新
func SettingSystemUpgrade(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SettingUpgradeReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	// 获取安装位置
	installPath := global.CONF.System.BaseDir

	// 获取当前系统、架构
	runtimeOS := runtime.GOOS
	runtimeArch := runtime.GOARCH

	currentVersionInfo, _ := appVersionService.GoPanelVersion()
	// 获取最新版本信息
	updateInfo, err := appVersionService.GetUpdateInfo(constant.UpgradeUrl, &dto.SettingUpgradeVersion{
		VersionName: req.CurrentVersion,
		VersionCode: currentVersionInfo.VersionCode,
		OS:          runtimeOS,
		Arch:        runtimeArch,
		Lang:        req.Lang,
		AppBrand:    constant.AppBrand,
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	// 创建日志文件记录安装过程
	installLogDir := path.Join(global.CONF.System.TmpDir, "install_logs")
	if _, err = os.Stat(installLogDir); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(installLogDir, os.ModePerm); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	logName := "gopanel_update_" + time.Now().Format("20060102150405") + ".log"
	logger := service.GetUpdateLogger(logName)
	go func() {
		defer service.RemoveUpdateLogger(logName)
		writeLog := func(text string, param interface{}) {
			logger.Append(text, param)
		}

		if common.CompareVersionV2(updateInfo.LatestVersionName, req.CurrentVersion) <= 0 {
			writeLog("skip older or same as current", updateInfo.LatestVersionName)
			logger.SetStatus("success")
			return
		}

		writeLog("start upload", updateInfo.LatestVersionName)
		// 开始更新
		err = appVersionService.GoPanelUpload(updateInfo.DownloadUrl, installPath, updateInfo.LatestVersionCode, writeLog)
		if err != nil {
			writeLog("upload error", err)
			logger.SetStatus("failed")
			return
		}
		writeLog("upload success", updateInfo.LatestVersionName)
		logger.SetStatus("success")
	}()

	// 返回异步任务的日志文件名给前端
	res := struct {
		Log string `json:"log"`
	}{Log: logName}
	return c.JSON(e.Succ(res))
}

func SettingSystemUpgradeLogs(c fiber.Ctx) error {
	logName := strings.TrimSpace(c.Query("log"))
	if logName == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid log name")
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Status(200)

	writeData := func(w *bufio.Writer, line string) error {
		_, err := fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(line, "\n", " "))
		if err == nil {
			err = w.Flush()
		}
		return err
	}
	writeStatus := func(w *bufio.Writer, status string) error {
		_, err := fmt.Fprintf(w, "event: status\ndata: %s\n\n", status)
		if err == nil {
			err = w.Flush()
		}
		return err
	}

	ctxRaw := c.RequestCtx()
	ctxRaw.SetBodyStreamWriter(func(w *bufio.Writer) {
		if !service.IsUpdateLoggerActive(logName) {
			lines, err := service.ReadUpdateLogFromFile(logName)
			if err == nil {
				if len(lines) > 3000 {
					_ = writeData(w, fmt.Sprintf("... 之前的日志已折叠，总计 %d 行，这里只显示最新 2000 行 ...", len(lines)))
					lines = lines[len(lines)-2000:]
				}
				for _, line := range lines {
					if err := writeData(w, line); err != nil {
						return
					}
				}
				_ = writeStatus(w, service.InferUpdateLogStatus(lines))
			}
			_, _ = fmt.Fprintf(w, "event: eof\ndata: EOF\n\n")
			_ = w.Flush()
			return
		}

		logger := service.GetUpdateLogger(logName)
		logs := logger.GetLogs()
		if len(logs) > 3000 {
			_ = writeData(w, fmt.Sprintf("... 之前的实时日志已折叠，总计 %d 行，这里只显示最新 2000 行 ...", len(logs)))
			logs = logs[len(logs)-2000:]
		}
		for _, line := range logs {
			if err := writeData(w, line); err != nil {
				return
			}
		}
		_ = writeStatus(w, logger.GetStatus())

		ch := logger.Subscribe()
		defer logger.Unsubscribe(ch)

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case event, ok := <-ch:
				if !ok || event.Type == "eof" {
					_, _ = fmt.Fprintf(w, "event: eof\ndata: EOF\n\n")
					_ = w.Flush()
					return
				}
				if event.Type == "status" {
					if err := writeStatus(w, event.Status); err != nil {
						return
					}
					continue
				}
				if err := writeData(w, event.Message); err != nil {
					return
				}
			case <-ticker.C:
				if _, err := fmt.Fprintf(w, "event: ping\ndata: ping\n\n"); err != nil {
					return
				}
				_ = w.Flush()
			}
		}
	})

	return nil
}

// 检查更新
func SettingSystemCheck(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.Lang](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	// 1. 获取当前版本信息
	currentVersionInfo, err := appVersionService.GoPanelVersion()
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	// 获取当前系统、架构
	runtimeOS := runtime.GOOS
	runtimeArch := runtime.GOARCH

	// 2. 获取最新版本信息
	updateInfo, err := appVersionService.GetUpdateInfo(constant.UpgradeUrl, &dto.SettingUpgradeVersion{
		VersionName: currentVersionInfo.VersionName,
		VersionCode: currentVersionInfo.VersionCode,
		OS:          runtimeOS,
		Arch:        runtimeArch,
		Lang:        R.Lang,
		AppBrand:    constant.AppBrand,
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(updateInfo))
}
