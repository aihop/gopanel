package middleware

import (
	"crypto/md5"
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

		// 允许 SSE 或 WebSocket 请求在建立连接时绕过 Header JWT 检查，如果在 Query 中带了有效 Token
		if (strings.HasSuffix(c.Path(), "/logs") || strings.HasSuffix(c.Path(), "/terminal")) && c.Query("token") != "" {
			c.Request().Header.Set("x-auth", c.Query("token"))
		}

		xAuth := XGetAuth(c)
		info, err := JwtCheck(xAuth, role)
		if err != nil && info == nil {
			tokenStr := c.Get(constant.AppToken)
			if tokenStr == "" {
				tokenStr = c.Query(constant.AppToken)
			}
			if tokenStr != "" {
				// 验证token是否合法
				user, err := service.NewUser().GetByToken(tokenStr)
				if err == nil && user.ID > 0 {
					if user.Token != tokenStr {
						return c.JSON(e.Auth("token invalid"))
					}
					// 设置用户信息到上下文
					c.Locals(constant.AppAuthName, &token.CustomClaims{
						UserId: user.ID,
						Role:   user.Role,
						SaltId: user.Salt,
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
		// 查询不到用户
		return info, errors.New("token invalid, user does not exist")
	}
	salt := user.Salt

	// 验证盐值是否正确
	if info.SaltId != salt {
		return info, errors.New("token Invalid, salt error")
	}

	// 如果传入了所需的 role，进行简单校验
	// SUB_ADMIN 也是后台管理员，如果要求 ADMIN，SUB_ADMIN 也可以放行（通过后续拦截器细化权限）
	if role != "" {
		if role == constant.UserRoleAdmin {
			if info.Role != constant.UserRoleAdmin && info.Role != constant.UserRoleSuper && info.Role != constant.UserRoleSubAdmin {
				return info, errors.New("permission denied")
			}
		} else if role == constant.UserRoleSuper {
			if info.Role != constant.UserRoleSuper {
				return info, errors.New("permission denied")
			}
		} else {
			// 如果路由要求特定角色（比如 SUB_ADMIN），则只允许对应角色或更高级别角色（Admin/Super）访问
			if role == constant.UserRoleSubAdmin {
				if info.Role != constant.UserRoleSubAdmin && info.Role != constant.UserRoleAdmin && info.Role != constant.UserRoleSuper {
					return info, errors.New("permission denied")
				}
			} else if info.Role != role {
				return info, errors.New("permission denied")
			}
		}
	}

	return info, nil
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
	if apiKeyValidityTime == "" {
		apiKeyValidityTime = "0"
	}
	apiTime, err := strconv.Atoi(apiKeyValidityTime)
	if err != nil || apiTime < 0 {
		global.LOG.Errorf("apiTime %s, err: %v", apiKeyValidityTime, err)
		return false
	}
	if apiTime == 0 {
		return true
	}
	panelTime, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		global.LOG.Errorf("timestamp %s, panelTime %d, apiTime %d, err: %v", timestamp, apiTime, panelTime, err)
		return false
	}
	nowTime := time.Now().Unix()
	tolerance := int64(60)
	if panelTime > nowTime+tolerance {
		global.LOG.Errorf("Valid Panel Timestamp, apiTime %d, panelTime %d, nowTime %d, err: %v", apiTime, panelTime, nowTime, err)
		return false
	}
	return nowTime-panelTime <= int64(apiTime)*60+tolerance
}

func isValidApiKEY(requestKey string, timestamp string) bool {
	serverKey := global.CONF.System.ApiKey
	expectedKey := GenerateMD5("gopanel_" + serverKey + "_" + timestamp)
	return requestKey == expectedKey
}

func GenerateMD5(param string) string {
	hash := md5.New()
	hash.Write([]byte(param))
	return hex.EncodeToString(hash.Sum(nil))
}
