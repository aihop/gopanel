package middleware

import (
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

func Session() func(fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		jwtInst := c.Locals(constant.AuthMethodName)
		if jwt, ok := jwtInst.(string); ok && jwt == constant.AuthMethodJWT {
			return c.Next()
		}
		sId := c.Cookies(constant.SessionName)
		if sId == "" {
			return c.JSON(e.Auth("session id error"))
		}
		sessionId, err := global.SESSION.Get(sId)
		if err != nil {
			return c.JSON(e.Auth("session not found"))
		}
		_ = global.SESSION.Set(sId, sessionId, 86400)
		return c.Next()
	}
}
