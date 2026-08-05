package middleware

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"

	"github.com/gofiber/fiber/v3"
)

// JWT is jwt middleware
func JWT(role string) func(fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		checkDemo := func(userRole string) error {
			if userRole != constant.UserRoleDemo {
				return nil
			}
			method := c.Method()
			if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions || strings.HasSuffix(c.Path(), "/list") || strings.HasSuffix(c.Path(), "/count") || strings.HasSuffix(c.Path(), "/search") {
				return nil
			}
			return errors.New("demo role is read-only")
		}

		// 允许 SSE 或 WebSocket 请求在建立连接时通过 Query token 兼容鉴权。
		// EventSource 无法自定义请求头，像 /runtime-logs 这类路径也需要走同一套 JWT 校验。
		// 后缀必须显式列全：新增 SSE 接口时忘了加，表现是连接被静默 401，
		// 前端只会看到 onerror，很难往鉴权上想（/host/disk/scan/stream 就踩过）。
		if isQueryTokenAllowed(c.Method(), c.Path()) && c.Query("token") != "" {
			c.Request().Header.Set("x-auth", c.Query("token"))
		}

		// 主控代理转发过来的请求没有用户 JWT，身份由节点控制令牌的签名证明。
		// 带了签名头就必须走这条路：验不过直接拒绝，不再回落到 JWT，
		// 否则错误会变成"未登录"，掩盖真正的原因（令牌不符 / 时钟偏差 / 重放）。
		if HasNodeProxySignature(c) {
			if err := VerifyNodeProxy(c); err != nil {
				return c.JSON(e.Auth(err.Error()))
			}
			// 与下方 API Key 鉴权一致：虚拟一个管理员身份放行。
			// 控制令牌本身就等价于该机管理员，这一点在签发时已向用户明示。
			c.Locals(constant.AppAuthName, &token.CustomClaims{
				UserId: 1,
				Role:   constant.UserRoleSuper,
			})
			c.Locals(constant.AuthMethodName, constant.AuthMethodNodeProxy)
			return c.Next()
		}

		xAuth := XGetAuth(c)
		info, err := JwtCheck(xAuth, role)
		if err != nil {
			if xAuth != "" {
				return c.JSON(e.Auth(err.Error()))
			}
			tokenStr := getUserAccessToken(c)
			if tokenStr != "" {
				// 验证token是否合法
				user, err := service.NewUser().GetByToken(tokenStr)
				if err == nil && user.ID > 0 {
					if user.Token != tokenStr {
						return c.JSON(e.Auth("token invalid"))
					}
					if user.Status != constant.UserStatusNormal {
						return c.JSON(e.Auth("user is disabled"))
					}
					if !isRoleAllowed(user.Role, role) {
						return c.JSON(e.Auth("permission denied"))
					}
					// 设置用户信息到上下文
					c.Locals(constant.AppAuthName, &token.CustomClaims{
						UserId:      user.ID,
						Role:        user.Role,
						SaltId:      user.Salt,
						FileBaseDir: user.FileBaseDir,
					})
					c.Locals(constant.AuthMethodName, constant.AuthMethodJWT)
					if err := checkDemo(user.Role); err != nil {
						return c.JSON(e.Fail(err))
					}
					return c.Next()
				}
			}
			apiKeyStr := c.Get(constant.AppAPIKey)
			if apiKeyStr == "" {
				apiKeyStr = c.Query(constant.AppAPIKey)
			}
			timestamp := c.Get(constant.AppTimestamp)
			if timestamp == "" {
				timestamp = c.Query(constant.AppTimestamp)
			}

			// 如果未开启 API 接口
			if global.CONF.System.ApiInterfaceStatus != "Open" {
				return c.JSON(e.Auth("API Interface is closed"))
			}
			if apiKeyStr == "" || timestamp == "" {
				return c.JSON(e.Auth("API Key or Timestamp missing"))
			}
			if !isValidTimestamp(timestamp) {
				return c.JSON(e.Auth("timestamp error or expired"))
			}
			if !isValidApiKEY(apiKeyStr, timestamp) {
				return c.JSON(e.Auth("apiKey error"))
			}
			// API 鉴权成功，虚拟一个管理员身份放行
			c.Locals(constant.AppAuthName, &token.CustomClaims{
				UserId: 1, // 虚拟管理员
				Role:   constant.UserRoleSuper,
			})
			return c.Next()
		}
		c.Locals(constant.AppAuthName, info)
		c.Locals(constant.AuthMethodName, constant.AuthMethodJWT)
		if err := checkDemo(info.Role); err != nil {
			return c.JSON(e.Fail(err))
		}
		return c.Next()
	}
}

