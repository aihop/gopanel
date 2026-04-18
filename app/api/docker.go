package api

import (
	"os"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/init/docker"
	"github.com/gofiber/fiber/v3"
)

// @Tags Container Docker
// @Summary Load docker status
// @Produce json
// @Success 200 {string} status
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/docker/status [get]
func LoadDockerStatus(c fiber.Ctx) error {
	status := dockerService.LoadDockerStatus()
	return c.JSON(e.Succ(status))
}

// @Tags Container Docker
// @Summary Load docker daemon.json
// @Produce json
// @Success 200 {object} string
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/daemonjson/file [get]
func LoadDaemonJsonFile(c fiber.Ctx) error {
	if _, err := os.Stat(docker.DaemonJsonPath); err != nil {
		return c.JSON(e.Error(buserr.Err(err)))
	}
	content, err := os.ReadFile(docker.DaemonJsonPath)
	if err != nil {
		return c.JSON(e.Error(buserr.Err(err)))
	}
	return c.JSON(e.Succ(string(content)))
}

// @Tags Container Docker
// @Summary Load docker daemon.json
// @Produce json
// @Success 200 {object} dto.DaemonJsonConf
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/daemonjson [get]
func LoadDaemonJson(c fiber.Ctx) error {
	conf := dockerService.LoadDockerConf()
	return c.JSON(e.Succ(conf))
}

// @Tags Container Docker
// @Summary Update docker daemon.json
// @Accept json
// @Param request body dto.SettingUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/daemonjson/update [post]
// @x-panel-log {"bodyKeys":["key", "value"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新配置 [key]","formatEN":"Updated configuration [key]"}
func UpdateDaemonJson(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SettingUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	if err := dockerService.UpdateConf(*req); err != nil {
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(nil))
}

// @Tags Container Docker
// @Summary Update docker daemon.json log option
// @Accept json
// @Param request body dto.LogOption true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/logoption/update [post]
// @x-panel-log {"bodyKeys":[],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新日志配置","formatEN":"Updated the log option"}
func UpdateLogOption(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.LogOption](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	if err := dockerService.UpdateLogOption(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(nil))
}

// @Tags Container Docker
// @Summary Update docker daemon.json ipv6 option
// @Accept json
// @Param request body dto.LogOption true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/ipv6option/update [post]
// @x-panel-log {"bodyKeys":[],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新 ipv6 配置","formatEN":"Updated the ipv6 option"}
func UpdateIpv6Option(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.Ipv6Option](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	if err := dockerService.UpdateIpv6Option(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(nil))
}

// @Tags Container Docker
// @Summary Update docker daemon.json by upload file
// @Accept json
// @Param request body dto.DaemonJsonUpdateByFile true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/daemonjson/update/byfile [post]
// @x-panel-log {"bodyKeys":[],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新配置文件","formatEN":"Updated configuration file"}
func UpdateDaemonJsonByFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.DaemonJsonUpdateByFile](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}

	if err := dockerService.UpdateConfByFile(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(nil))
}

// @Tags Container Docker
// @Summary Operate docker
// @Accept json
// @Param request body dto.DockerOperation true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/docker/operate [post]
// @x-panel-log {"bodyKeys":["operation"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"docker 服务 [operation]","formatEN":"[operation] docker service"}
func OperateDocker(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.DockerOperation](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}

	if err := dockerService.OperateDocker(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(nil))
}
