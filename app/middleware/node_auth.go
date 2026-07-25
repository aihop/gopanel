package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/nodesign"
	"github.com/gofiber/fiber/v3"
)

// NodeReadOnlyAuth 被控侧只读接口鉴权。
//
// 主控用节点令牌对 ts+nonce+method+path 做 HMAC-SHA256，令牌本身不出现在请求里，
// 因此即使节点面板跑在明文 HTTP 上（内网常见），令牌也不会被旁路嗅探到。
//
// 注意：nonce 只用来让签名不可预测，并没有服务端存储去查重，
// 所以在 nodeSignSkewSeconds 窗口内重放是可能的。本中间件只挂在只读摘要接口上，
// 重放的后果仅是重复读取一次摘要，可以接受。如果以后要挂到写接口上，必须补 nonce 查重。
func NodeReadOnlyAuth(c fiber.Ctx) error {
	token, err := loadNodeAccessToken()
	if err != nil || token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.Result{
			Code: constant.StatusCodeAuthInvalid,
			Msg:  "node read-only access is not enabled",
		})
	}

	ts := strings.TrimSpace(c.Get("X-Node-Ts"))
	nonce := strings.TrimSpace(c.Get("X-Node-Nonce"))
	sign := strings.TrimSpace(c.Get("X-Node-Sign"))
	if ts == "" || nonce == "" || sign == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.Result{
			Code: constant.StatusCodeAuthInvalid,
			Msg:  "missing node signature headers",
		})
	}

	tsValue, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.Result{
			Code: constant.StatusCodeAuthInvalid,
			Msg:  "invalid node timestamp",
		})
	}
	if skew := time.Now().Unix() - tsValue; skew > nodesign.SkewSeconds || skew < -nodesign.SkewSeconds {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.Result{
			Code: constant.StatusCodeAuthInvalid,
			Msg:  "node timestamp expired",
		})
	}

	expected := nodesign.Sign(token, ts, nonce, c.Method(), c.Path())
	if !nodesign.Equal(expected, sign) {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.Result{
			Code: constant.StatusCodeAuthInvalid,
			Msg:  "node signature mismatch",
		})
	}
	return c.Next()
}

func loadNodeAccessToken() (string, error) {
	setting, err := repo.NewISettingRepo().Get(repo.NewISettingRepo().WithByKey(constant.NodeAccessTokenKey))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(setting.Value) == "" {
		return "", nil
	}
	return encrypt.StringDecrypt(setting.Value)
}
