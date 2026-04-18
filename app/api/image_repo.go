package api

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

// @Tags Container Image-repo
// @Summary Page image repos
// @Accept json
// @Param request body dto.SearchWithPage true "request"
// @Produce json
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/repo/search [post]
func SearchRepo(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SearchWithPage](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	total, list, err := imageRepoService.Page(*req)
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}

	return c.JSON(e.Succ(dto.PageResult{
		Items: list,
		Total: total,
	}))
}

// @Tags Container Image-repo
// @Summary List image repos
// @Produce json
// @Success 200 {array} dto.ImageRepoOption
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/repo [get]
func ListRepo(c fiber.Ctx) error {
	list, err := imageRepoService.List()
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(list))
}

// @Tags Container Image-repo
// @Summary Load repo status
// @Accept json
// @Param request body dto.OperateByID true "request"
// @Produce json
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/repo/status [get]
func CheckRepoStatus(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.OperateByID](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}
	if err := imageRepoService.Login(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(nil))
}

// @Tags Container Image-repo
// @Summary Create image repo
// @Accept json
// @Param request body dto.ImageRepoDelete true "request"
// @Produce json
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/repo [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"创建镜像仓库 [name]","formatEN":"create image repo [name]"}
func CreateRepo(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ImageRepoCreate](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	if err := imageRepoService.Create(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(nil))
}

// @Tags Container Image-repo
// @Summary Delete image repo
// @Accept json
// @Param request body dto.ImageRepoDelete true "request"
// @Produce json
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/repo/del [post]
// @x-panel-log {"bodyKeys":["ids"],"paramKeys":[],"BeforeFunctions":[{"input_column":"id","input_value":"ids","isList":true,"db":"image_repos","output_column":"name","output_value":"names"}],"formatZH":"删除镜像仓库 [names]","formatEN":"delete image repo [names]"}
func DeleteRepo(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ImageRepoDelete](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}
	if err := imageRepoService.BatchDelete(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(nil))
}

// @Tags Container Image-repo
// @Summary Update image repo
// @Accept json
// @Param request body dto.ImageRepoUpdate true "request"
// @Produce json
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /container/repo/update [post]
// @x-panel-log {"bodyKeys":["id"],"paramKeys":[],"BeforeFunctions":[{"input_column":"id","input_value":"id","isList":false,"db":"image_repos","output_column":"name","output_value":"name"}],"formatZH":"更新镜像仓库 [name]","formatEN":"update image repo information [name]"}
func UpdateRepo(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.ImageRepoUpdate](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	if err := imageRepoService.Update(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(nil))
}
