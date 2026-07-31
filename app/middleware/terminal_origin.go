package middleware

import (
	"errors"
	"net/url"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/gofiber/fiber/v3"
)

func HostTerminalSameOrigin(c fiber.Ctx) error {
	origin := strings.TrimSpace(c.Get(fiber.HeaderOrigin))
	host := string(c.Request().Header.Host())
	if isHostTerminalOriginAllowed(origin, host) {
		return c.Next()
	}
	return c.JSON(e.Auth(errors.New("终端 WebSocket 来源不受信任").Error()))
}

func isHostTerminalOriginAllowed(origin, host string) bool {
	origin, host = strings.TrimSpace(origin), strings.TrimSpace(host)
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Scheme != "" && strings.EqualFold(parsed.Host, host)
}