// queryTokenAllowedSuffixes 允许用 ?token= 代替请求头做鉴权的路径后缀。
// 仅限 EventSource / WebSocket 这类无法自定义请求头的场景——
// URL 里的 token 会进访问日志和 Referer，不能对所有接口开放。
var queryTokenAllowedSuffixes = []string{"/logs", "/terminal", "/stream", "/ws", "/container/exec"}

func isQueryTokenAllowed(method, path string) bool {
	if method != fiber.MethodGet {
		return false
	}
	for _, suffix := range queryTokenAllowedSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func getUserAccessToken(c fiber.Ctx) string {
	tokenStr := c.Get(constant.AppToken)
	if tokenStr == "" && isQueryTokenAllowed(c.Method(), c.Path()) {
		tokenStr = c.Query(constant.AppToken)
	}
	return tokenStr
}

func JwtCheck(xAuth, role string) (info *token.CustomClaims, err error) {
	if xAuth == "" {
		return info, errors.New("not logged in")
	}
	// 编写记录黑名单
	info, err = token.Parse(xAuth)
	if err != nil {
		return info, errors.New("token parse Invalid")
	}

	user, err := service.NewUser().Get(info.UserId)
	if err != nil {
		return nil, errors.New("token invalid, user does not exist")
	}
	if user.Status != constant.UserStatusNormal {
		return nil, errors.New("user is disabled")
	}
	salt := user.Salt

	// 验证盐值是否正确
	if info.SaltId != salt {
		return nil, errors.New("token Invalid, salt error")
	}

	// 权限和文件作用域以数据库当前值为准，避免账号降权或目录调整后，
	// 尚未过期的 JWT 继续携带旧权限。
	if !isRoleAllowed(user.Role, role) {
		return nil, errors.New("permission denied")
	}
	info.Role = user.Role
	info.FileBaseDir = user.FileBaseDir

	return info, nil
}

func isRoleAllowed(actualRole, requiredRole string) bool {
	if requiredRole == "" {
		return true
	}
	switch requiredRole {
	case constant.UserRoleAdmin:
		return actualRole == constant.UserRoleAdmin || actualRole == constant.UserRoleSuper
	case constant.UserRoleSuper:
		return actualRole == constant.UserRoleSuper
	case constant.UserRoleSubAdmin:
		return actualRole == constant.UserRoleSubAdmin || actualRole == constant.UserRoleAdmin || actualRole == constant.UserRoleSuper
	default:
		return actualRole == requiredRole
	}
}

func JwtClaims(c fiber.Ctx) (info *token.CustomClaims, err error) {
	info, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok {
		return nil, errors.New("failed to get JWT claims from context")
	}
	return info, nil
}

func isValidTimestamp(timestamp string) bool {
	apiKeyValidityTime := global.CONF.System.ApiKeyValidityTime
	apiTime, err := strconv.Atoi(apiKeyValidityTime)
	if err != nil || apiTime <= 0 {
		if global.LOG != nil {
			global.LOG.Errorf("apiTime %s, err: %v", apiKeyValidityTime, err)
		}
		return false
	}
	panelTime, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		if global.LOG != nil {
			global.LOG.Errorf("timestamp %s, panelTime %d, apiTime %d, err: %v", timestamp, apiTime, panelTime, err)
		}
		return false
	}
	nowTime := time.Now().Unix()
	tolerance := int64(60)
	if panelTime > nowTime+tolerance {
		if global.LOG != nil {
			global.LOG.Errorf("Valid Panel Timestamp, apiTime %d, panelTime %d, nowTime %d, err: %v", apiTime, panelTime, nowTime, err)
		}
		return false
	}
	return nowTime-panelTime <= int64(apiTime)*60+tolerance
}

func isValidApiKEY(requestKey string, timestamp string) bool {
	serverKey := global.CONF.System.ApiKey
	expectedKey := GenerateMD5("gopanel_" + serverKey + "_" + timestamp)
	return subtle.ConstantTimeCompare([]byte(requestKey), []byte(expectedKey)) == 1
}

func GenerateMD5(param string) string {
	hash := md5.New()
	hash.Write([]byte(param))
	return hex.EncodeToString(hash.Sum(nil))
}
