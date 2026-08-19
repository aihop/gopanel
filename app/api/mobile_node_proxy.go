package api

import (
	"errors"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/gofiber/fiber/v3"
)

func MobileNodeProxy(c fiber.Ctx) error {
	if !middleware.IsMobileNodeProxyTargetAllowed(c.Method(), c.Params("*")) {
		return c.JSON(e.Fail(errors.New(middleware.MobileErrorMessage(c, "ErrMobileNodeProxyDenied"))))
	}
	return NodeProxy(c)
}
