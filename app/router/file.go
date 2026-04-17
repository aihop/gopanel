package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func FileRouter(r fiber.Router) {
	fileRouter := r.Group("file").
		Use(middleware.JWT(constant.UserRoleSubAdmin)) // 放宽到 SUB_ADMIN

	{
		fileRouter.Get("/download", middleware.Session(), api.Download)
		fileRouter.Post("/search", api.ListFiles)
		fileRouter.Post("/dirExist", api.DirExist)
		fileRouter.Post("/create", api.CreateFile)
		fileRouter.Post("/del", api.DeleteFile)
		fileRouter.Post("/batch/del", api.BatchDeleteFile)
		fileRouter.Post("/mode", api.ChangeFileMode)
		fileRouter.Post("/owner", api.ChangeFileOwner)
		fileRouter.Post("/compress", api.CompressFile)
		fileRouter.Post("/decompress", api.DeCompressFile)
		fileRouter.Post("/content", api.GetContent)
		fileRouter.Post("/save", api.SaveContent)
		fileRouter.Post("/check", api.CheckFile)
		fileRouter.Post("/batch/check", api.BatchCheckFiles)
		fileRouter.Post("/upload", api.UploadFiles)
		fileRouter.Post("/chunkUpload", api.UploadChunkFiles)
		fileRouter.Post("/rename", api.ChangeFileName)
		fileRouter.Post("/wget", api.WgetFile)
		fileRouter.Post("/move", api.MoveFile)
		fileRouter.Post("/chunkDownload", api.DownloadChunkFiles)
		fileRouter.Post("/size", api.Size)
		fileRouter.Get("/ws", websocket.New(api.Ws))
		fileRouter.Get("/keys", api.Keys)
		fileRouter.Post("/read", api.ReadFileByLine)
		fileRouter.Post("/batch/role", api.BatchChangeModeAndOwner)

		fileRouter.Post("/recycle/status", api.FileRecycleStatus)

		fileRouter.Post("/upload/search", api.FileUploadSearch)
	}
}
