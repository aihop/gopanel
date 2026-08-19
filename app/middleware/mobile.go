package middleware

import (
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	paneli18n "github.com/aihop/gopanel/pkg/i18n"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const (
	MobileDeviceCookie = "gopanel_mobile"
	MobileDeviceIDKey  = "mobile_device_id"
)

func MobileErrorMessage(c fiber.Ctx, key string) string {
	language := "zh"
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.Get("Accept-Language"))), "en") {
		language = "en"
	}
	return paneli18n.MustLocalize(key, language)
}

func MobileDeviceAuth(c fiber.Ctx) error {
	if HasNodeProxySignature(c) {
		if !IsMobileNodeProxyTargetAllowed(c.Method(), c.Path()) {
			return c.JSON(e.Auth(MobileErrorMessage(c, "ErrMobileNodeProxyUnauthorized")))
		}
		if err := VerifyNodeProxy(c); err != nil {
			return c.JSON(e.Auth(err.Error()))
		}
		c.Locals(constant.AppAuthName, &token.CustomClaims{UserId: 1, Role: constant.UserRoleSuper})
		c.Locals(constant.AuthMethodName, constant.AuthMethodNodeProxy)
		return c.Next()
	}
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead && c.Get("X-Mobile-Request") != "1" {
		return c.JSON(e.Auth("手机请求校验失败"))
	}
	deviceToken := strings.TrimSpace(c.Cookies(MobileDeviceCookie))
	if deviceToken == "" {
		deviceToken = strings.TrimSpace(c.Get("X-Mobile-Token"))
	}
	device, user, err := service.AuthenticateMobileDevice(deviceToken, c.IP(), string(c.Request().Header.UserAgent()))
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	c.Locals(constant.AppAuthName, &token.CustomClaims{UserId: user.ID, Role: user.Role, SaltId: user.Salt, FileBaseDir: user.FileBaseDir})
	c.Locals(constant.AuthMethodName, constant.AuthMethodMobile)
	c.Locals(MobileDeviceIDKey, device.ID)
	return c.Next()
}
