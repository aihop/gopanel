package api

import (
	"errors"
	"strconv"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/gofiber/fiber/v3"
)

// @Tags App
// @Summary List apps
// @Accept json
// @Param request body request.AppSearch true "request"
// @Success 200 {object} response.AppRes
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /apps/search [GET]
func AppsSearch(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppSearch](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	list, err := appService.PageApp(c, *req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(list))
}

// @Tags App
// @Summary Get app
// @Accept json
// @Param key path string true "app key"
// @Success 200 {object} response.AppDTO
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /apps/{key} [get]
func AppGet(c fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.JSON(e.Fail(errors.New("key is required")))
	}
	res, err := appService.GetApp(c, key)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}

// @Tags App
// @Summary Get app detail
// @Accept json
// @Param id path int true "app id"
// @Param version query string false "app version"
// @Success 200 {object} response.AppDetailDTO
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /apps/detail/{id} [get]
func AppDetailGet(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.JSON(e.Fail(errors.New("invalid app id")))
	}

	version := c.Query("version")
	res, err := appService.GetAppDetail(c, uint(id), version)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}

func AppSync(c fiber.Ctx) error {
	if err := appService.SyncAppList(); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
