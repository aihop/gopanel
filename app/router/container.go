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
		containerRouter.Get("/stats/:id", api.ContainerStatsID)
		// 容器
		containerRouter.Post("/create", api.ContainerCreate)
		containerRouter.Post("/update", api.ContainerUpdate)
		containerRouter.Post("/upgrade", api.ContainerUpgrade)
		containerRouter.Post("/info", api.ContainerInfo)
		containerRouter.Post("/list", api.ContainerList)
		containerRouter.Post("/all", api.ContainerAll)
		containerRouter.Get("/stats", api.ContainerListStats)
		containerRouter.Get("/logs", websocket.New(api.ContainerLogs))
		containerRouter.Get("/limit", api.LoadResourceLimit)
		containerRouter.Post("/clean/logs", api.ContainerCleanLogs)
		containerRouter.Post("/load/logs", api.ContainerLoadLogs)
		containerRouter.Post("/inspect", api.ContainerInspect)
		containerRouter.Post("/rename", api.ContainerRename)
		containerRouter.Post("/commit", api.ContainerCommit)
		containerRouter.Post("/operate", api.ContainerOperation)
		containerRouter.Post("/prune", api.ContainerPrune)

		// 镜像源
		containerRouter.Get("/repo", api.ListRepo)
		containerRouter.Post("/repo/status", api.CheckRepoStatus)
		containerRouter.Post("/repo/list", api.ContainerRepoList)
		containerRouter.Post("/repo/update", api.UpdateRepo)
		containerRouter.Post("/repo", api.CreateRepo)
		containerRouter.Post("/repo/del", api.DeleteRepo)

		// 编排
		containerRouter.Post("/compose/list", api.ContainerComposeList)
		containerRouter.Post("/compose", api.CreateCompose)
		containerRouter.Post("/compose/test", api.TestCompose)
		containerRouter.Post("/compose/operate", api.OperatorCompose)
		containerRouter.Post("/compose/update", api.ComposeUpdate)
		containerRouter.Get("/compose/logs", websocket.New(api.ComposeLogs))

		// 镜像
		containerRouter.Get("/image", api.ListImage)
		containerRouter.Get("/image/all", api.ListAllImage)
		containerRouter.Post("/image/list", api.ImageList)
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
		containerRouter.Post("/network/list", api.ContainerNetworkList)
		containerRouter.Post("/network", api.CreateNetwork)

		// 卷
		containerRouter.Get("/volume", api.ListVolume)
		containerRouter.Post("/volume/del", api.DeleteVolume)
		containerRouter.Post("/volume/list", api.ContainerVolumeList)
		containerRouter.Post("/volume", api.CreateVolume)
		// 配置
		containerRouter.Post("/repair/podman-socket", api.ContainerRepairPodmanSocket)
		containerRouter.Post("/repair/linger", api.ContainerRepairSystemdLinger)
		containerRouter.Get("/engine/status", api.ContainerEngineStatus)
		containerRouter.Get("/engine/validate", api.ContainerEngineValidate)
		containerRouter.Post("/engine/operate", api.ContainerEngineOperate)
		// Docker 配置
		containerRouter.Get("/engine/config", api.ContainerEngineConfig)
		containerRouter.Get("/engine/file", api.ContainerEngineFile)
		containerRouter.Post("/engine/update", api.ContainerEngineUpdate)
		containerRouter.Post("/engine/update-file", api.ContainerEngineUpdateFile)
		containerRouter.Post("/options/log", api.ContainerOptionsLog)
		containerRouter.Post("/options/ipv6", api.ContainerOptionsIpv6)

		containerRouter.Post("/download/logs", api.DownloadContainerLogs)
	}
}
