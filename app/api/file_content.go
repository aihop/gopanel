package api

import (
	"errors"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"path/filepath"
	"strings"
)

func GetContent(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileContentReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(req.Path), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}
	info, err := fileService.GetContent(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(info))
}
func SaveContent(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileEdit](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(req.Path), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}
	if err := fileService.SaveContent(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}
func ReadFileByLine(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileReadByLineReq](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}
	res, err := fileService.ReadLogByLine(*req)
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(res))
}
