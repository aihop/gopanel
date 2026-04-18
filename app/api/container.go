package api

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
)

func ContainerSearch(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.PageContainer](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	total, list, err := service.NewIContainerService().Page(R)
	if err != nil {
		return c.JSON(e.Error(buserr.Err(err)))
	}
	return c.JSON(e.Succ(dto.PageResult{
		Items: list,
		Total: total,
	}))
}

func ContainerList(c fiber.Ctx) error {
	list, err := service.NewIContainerService().List()
	if err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(list))
}

func ContainerUpdate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerOperate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().ContainerUpdate(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

// @Tags Container
// @Summary Load container info
// @Accept json
// @Param request body dto.OperationWithName true "request"
// @Success 200 {object} dto.ContainerOperate
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/info [post]
func ContainerInfo(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.OperationWithName](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	data, err := service.NewIContainerService().ContainerInfo(R)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(data))
}

// @Summary Load container limits
// @Success 200 {object} dto.ResourceLimit
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/limit [get]
func LoadResourceLimit(c fiber.Ctx) error {
	data, err := service.NewIContainerService().LoadResourceLimit()
	if err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(data))
}

// @Summary Load container stats
// @Success 200 {array} dto.ContainerListStats
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/list/stats [get]
func ContainerListStats(c fiber.Ctx) error {
	data, err := service.NewIContainerService().ContainerListStats()
	if err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(data))
}

// @Tags Container
// @Summary Create container
// @Accept json
// @Param request body dto.ContainerOperate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/create [post]
// @x-panel-log {"bodyKeys":["name","image"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"创建容器 [name][image]","formatEN":"create container [name][image]"}
func ContainerCreate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerOperate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	if err := service.NewIContainerService().ContainerCreate(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}

// @Tags Container
// @Summary Upgrade container
// @Accept json
// @Param request body dto.ContainerUpgrade true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/upgrade [post]
// @x-panel-log {"bodyKeys":["name","image"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新容器镜像 [name][image]","formatEN":"upgrade container image [name][image]"}
func ContainerUpgrade(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerUpgrade](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	if err := service.NewIContainerService().ContainerUpgrade(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}

// @Tags Container
// @Summary Clean container
// @Accept json
// @Param request body dto.ContainerPrune true "request"
// @Success 200 {object} dto.ContainerPruneReport
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/prune [post]
// @x-panel-log {"bodyKeys":["pruneType"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"清理容器 [pruneType]","formatEN":"clean container [pruneType]"}
func ContainerPrune(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerPrune](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	report, err := service.NewIContainerService().Prune(R)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(report))
}

// @Tags Container
// @Summary Clean container log
// @Accept json
// @Param request body dto.OperationWithName true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/clean/log [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"清理容器 [name] 日志","formatEN":"clean container [name] logs"}
func ContainerCleanLog(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.OperationWithName](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	if err := service.NewIContainerService().ContainerLogClean(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

// @Tags Container
// @Summary Load container log
// @Accept json
// @Param request body dto.OperationWithNameAndType true "request"
// @Success 200 {string} content
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/load/log [post]
func ContainerLoadLog(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.OperationWithNameAndType](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	content := service.NewIContainerService().LoadContainerLogs(R)
	return c.JSON(e.Succ(content))
}

// @Tags Container
// @Summary Rename Container
// @Accept json
// @Param request body dto.ContainerRename true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/rename [post]
// @x-panel-log {"bodyKeys":["name","newName"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"容器重命名 [name] => [newName]","formatEN":"rename container [name] => [newName]"}
func ContainerRename(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerRename](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().ContainerRename(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}

// @Tags Container
// @Summary Commit Container
// @Accept json
// @Param request body dto.ContainerCommit true "request"
// @Success 200
// @Router /container/commit [post]
func ContainerCommit(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerCommit](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().ContainerCommit(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}

// @Tags Container
// @Summary Operate Container
// @Accept json
// @Param request body dto.ContainerOperation true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/operate [post]
// @x-panel-log {"bodyKeys":["names","operation"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"容器 [names] 执行 [operation]","formatEN":"container [operation] [names]"}
func ContainerOperation(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerOperation](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().ContainerOperation(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}

// @Tags Container
// @Summary Container stats
// @Param id path string true "container id"
// @Success 200 {object} dto.ContainerStats
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/stats/{id} [get]
func ContainerStats(c fiber.Ctx) error {
	containerID := c.Params("id")
	fmt.Print(containerID)
	if containerID == "" {
		return c.JSON(e.Fail(buserr.Err(errors.New("error container id in path"))))
	}

	result, err := service.NewIContainerService().ContainerStats(containerID)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(result))
}

// @Tags Container
// @Summary Container inspect
// @Accept json
// @Param request body dto.InspectReq true "request"
// @Success 200 {string} result
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/inspect [post]
func ContainerInspect(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.InspectReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	result, err := service.NewIContainerService().Inspect(R)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(result))
}

// @Tags Container
// @Summary Container logs
// @Param container query string false "container name"
// @Param since query string false "since"
// @Param follow query string false "follow"
// @Param tail query string false "tail"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/search/log [get]
func ContainerLogs(c *websocket.Conn) {
	defer c.Close()
	// 提取查询参数
	container := c.Query("container")
	since := c.Query("since")
	follow := c.Query("follow") == "true"
	tail := c.Query("tail")

	// 获取容器日志
	if err := containerService.ContainerLogs(c, "container", container, since, tail, follow); err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
	}
}

// @Tags Container
// @Summary Download Container logs
// @Accept json
// @Param request body dto.ContainerLog true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/download/log [post]
func DownloadContainerLogs(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerLog](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	info, err := service.NewIContainerService().DownloadContainerLogs(R.ContainerType, R.Container, R.Since, strconv.Itoa(int(R.Tail)))
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	defer os.Remove(info)
	return c.Download(info)
}

// @Tags Container Network
// @Summary Page networks
// @Accept json
// @Param request body dto.SearchWithPage true "request"
// @Produce json
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/network/search [post]
func SearchNetwork(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.SearchWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}

	total, list, err := service.NewIContainerService().PageNetwork(R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(dto.PageResult{
		Items: list,
		Total: total,
	}))
}

// @Tags Container Network
// @Summary List networks
// @Accept json
// @Produce json
// @Success 200 {array} dto.Options
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/network [get]
func ListNetwork(c fiber.Ctx) error {
	list, err := service.NewIContainerService().ListNetwork()
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(list))
}

