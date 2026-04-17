package api

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

// @Tags Container Image
// @Summary Page images
// @Accept json
// @Param request body dto.SearchWithPage true "request"
// @Produce json
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image/search [post]
func SearchImage(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SearchWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}

	total, list, err := imageService.Page(*req)
	if err != nil {
		return c.JSON(e.Result(err))
	}

	return c.JSON(e.Succ(dto.PageResult{
		Items: list,
		Total: total,
	}))
}

// @Tags Container Image
// @Summary List all images
// @Produce json
// @Success 200 {array} dto.ImageInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image/all [get]
func ListAllImage(c fiber.Ctx) error {
	list, err := imageService.ListAll()
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(list))
}

// @Tags Container Image
// @Summary load images options
// @Produce json
// @Success 200 {array} dto.Options
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image [get]
func ListImage(c fiber.Ctx) error {
	list, err := imageService.List()
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(list))
}

// @Tags Container Image
// @Summary Build image
// @Accept json
// @Param request body dto.ImageBuild true "request"
// @Success 200 {string} log
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image/build [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"构建镜像 [name]","formatEN":"build image [name]"}
func ImageBuild(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ImageBuild](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	log, err := imageService.ImageBuild(*req)
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(log))
}

// @Tags Container Image
// @Summary Pull image
// @Accept json
// @Param request body dto.ImagePull true "request"
// @Success 200 {string} logPath
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image/pull [post]
// @x-panel-log {"bodyKeys":["repoID","imageName"],"paramKeys":[],"BeforeFunctions":[{"input_column":"id","input_value":"repoID","isList":false,"db":"image_repos","output_column":"name","output_value":"reponame"}],"formatZH":"镜像拉取 [reponame][imageName]","formatEN":"image pull [reponame][imageName]"}
func ImagePull(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ImagePull](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	logPath, err := imageService.ImagePull(*req)
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(logPath))
}

// @Tags Container Image
// @Summary Push image
// @Accept json
// @Param request body dto.ImagePush true "request"
// @Success 200 {string} logPath
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image/push [post]
// @x-panel-log {"bodyKeys":["repoID","tagName","name"],"paramKeys":[],"BeforeFunctions":[{"input_column":"id","input_value":"repoID","isList":false,"db":"image_repos","output_column":"name","output_value":"reponame"}],"formatZH":"[tagName] 推送到 [reponame][name]","formatEN":"push [tagName] to [reponame][name]"}
func ImagePush(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ImagePush](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	logPath, err := imageService.ImagePush(*req)
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(logPath))
}

// @Tags Container Image
// @Summary Delete image
// @Accept json
// @Param request body dto.BatchDelete true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image/remove [post]
// @x-panel-log {"bodyKeys":["names"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"移除镜像 [names]","formatEN":"remove image [names]"}
func ImageRemove(c fiber.Ctx) error {

	req, err := e.BodyToStruct[dto.BatchDelete](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	if err := imageService.ImageRemove(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(nil))
}

// @Tags Container Image
// @Summary Save image
// @Accept json
// @Param request body dto.ImageSave true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image/save [post]
// @x-panel-log {"bodyKeys":["tagName","path","name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"保留 [tagName] 为 [path]/[name]","formatEN":"save [tagName] as [path]/[name]"}
func ImageSave(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ImageSave](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	if err := imageService.ImageSave(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(nil))
}

// @Tags Container Image
// @Summary Tag image
// @Accept json
// @Param request body dto.ImageTag true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image/tag [post]
// @x-panel-log {"bodyKeys":["repoID","targetName"],"paramKeys":[],"BeforeFunctions":[{"input_column":"id","input_value":"repoID","isList":false,"db":"image_repos","output_column":"name","output_value":"reponame"}],"formatZH":"tag 镜像 [reponame][targetName]","formatEN":"tag image [reponame][targetName]"}
func ImageTag(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ImageTag](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	if err := imageService.ImageTag(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(nil))
}

// @Tags Container Image
// @Summary Load image
// @Accept json
// @Param request body dto.ImageLoad true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/image/load [post]
// @x-panel-log {"bodyKeys":["path"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"从 [path] 加载镜像","formatEN":"load image from [path]"}
func ImageLoad(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ImageLoad](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	if err := imageService.ImageLoad(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(nil))
}
