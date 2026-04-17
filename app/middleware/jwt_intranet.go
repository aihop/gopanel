package middleware

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// JWTOrIntranet: 如果来源 IP 在私有网段或回环地址则直接放行，否则继续执行原有的 JWT(role) 校验。
// role 直接透传给现有的 JWT 中间件。
func JWTOrIntranet(role string) fiber.Handler {
	return func(c fiber.Ctx) error {
		ipStr := clientFirstIP(c.IP())
		if ipStr != "" && isPrivateIP(net.ParseIP(ipStr)) {
			return c.Next()
		}
		return JWT(role)(c)
	}
}

func clientFirstIP(ip string) string {
	// fiber.Ctx.IP() 可能包含多个用逗号分隔的地址（X-Forwarded-For），取第一个
	if ip == "" {
		return ""
	}
	parts := strings.Split(ip, ",")
	return strings.TrimSpace(parts[0])
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	// loopback
	if ip.IsLoopback() {
		return true
	}
	// IPv4 RFC1918
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		}
		return false
	}
	// IPv6 unique local fc00::/7
	if ip.IsPrivate() {
		return true
	}
	// fe80 link-local 也认为是本机网络
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	return false
}
