package api

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

func BackupRecordSearch(c fiber.Ctx) error {
	R, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	data, err := service.NewBackupRecord().Search(&R)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(data))
}

func BackupRecordCount(c fiber.Ctx) error {
	R, err := e.BodyToWhere(c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	data, err := service.NewBackupRecord().CountByWhere(&R)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(data))
}

func BackupRecordSize(c fiber.Ctx) error {
	R, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	data, err := service.NewBackupRecord().Sizes(&R)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(data))
}

func BackupRecordDeletes(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.Ids](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewBackupRecord().DeleteByIds(R.Ids)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}

func BackupRecordDownload(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.DownloadRecord](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	backupService := service.NewBackup()
	filePath, err := backupService.DownloadRecord(*R)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(filePath))
}
