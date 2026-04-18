package middleware

import (
	"github.com/gofiber/fiber/v3"

	"github.com/aihop/gopanel/constant"
)

func XGetAuth(c fiber.Ctx) string {
	xAuth := c.Get(constant.AppXAuth)
	if xAuth != "" {
		return xAuth
	}
	xAuth = c.Query(constant.AppAuth)
	if xAuth != "" {
		return xAuth
	}
	xAuth = c.Cookies(constant.AppXAuth)
	if xAuth != "" {
		return xAuth
	}
	return ""
}
