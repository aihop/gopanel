package api

import (
	"errors"
	"path"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func BackupHandle(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.CommonBackup](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	backupService := service.NewBackup()

	switch req.Type {
	case "mysql", "mariadb":
		if err := backupService.MysqlBackup(req); err != nil {
			return c.JSON(e.Fail(err))
		}
	case constant.AppPostgresql:
		if err := backupService.PostgresqlBackup(req); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	return c.JSON(e.Succ())
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
