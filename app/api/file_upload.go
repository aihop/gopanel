package api

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/gofiber/fiber/v3"
)

func FileUploadSearch(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.SearchUploadWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	total, files, err := fileService.SearchUploadWithPage(R)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(dto.PageResult{
		Items: files,
		Total: total,
	}))
}
