package api

import (
	"bufio"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/common"
	"github.com/gofiber/fiber/v3"
)

func BackupHandle(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.CommonBackup](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	key := "backup_" + common.RandStrAndNum(20)
	logger := service.GetBackupLogger(key)
	logger.Appendf("已提交备份任务：类型=%s，实例=%s，数据=%s，ID=%d", req.Type, req.Name, req.DetailName, req.DetailId)

	go func() {
		defer func() {
			service.RemoveBackupLogger(key)
		}()

		backupService := service.NewBackup()
		var runErr error
		switch req.Type {
		case "mysql", "mariadb":
			runErr = backupService.MysqlBackup(req, logger)
		case constant.AppPostgresql:
			runErr = backupService.PostgresqlBackup(req, logger)
		default:
			runErr = fmt.Errorf("暂不支持的备份类型：%s", req.Type)
		}

		if runErr != nil {
			logger.Appendf("备份失败：%v", runErr)
			logger.SetStatus("failed")
			return
		}
		logger.AppendLine("备份完成")
		logger.SetStatus("success")
	}()

	return c.JSON(e.Succ(map[string]interface{}{"key": key}))
}

func BackupLogsStream(c fiber.Ctx) error {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		return c.JSON(e.Fail(errors.New("key is required")))
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.RequestCtx().SetBodyStreamWriter(func(w *bufio.Writer) {
		writeData := func(data string) {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(data, "\n", " "))
			_ = w.Flush()
		}
		writeEvent := func(event, data string) {
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, strings.ReplaceAll(data, "\n", " "))
			_ = w.Flush()
		}

		if !service.IsBackupLoggerActive(key) {
			lines, err := service.ReadBackupLogFromFile(key)
			if err == nil {
				for _, line := range lines {
					writeData(line)
				}
			}
			writeEvent("eof", "EOF")
			return
		}

		logger := service.GetBackupLogger(key)
		for _, line := range logger.GetLogs() {
			writeData(line)
		}
		writeEvent("status", logger.GetStatus())

		ch := logger.Subscribe()
		defer logger.Unsubscribe(ch)

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.Context().Done():
				return
			case <-ticker.C:
				_, _ = fmt.Fprintf(w, "event: ping\ndata: ping\n\n")
				_ = w.Flush()
			case evt, ok := <-ch:
				if !ok {
					return
				}
				switch evt.Type {
				case "log":
					writeData(evt.Message)
				case "status":
					writeEvent("status", evt.Status)
				case "eof":
					writeEvent("eof", "EOF")
					return
				default:
					if evt.Message != "" {
						writeData(evt.Message)
					}
				}
			}
		}
	})

	return nil
}

func BackupRecover(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.CommonRecover](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	key := "recover_" + common.RandStrAndNum(20)
	logger := service.GetBackupLogger(key)
	logger.Appendf("已提交恢复任务：类型=%s，实例=%s，数据=%s，文件=%s", req.Type, req.Name, req.DetailName, req.File)

	go func() {
		defer func() {
			service.RemoveBackupLogger(key)
		}()

		backupService := service.NewBackup()
		logger.AppendLine("正在下载备份文件...")
		downloadPath, err := backupService.DownloadRecord(dto.DownloadRecord{Source: req.Source, FileDir: path.Dir(req.File), FileName: path.Base(req.File)})
		if err != nil {
			logger.Appendf("下载备份文件失败：%v", err)
			logger.SetStatus("failed")
			return
		}

		req.File = downloadPath
		var runErr error
		switch req.Type {
		case "mysql", "mariadb":
			runErr = backupService.MysqlRecover(req, logger)
		case constant.AppPostgresql:
			runErr = backupService.PostgresqlRecover(req, logger)
		default:
			runErr = fmt.Errorf("暂不支持的恢复类型：%s", req.Type)
		}
		if runErr != nil {
			logger.Appendf("恢复失败：%v", runErr)
			logger.SetStatus("failed")
			return
		}
		logger.AppendLine("恢复完成")
		logger.SetStatus("success")
	}()

	return c.JSON(e.Succ(map[string]interface{}{"key": key}))
}

// @Tags Backup Account
// @Summary Recover system data by upload
// @Accept json
// @Param request body dto.CommonRecover true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /backup/recover/byupload [post]
// @x-panel-log {"bodyKeys":["type","name","detailName","file"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"从 [file] 恢复 [type] 数据 [name][detailName]","formatEN":"recover [type] data [name][detailName] from [file]"}
func BackupRecoverByUpload(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.CommonRecover](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	key := "recover_" + common.RandStrAndNum(20)
	logger := service.GetBackupLogger(key)
	logger.Appendf("已提交上传恢复任务：类型=%s，实例=%s，数据=%s，文件=%s", req.Type, req.Name, req.DetailName, req.File)

	go func() {
		defer func() {
			service.RemoveBackupLogger(key)
		}()

		backupService := service.NewBackup()
		var runErr error
		switch req.Type {
		case "mysql", "mariadb":
			runErr = backupService.MysqlRecoverByUpload(req, logger)
		case constant.AppPostgresql:
			runErr = backupService.PostgresqlRecoverByUpload(req, logger)
		default:
			runErr = fmt.Errorf("暂不支持的恢复类型：%s", req.Type)
		}
		if runErr != nil {
			logger.Appendf("恢复失败：%v", runErr)
			logger.SetStatus("failed")
			return
		}
		logger.AppendLine("恢复完成")
		logger.SetStatus("success")
	}()

	return c.JSON(e.Succ(map[string]interface{}{"key": key}))
}
