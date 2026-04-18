package middleware

import (
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v3"
)

func PasswordPublicKey() func(fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		cookieKey := c.Cookies("public_key")
		base64Key := base64.StdEncoding.EncodeToString([]byte("xxxxxx"))
		if base64Key == cookieKey {
			return c.Next()
		}
		// 设置 cookie，有效期1天
		c.Cookie(&fiber.Cookie{
			Name:     "public_key",
			Value:    base64Key,
			Expires:  time.Now().Add(1 * 24 * time.Hour),
			Path:     "/",
			HTTPOnly: false,
			Secure:   c.Scheme() == "https",
		})
		return c.Next()
	}
}
