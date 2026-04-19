package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func ContainerRouter(r fiber.Router) {
	containerRouter := r.Group("container").
		Use(middleware.JWT(constant.UserRoleAdmin))
	{

		containerRouter.Get("/exec", websocket.New(api.ContainerWsSSH))
		containerRouter.Get("/stats/:id", api.ContainerStats)

		// 容器
		containerRouter.Post("/create", api.ContainerCreate)
		containerRouter.Post("/update", api.ContainerUpdate)
		containerRouter.Post("/upgrade", api.ContainerUpgrade)
		containerRouter.Post("/info", api.ContainerInfo)
		containerRouter.Post("/search", api.ContainerSearch)
		containerRouter.Post("/list", api.ContainerList)
		containerRouter.Get("/list/stats", api.ContainerListStats)
		containerRouter.Get("/search/log", websocket.New(api.ContainerLogs))
		containerRouter.Get("/limit", api.LoadResourceLimit)
		containerRouter.Post("/clean/log", api.ContainerCleanLog)
		containerRouter.Post("/load/log", api.ContainerLoadLog)
		containerRouter.Post("/inspect", api.ContainerInspect)
		containerRouter.Post("/rename", api.ContainerRename)
		containerRouter.Post("/commit", api.ContainerCommit)
		containerRouter.Post("/operate", api.ContainerOperation)
		containerRouter.Post("/prune", api.ContainerPrune)

		// 镜像源
		containerRouter.Get("/repo", api.ListRepo)
		containerRouter.Post("/repo/status", api.CheckRepoStatus)
		containerRouter.Post("/repo/search", api.SearchRepo)
		containerRouter.Post("/repo/update", api.UpdateRepo)
		containerRouter.Post("/repo", api.CreateRepo)
		containerRouter.Post("/repo/del", api.DeleteRepo)

		// 编排
		containerRouter.Post("/compose/search", api.SearchCompose)
		containerRouter.Post("/compose", api.CreateCompose)
		containerRouter.Post("/compose/test", api.TestCompose)
		containerRouter.Post("/compose/operate", api.OperatorCompose)
		containerRouter.Post("/compose/update", api.ComposeUpdate)
		containerRouter.Get("/compose/search/log", websocket.New(api.ComposeLogs))

		// 镜像
		containerRouter.Get("/image", api.ListImage)
		containerRouter.Get("/image/all", api.ListAllImage)
		containerRouter.Post("/image/search", api.SearchImage)
		containerRouter.Post("/image/pull", api.ImagePull)
		containerRouter.Post("/image/push", api.ImagePush)
		containerRouter.Post("/image/save", api.ImageSave)
		containerRouter.Post("/image/load", api.ImageLoad)
		containerRouter.Post("/image/remove", api.ImageRemove)
		containerRouter.Post("/image/tag", api.ImageTag)
		containerRouter.Post("/image/build", api.ImageBuild)

		// 网络
		containerRouter.Get("/network", api.ListNetwork)
		containerRouter.Post("/network/del", api.DeleteNetwork)
		containerRouter.Post("/network/search", api.SearchNetwork)
		containerRouter.Post("/network", api.CreateNetwork)

		// 卷
		containerRouter.Get("/volume", api.ListVolume)
		containerRouter.Post("/volume/del", api.DeleteVolume)
		containerRouter.Post("/volume/search", api.SearchVolume)
		containerRouter.Post("/volume", api.CreateVolume)
		// 配置
		containerRouter.Get("/instance/status", api.LoadDockerStatus)
		containerRouter.Post("/instance/operate", api.OperateDocker)
		// Docker 配置
		containerRouter.Get("/daemon/config", api.LoadDaemonJson)
		containerRouter.Get("/daemon/file", api.LoadDaemonJsonFile)
		containerRouter.Post("/daemon/update", api.UpdateDaemonJson)
		containerRouter.Post("/daemon/update/byfile", api.UpdateDaemonJsonByFile)
		containerRouter.Post("/logoption/update", api.UpdateLogOption)
		containerRouter.Post("/ipv6option/update", api.UpdateIpv6Option)

		containerRouter.Post("/download/log", api.DownloadContainerLogs)
	}
}
