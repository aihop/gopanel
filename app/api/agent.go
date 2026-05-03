package api

import (
	"bufio"
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/gpagent"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/gofiber/fiber/v3"
)

type agentStatus struct {
	Version          string `json:"version"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
	BaseDir          string `json:"base_dir"`
	SocketPath       string `json:"socket"`
	CaddyStatus      string `json:"caddy_status"`
	DaemonStatus     string `json:"daemon_status"`
	ManagedAppsCount int    `json:"managed_apps_count"`
	LastError        string `json:"last_error"`
}

func AgentStatus(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp := map[string]interface{}{
		"online": false,
		"socket": gpagent.SocketPath(),
	}

	r, err := gpagent.Do(ctx, "AGENT_STATUS", nil)
	if err != nil {
		resp["error"] = err.Error()
		return c.JSON(e.Succ(resp))
	}

	info, err := gpagent.DecodeOutput[agentStatus](r)
	if err != nil {
		resp["error"] = err.Error()
		return c.JSON(e.Succ(resp))
	}

	resp["online"] = true
	resp["agent"] = info
	return c.JSON(e.Succ(resp))
}

func AgentEnsure(c fiber.Ctx) error {
	logName := "gp_agent_ensure_" + time.Now().Format("20060102150405") + ".log"
	logger := service.GetUpdateLogger(logName)

	go func() {
		defer service.RemoveUpdateLogger(logName)
		writeLog := func(text string, param interface{}) {
			logger.Append(text, param)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		writeLog("check gp-agent ping", gpagent.SocketPath())
		if _, err := gpagent.Do(ctx, "PING", nil); err == nil {
			writeLog("gp-agent already online", "ok")
			logger.SetStatus("success")
			return
		}

		currentVersionInfo, err := appVersionService.GoPanelVersion()
		if err != nil {
			writeLog("get gopanel version error", err)
			logger.SetStatus("failed")
			return
		}

		baseUpgradeReq := dto.SettingUpgradeVersion{
			VersionName: currentVersionInfo.VersionName,
			VersionCode: currentVersionInfo.VersionCode,
			OS:          runtime.GOOS,
			Arch:        runtime.GOARCH,
			Lang:        "zh",
			AppBrand:    constant.AppBrand,
		}
		var updateInfo *dto.AppUpdateData
		for _, pkg := range []string{"gp-agent", ""} {
			req := baseUpgradeReq
			req.Package = pkg
			updateInfo, err = appVersionService.GetUpdateInfo(constant.UpgradeUrl, &req)
			if err != nil {
				writeLog("fetch upgrade info error", err)
				logger.SetStatus("failed")
				return
			}
			if updateInfo != nil && strings.TrimSpace(updateInfo.DownloadUrl) != "" {
				if pkg == "" {
					writeLog("gp-agent package fallback", "reuse main package")
				}
				break
			}
		}
		if updateInfo == nil || strings.TrimSpace(updateInfo.DownloadUrl) == "" {
			writeLog("invalid upgrade info", updateInfo)
			logger.SetStatus("failed")
			return
		}

		writeLog("ensure gp-agent via gpc", "start")
		out, err := gpc.Do(ctx, "GOPANEL_AGENT_ENSURE", map[string]interface{}{
			"download_url": updateInfo.DownloadUrl,
			"base_dir":     global.CONF.System.BaseDir,
			"service_name": gpAgentServiceName(),
		})
		if err != nil {
			writeLog("download url", updateInfo.DownloadUrl)
			writeLog("gpc ensure error", err)
			if out.Output != "" {
				writeLog("gpc ensure output", out.Output)
			}
			logger.SetStatus("failed")
			return
		}
		if strings.TrimSpace(out.Output) != "" {
			writeLog("gpc ensure output", out.Output)
		}

		writeLog("wait gp-agent online", gpagent.SocketPath())
		var pingErr error
		onlineDeadline := time.Now().Add(15 * time.Second)
	waitOnline:
		for {
			if _, err := gpagent.Do(ctx, "PING", nil); err == nil {
				pingErr = nil
				break
			} else {
				pingErr = err
			}
			if time.Now().After(onlineDeadline) {
				break
			}
			select {
			case <-ctx.Done():
				pingErr = ctx.Err()
				break waitOnline
			case <-time.After(500 * time.Millisecond):
			}
		}
		if pingErr != nil {
			writeLog("gp-agent still offline after ensure", pingErr)
			logger.SetStatus("failed")
			return
		}
		writeLog("gp-agent online", "ok")
		logger.SetStatus("success")
	}()

	res := struct {
		Log string `json:"log"`
	}{Log: logName}
	return c.JSON(e.Succ(res))
}

func AgentEnsureLogs(c fiber.Ctx) error {
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
