package api

import (
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/gofiber/fiber/v3"
	"os"
	"strconv"
)

func ContainerList(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.PageContainer](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	total, list, err := service.NewIContainerService().Page(R)
	if err != nil {
		return c.JSON(e.Error(buserr.Err(err)))
	}
	return c.JSON(e.Succ(dto.PageResult{Items: list, Total: total}))
}
func ContainerAll(c fiber.Ctx) error {
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
func LoadResourceLimit(c fiber.Ctx) error {
	data, err := service.NewIContainerService().LoadResourceLimit()
	if err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(data))
}
func ContainerListStats(c fiber.Ctx) error {
	data, err := service.NewIContainerService().ContainerListStats()
	if err != nil {
		return c.JSON(e.Error(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(data))
}
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
func ContainerCleanLogs(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.OperationWithName](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().ContainerLogClean(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}
func ContainerLoadLogs(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.OperationWithNameAndType](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	content := service.NewIContainerService().LoadContainerLogs(R)
	return c.JSON(e.Succ(content))
}
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
func ContainerOperation(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerOperation](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewIContainerService().ContainerOperation(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}
func ContainerStatsID(c fiber.Ctx) error {
	containerID := c.Params("id")
	fmt.Print(containerID)
	if containerID == "" {
		return c.JSON(e.Fail(buserr.Err(errors.New("error container id in path"))))
	}
	result, err := service.NewIContainerService().ContainerStatsByID(containerID)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	return c.JSON(e.Succ(result))
}
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
func ContainerLogs(c *websocket.Conn) {
	defer c.Close()
	container := c.Query("container")
	since := c.Query("since")
	follow := c.Query("follow") == "true"
	tail := c.Query("tail")
	runtimeHost := c.Query("runtimeHost")
	if err := containerService.ContainerLogs(c, "container", container, since, tail, runtimeHost, follow); err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
	}
}
func DownloadContainerLogs(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.ContainerLog](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	info, err := service.NewIContainerService().DownloadContainerLogs(R.ContainerType, R.Container, R.Since, strconv.Itoa(int(R.Tail)), R.RuntimeHost)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(errors.New(constant.ErrTypeInternalServer))))
	}
	defer os.Remove(info)
	return c.Download(info)
}
