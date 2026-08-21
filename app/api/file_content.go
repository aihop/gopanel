package api

import (
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

func GetContent(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileContentReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := requireFileAccess(c, req.Path); err != nil {
		return c.JSON(e.Fail(err))
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
	if err := requireFileAccess(c, req.Path); err != nil {
		return c.JSON(e.Fail(err))
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
	if _, scoped := fileBaseDir(c); scoped {
		return c.JSON(e.Fail(buserr.New("ErrSubAdminCannotAccessServiceLog")))
	}
	res, err := fileService.ReadLogByLine(*req)
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(res))
}
