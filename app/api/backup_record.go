package api

import (
	"errors"
	"strconv"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/gofiber/fiber/v3"
)

func BackupRecordList(c fiber.Ctx) error {
	R, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	data, err := service.NewBackupRecord().Search(&R)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	total, _ := service.NewBackupRecord().CountByWhere(&gormx.Wherex{
		Wheres: R.Wheres,
	})
	return c.JSON(e.Succ(dto.PageResult{
		Items: data,
		Total: total,
	}))
}

func BackupRecordSize(c fiber.Ctx) error {
	R, err := e.BodyToStruct[struct {
		ID uint `json:"id" validate:"required"`
	}](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	data, err := service.NewBackupRecord().SizeByID(R.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
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

func BackupRecordDownloadDirect(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil || id == 0 {
		return c.JSON(e.Fail(errors.New("invalid backup record id")))
	}
	filePath, fileName, err := service.NewBackup().DownloadRecordByID(uint(id))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	c.Set(fiber.HeaderContentType, "application/octet-stream")
	return c.Download(filePath, fileName)
}
