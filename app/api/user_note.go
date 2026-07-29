package api

import (
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

type userNoteSaveRequest struct {
	Content string `json:"content"`
}

func UserNoteGet(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	note, err := service.NewUserNoteService().Get(claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(note))
}

func UserNoteSave(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	req, err := e.BodyToStruct[userNoteSaveRequest](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	note, err := service.NewUserNoteService().Save(claims.UserId, req.Content)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(note))
}
