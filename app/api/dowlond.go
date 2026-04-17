package api

import (
	"encoding/json"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
	"github.com/gofiber/fiber/v3"
)

func DownloadFile(c fiber.Ctx) error {
	downloadUrl := ""

	tmpFolder := global.CONF.System.TmpDir + "/" + common.RandStr(10)
	suffix := ".tar.gz"
	tmpFile := common.RandStr(10) + suffix

	if err := files.NewFileOp().CreateDir(tmpFolder, 0755); err != nil {
		return c.JSON(e.Fail(err))
	}

	wget := request.FileWget{
		Url:               downloadUrl,
		Path:              tmpFolder,
		Name:              tmpFile,
		IgnoreCertificate: false,
	}

	key, err := service.NewIFileService().Wget(wget)
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	res := response.FileDownloadRes{
		Path: tmpFolder,
		Name: tmpFile,
		Key:  key,
	}
	return c.JSON(e.Succ(res))
}

func CheckDownloadProgress(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileProcessReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	progress, err := global.CACHE.Get(req.Key)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var p files.Process
	json.Unmarshal(progress, &p)
	return c.JSON(e.Succ(p))
}

func DownloadAfter(c fiber.Ctx) error {
	// req, err := e.BodyToStruct[response.FileDownloadRes](c.Body())
	// if err != nil {
	// 	return c.JSON(e.Fail(err))
	// }

	return c.JSON(e.Succ(nil))
}
