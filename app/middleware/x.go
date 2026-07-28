package middleware

import (
	"github.com/gofiber/fiber/v3"

	"github.com/aihop/gopanel/constant"
)

func XGetAuth(c fiber.Ctx) string {
	return c.Get(constant.AppXAuth)
}
