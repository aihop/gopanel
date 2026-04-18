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
	logger.Appendf("backup submitted: type=%s name=%s detail=%s detailId=%d", req.Type, req.Name, req.DetailName, req.DetailId)

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
			runErr = fmt.Errorf("unsupported backup type: %s", req.Type)
		}

		if runErr != nil {
			logger.Appendf("backup failed: %v", runErr)
			logger.SetStatus("failed")
			return
		}
		logger.AppendLine("backup completed")
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
	backupService := service.NewBackup()

	downloadPath, err := backupService.DownloadRecord(dto.DownloadRecord{Source: req.Source, FileDir: path.Dir(req.File), FileName: path.Base(req.File)})
	if err != nil {
		return c.JSON(e.Fail(errors.New("download file failed, err: " + err.Error())))
	}
	req.File = downloadPath
	switch req.Type {
	case "mysql", "mariadb":
		if err := backupService.MysqlRecover(req); err != nil {
			return c.JSON(e.Fail(err))
		}
	case constant.AppPostgresql:
		if err := backupService.PostgresqlRecover(req); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	return c.JSON(e.Succ())
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
	backupService := service.NewBackup()
	switch req.Type {
	case "mysql", "mariadb":
		if err := backupService.MysqlRecoverByUpload(req); err != nil {
			return c.JSON(e.Fail(err))
		}
	case constant.AppPostgresql:
		if err := backupService.PostgresqlRecoverByUpload(req); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	return c.JSON(e.Succ())
}