// @Tags Container Network
// @Summary Delete network
// @Accept json
// @Param request body dto.BatchDelete true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/network/del [post]
// @x-panel-log {"bodyKeys":["names"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"删除容器网络 [names]","formatEN":"delete container network [names]"}
func DeleteNetwork(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.BatchDelete](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().DeleteNetwork(R); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err))
	}
	return c.JSON(e.Succ())
}

// @Tags Container Network
// @Summary Create network
// @Accept json
// @Param request body dto.NetworkCreate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/network [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"创建容器网络 name","formatEN":"create container network [name]"}
func CreateNetwork(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.NetworkCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().CreateNetwork(R); err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}

// @Tags Container Volume
// @Summary Page Container Volumes
// @Accept json
// @Param request body dto.SearchWithPage true "request"
// @Produce json
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/volume/search [post]
func SearchVolume(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.SearchWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	total, list, err := service.NewIContainerService().PageVolume(R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(dto.PageResult{
		Items: list,
		Total: total,
	}))
}

// @Tags Container Volume
// @Summary List Container Volumes
// @Accept json
// @Produce json
// @Success 200 {array} dto.Options
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/volume [get]
func ListVolume(c fiber.Ctx) error {
	list, err := service.NewIContainerService().ListVolume()
	if err != nil {
		return c.JSON(e.Result(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(list))
}

// @Tags Container Volume
// @Summary Delete Container Volume
// @Accept json
// @Param request body dto.BatchDelete true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/volume/del [post]
// @x-panel-log {"bodyKeys":["names"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"删除容器存储卷 [names]","formatEN":"delete container volume [names]"}
func DeleteVolume(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.BatchDelete](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().DeleteVolume(R); err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}

// @Tags Container Volume
// @Summary Create Container Volume
// @Accept json
// @Param request body dto.VolumeCreate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/volume [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"创建容器存储卷 [name]","formatEN":"create container volume [name]"}
func CreateVolume(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.VolumeCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().CreateVolume(R); err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}

// @Tags Container Compose
// @Summary Page composes
// @Accept json
// @Param request body dto.SearchWithPage true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/compose/search [post]
func SearchCompose(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SearchWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}

	total, list, err := containerService.PageCompose(req)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(dto.PageResult{
		Items: list,
		Total: total,
	}))
}

// @Tags Container Compose
// @Summary Create compose
// @Accept json
// @Param request body dto.ComposeCreate true "request"
// @Success 200 {string} log
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/compose [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"创建 compose [name]","formatEN":"create compose [name]"}
func CreateCompose(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ComposeCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}

	log, err := containerService.CreateCompose(req)
	if err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(log))
}

// @Tags Container Compose
// @Summary Test compose
// @Accept json
// @Param request body dto.ComposeCreate true "request"
// @Success 200 {boolean} isOK
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/compose/test [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"检测 compose [name] 格式","formatEN":"check compose [name]"}
func TestCompose(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ComposeCreate](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}

	isOK, err := containerService.TestCompose(req)
	if err != nil {
		return c.JSON(e.Fail(err))
		// return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(isOK))
}

// @Tags Container Compose
// @Summary Operate compose
// @Accept json
// @Param request body dto.ComposeOperation true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/compose/operate [post]
// @x-panel-log {"bodyKeys":["name","operation"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"compose [operation] [name]","formatEN":"compose [operation] [name]"}
func OperatorCompose(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ComposeOperation](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}

	if err := containerService.ComposeOperation(req); err != nil {
		return c.JSON(e.Error(buserr.Err(err)))
	}

	if req.Operation == "delete" {
		appInstallService := service.NewAppInstall()

		appInstall := appInstallService.GetByName(req.Name)
		if appInstall != nil {
			appInstallService.Delete(appInstall.ID)
		}
	}

	return c.JSON(e.Succ())
}

// @Tags Container Compose
// @Summary Update Container Compose
// @Accept json
// @Param request body dto.ComposeUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/compose/update [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新 compose [name]","formatEN":"update compose information [name]"}
func ComposeUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ComposeUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}

	if err := containerService.ComposeUpdate(req); err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ())
}

// @Tags Container Compose
// @Summary Container Compose logs
// @Param compose query string false "compose file address"
// @Param since query string false "date"
// @Param follow query string false "follow"
// @Param tail query string false "tail"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/compose/search/log [get]
func ComposeLogs(c *websocket.Conn) {
	defer c.Close()

	compose := c.Query("compose")
	since := c.Query("since")
	follow := c.Query("follow") == "true"
	tail := c.Query("tail")

	if err := containerService.ContainerLogs(c, "compose", compose, since, tail, follow); err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
}
