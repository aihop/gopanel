package middleware

import (
	"encoding/base64"
	"net"
	"strings"
	"time"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

func LocalOrJWT(role string) fiber.Handler {
	jwtMiddleware := JWT(role)
	return func(c fiber.Ctx) error {
		ip := c.IP()
		if isLocalIP(ip) {
			return c.Next()
		}
		return jwtMiddleware(c)
	}
}

func isLocalIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.IsLoopback()
}

const NoEntrance = `<!doctype html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>安全访问受限</title>
    <style>
        :root {
            --primary: #4f46e5;
            --bg: #f8fafc;
            --card-bg: #ffffff;
            --text-main: #0f172a;
            --text-sub: #64748b;
        }
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background-color: var(--bg);
            background-image: radial-gradient(#e2e8f0 1px, transparent 1px);
            background-size: 24px 24px;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            color: var(--text-main);
        }
        .container {
            background-color: var(--card-bg);
            border-radius: 24px;
            box-shadow: 0 20px 40px -10px rgba(0, 0, 0, 0.08), 0 0 0 1px rgba(0, 0, 0, 0.02);
            padding: 48px 40px;
            text-align: center;
            width: 90%;
            max-width: 470px;
            position: relative;
            overflow: hidden;
            animation: float-up 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
        }
        .container::before {
            content: "";
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 4px;
            background: linear-gradient(90deg, #3b82f6, #8b5cf6, #6366f1);
        }
        .icon-wrapper {
            width: 72px;
            height: 72px;
            background: #eff6ff;
            border-radius: 20px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 24px;
            color: #3b82f6;
        }
        h1 {
            font-size: 24px;
            font-weight: 700;
            letter-spacing: -0.02em;
            margin-bottom: 12px;
            color: var(--text-main);
        }
        p {
            font-size: 15px;
            color: var(--text-sub);
            line-height: 1.6;
            margin-bottom: 28px;
        }
        .action-box {
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 12px;
            padding: 16px;
            font-size: 14px;
            color: var(--text-sub);
            margin-bottom: 12px;
        }
        .action-box strong {
            color: var(--text-main);
            font-weight: 600;
        }
        @keyframes float-up {
            0% { opacity: 0; transform: translateY(20px); }
            100% { opacity: 1; transform: translateY(0); }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon-wrapper">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
            </svg>
        </div>
        <h1>安全访问受限</h1>
        <p>当前系统环境已开启安全防护模式，为保障数据安全，拦截了您的非法访问请求。</p>
        <div class="action-box">
            请使用管理员配置的 <strong>专属安全入口 URL</strong> 重新进行访问
        </div>
        <div class="action-box">
            <p>可在 SSH 终端输入以下命令来查看面板入口：</p>
            <p><strong>gpc panel user-info</strong></p>
        </div>
    </div>
</body>
</html>
`

// 安全入口
func Entrance(c fiber.Ctx) error {
	entrance := strings.TrimSpace(global.CONF.System.Entrance)
	if entrance == "" || shouldBypassEntrance(c.Path()) {
		return c.Next()
	}
	if authorizeEntrance(c, entrance) {
		return c.Next()
	}
	return renderNoEntrance(c)
}

func shouldBypassEntrance(path string) bool {
	switch path {
	case "/health":
		return true
	default:
		return false
	}
}

func authorizeEntrance(c fiber.Ctx, entrance string) bool {
	securityEntrance := "/" + strings.TrimPrefix(entrance, "/")
	currentPath := strings.TrimRight(c.Path(), "/")
	if currentPath == "" {
		currentPath = "/"
	}
	if currentPath == securityEntrance {
		setEntranceCookie(c, entrance)
		return true
	}

	if matchesEntranceValue(c.Query("entrance"), entrance) {
		setEntranceCookie(c, entrance)
		return true
	}

	if matchesEntranceValue(c.Get("EntranceCode"), entrance) {
		setEntranceCookie(c, entrance)
		return true
	}

	if matchesEntranceValue(c.Cookies(constant.Entrance), entrance) {
		return true
	}
	return false
}

func matchesEntranceValue(raw string, expected string) bool {
	raw = strings.TrimSpace(raw)
	expected = strings.TrimSpace(expected)
	if raw == "" || expected == "" {
		return false
	}
	if raw == expected {
		return true
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return false
	}
	return string(decoded) == expected
}

func setEntranceCookie(c fiber.Ctx, entrance string) {
	c.Cookie(&fiber.Cookie{
		Name:     constant.Entrance,
		Value:    base64.StdEncoding.EncodeToString([]byte(entrance)),
		Expires:  time.Now().Add(24 * time.Hour),
		Path:     "/",
		HTTPOnly: false,
		SameSite: "Lax",
		Secure:   c.Scheme() == "https",
	})
}

func renderNoEntrance(c fiber.Ctx) error {
	// 如果是 API 请求或前端带有 JSON 预期，返回 JSON 格式的错误
	isAPI := len(c.Path()) > 4 && c.Path()[:4] == "/api"
	wantsJSON := strings.Contains(c.Get("Accept"), "application/json") || strings.Contains(c.Get("Content-Type"), "application/json")

	if isAPI || wantsJSON {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"code":    403,
			"message": "当前环境已经开启了安全入口登录，请从安全入口登录",
		})
	}
	// 否则返回 HTML 页面
	c.Response().Header.SetContentType(fiber.MIMETextHTML)
	return c.SendString(NoEntrance)
}
