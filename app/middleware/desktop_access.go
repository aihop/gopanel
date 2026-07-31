package middleware

import (
	"crypto/subtle"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
)

const desktopTokenHeader = "X-GoPanel-Desktop-Token"

var desktopAccess struct {
	sync.RWMutex
	token string
}

func SetDesktopAccessToken(token string) {
	desktopAccess.Lock()
	desktopAccess.token = strings.TrimSpace(token)
	desktopAccess.Unlock()
}

func authorizeDesktopAccess(c fiber.Ctx) bool {
	desktopAccess.RLock()
	expected := desktopAccess.token
	desktopAccess.RUnlock()

	provided := strings.TrimSpace(c.Get(desktopTokenHeader))
	if provided == "" {
		provided = strings.TrimSpace(c.Query("desktop_token"))
	}
	return matchesDesktopAccess(c.IP(), provided, expected)
}

func matchesDesktopAccess(ip, provided, expected string) bool {
	if !isLocalIP(ip) || provided == "" || expected == "" {
		return false
	}
	return len(provided) == len(expected) && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
