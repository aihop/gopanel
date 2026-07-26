package middleware

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/utils/nodesign"
	"github.com/gofiber/fiber/v3"
)

// 节点代理签名头。只读摘要接口复用同名头但走另一套令牌与算法，
// 两者按路由区分：/api/node/summary 用只读令牌 + Sign，其余接口用控制令牌 + SignBody。
const (
	NodeSignHeader  = "X-Node-Sign"
	NodeTsHeader    = "X-Node-Ts"
	NodeNonceHeader = "X-Node-Nonce"
)

// HasNodeProxySignature 请求是否声称自己是主控代理过来的
func HasNodeProxySignature(c fiber.Ctx) bool {
	return strings.TrimSpace(c.Get(NodeSignHeader)) != ""
}

// NodeProxyQueryLocalKey 过滤后的节点侧查询串在 Locals 里的键。
// ws 升级后拿不到完整 query（pkg/websocket 只暴露按 key 取值），所以在升级前算好存进来。
const NodeProxyQueryLocalKey = "nodeProxyQuery"

// masterOnlyQueryKeys 只属于主控鉴权、绝不能转发给节点的查询参数。
// ws 握手加不了请求头，浏览器只能把主控 JWT 放在 query 里，原样转发等于把它泄露给节点。
var masterOnlyQueryKeys = []string{"auth", "token"}

// NodeProxyWsQuery 在 WebSocket 升级前算出要转发给节点的查询串。
// 除主控鉴权参数外全部透传，这样 Terminal.vue 之类现有组件的 cols/rows/containerID
// 不用任何改动就能正确到达节点。
func NodeProxyWsQuery(c fiber.Ctx) error {
	values := url.Values{}
	c.RequestCtx().QueryArgs().VisitAll(func(key, value []byte) {
		name := string(key)
		for _, masterOnly := range masterOnlyQueryKeys {
			if strings.EqualFold(name, masterOnly) {
				return
			}
		}
		values.Add(name, string(value))
	})
	// Encode 会按 key 排序，保证主控签名的串与实际发出的串完全一致
	c.Locals(NodeProxyQueryLocalKey, values.Encode())
	return c.Next()
}

// VerifyNodeProxy 校验主控代理过来的请求。
//
// 与只读校验的三个区别：
//  1. 用控制令牌（NodeControlToken），只读令牌不能授权写操作
//  2. 签名覆盖请求体哈希——节点常跑明文 HTTP，不签 body 中间人可篡改参数
//  3. nonce 查重，防止重放。重放一次"删除容器"是真实损失，不像重放读摘要那样无害
func VerifyNodeProxy(c fiber.Ctx) error {
	token, err := service.LoadLocalControlToken()
	if err != nil {
		return errors.New("read node control token failed")
	}
	if token == "" {
		return errors.New("node control access is not enabled")
	}

	ts := strings.TrimSpace(c.Get(NodeTsHeader))
	nonce := strings.TrimSpace(c.Get(NodeNonceHeader))
	sign := strings.TrimSpace(c.Get(NodeSignHeader))
	if ts == "" || nonce == "" || sign == "" {
		return errors.New("missing node signature headers")
	}

	tsValue, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("invalid node timestamp")
	}
	if skew := time.Now().Unix() - tsValue; skew > nodesign.SkewSeconds || skew < -nodesign.SkewSeconds {
		return errors.New("node timestamp expired")
	}

	// 先验签再消费 nonce：否则攻击者可以用伪造签名把合法 nonce 提前占掉，
	// 让主控随后那次真实请求被当成重放拒绝（自造的拒绝服务）
	rawQuery := string(c.RequestCtx().QueryArgs().QueryString())
	expected := nodesign.SignBody(token, ts, nonce, c.Method(), c.Path(), rawQuery, nodesign.BodyHash(c.Body()))
	if !nodesign.Equal(expected, sign) {
		return errors.New("node signature mismatch")
	}
	if !nodesign.ConsumeNonce(nonce) {
		return errors.New("node nonce replayed")
	}
	return nil
}
