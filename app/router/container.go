package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func ContainerRouter(r fiber.Router) {
	dockerRouter := r.Group("container").
		Use(middleware.JWT(constant.UserRoleAdmin))
	{

		dockerRouter.Get("/exec", websocket.New(api.ContainerWsSSH))
		dockerRouter.Get("/stats/:id", api.ContainerStats)

		// 容器
		dockerRouter.Post("/create", api.ContainerCreate)
		dockerRouter.Post("/update", api.ContainerUpdate)
		dockerRouter.Post("/upgrade", api.ContainerUpgrade)
		dockerRouter.Post("/info", api.ContainerInfo)
		dockerRouter.Post("/search", api.ContainerSearch)
		dockerRouter.Post("/list", api.ContainerList)
		dockerRouter.Get("/list/stats", api.ContainerListStats)
		dockerRouter.Get("/search/log", websocket.New(api.ContainerLogs))
		dockerRouter.Get("/limit", api.LoadResourceLimit)
		dockerRouter.Post("/clean/log", api.ContainerCleanLog)
		dockerRouter.Post("/load/log", api.ContainerLoadLog)
		dockerRouter.Post("/inspect", api.ContainerInspect)
		dockerRouter.Post("/rename", api.ContainerRename)
		dockerRouter.Post("/commit", api.ContainerCommit)
		dockerRouter.Post("/operate", api.ContainerOperation)
		dockerRouter.Post("/prune", api.ContainerPrune)

		// 镜像源
		dockerRouter.Get("/repo", api.ListRepo)
		dockerRouter.Post("/repo/status", api.CheckRepoStatus)
		dockerRouter.Post("/repo/search", api.SearchRepo)
		dockerRouter.Post("/repo/update", api.UpdateRepo)
		dockerRouter.Post("/repo", api.CreateRepo)
		dockerRouter.Post("/repo/del", api.DeleteRepo)

		// 编排
		dockerRouter.Post("/compose/search", api.SearchCompose)
		dockerRouter.Post("/compose", api.CreateCompose)
		dockerRouter.Post("/compose/test", api.TestCompose)
		dockerRouter.Post("/compose/operate", api.OperatorCompose)
		dockerRouter.Post("/compose/update", api.ComposeUpdate)
		dockerRouter.Get("/compose/search/log", websocket.New(api.ComposeLogs))

		// 镜像
		dockerRouter.Get("/image", api.ListImage)
		dockerRouter.Get("/image/all", api.ListAllImage)
		dockerRouter.Post("/image/search", api.SearchImage)
		dockerRouter.Post("/image/pull", api.ImagePull)
		dockerRouter.Post("/image/push", api.ImagePush)
		dockerRouter.Post("/image/save", api.ImageSave)
		dockerRouter.Post("/image/load", api.ImageLoad)
		dockerRouter.Post("/image/remove", api.ImageRemove)
		dockerRouter.Post("/image/tag", api.ImageTag)
		dockerRouter.Post("/image/build", api.ImageBuild)

		// 网络
		dockerRouter.Get("/network", api.ListNetwork)
		dockerRouter.Post("/network/del", api.DeleteNetwork)
		dockerRouter.Post("/network/search", api.SearchNetwork)
		dockerRouter.Post("/network", api.CreateNetwork)

		// 卷
		dockerRouter.Get("/volume", api.ListVolume)
		dockerRouter.Post("/volume/del", api.DeleteVolume)
		dockerRouter.Post("/volume/search", api.SearchVolume)
		dockerRouter.Post("/volume", api.CreateVolume)

		// 配置
		dockerRouter.Get("/docker/status", api.LoadDockerStatus)
		dockerRouter.Get("/daemonjson", api.LoadDaemonJson)
		dockerRouter.Get("/daemonjson/file", api.LoadDaemonJsonFile)
		dockerRouter.Post("/daemonjson/update", api.UpdateDaemonJson)
		dockerRouter.Post("/daemonjson/update/byfile", api.UpdateDaemonJsonByFile)
		dockerRouter.Post("/docker/operate", api.OperateDocker)
		dockerRouter.Post("/logoption/update", api.UpdateLogOption)
		dockerRouter.Post("/ipv6option/update", api.UpdateIpv6Option)

		dockerRouter.Post("/download/log", api.DownloadContainerLogs)
	}
}
