package api

import (
	"errors"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

type containerRuntimeInstallRequest struct {
	Runtime string `json:"runtime"`
}

func ContainerRuntimeInstall(c fiber.Ctx) error {
	req, err := e.BodyToStruct[containerRuntimeInstallRequest](c.Body())
	if err != nil {
		return c.JSON(e.Error(err))
	}
	runtimeKind := strings.ToLower(strings.TrimSpace(req.Runtime))
	task, err := service.StartContainerRuntimeInstall(runtimeKind)
	if err != nil {
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(task))
}

func ContainerRuntimeInstallStatus(c fiber.Ctx) error {
	taskID := strings.TrimSpace(c.Params("id"))
	if taskID == "" {
		return c.JSON(e.Error(errors.New("runtime install task id is required")))
	}
	task, err := service.GetContainerRuntimeInstallTask(taskID)
	if err != nil {
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(task))
}
