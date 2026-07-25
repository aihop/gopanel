// Package nodesign 提供主控与被控节点之间的请求签名算法。
// 单独成包是为了让 app/middleware（校验方）和 app/service（签名方）都能引用而不产生循环依赖。
package nodesign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SkewSeconds 允许的时间偏移，主控与节点时钟不同步时留出余量
const SkewSeconds = 300

// Sign 对 ts + nonce + method + path 做 HMAC-SHA256。
// 令牌本身不出现在请求里，因此节点跑在明文 HTTP 上（内网常见）时令牌也不会被嗅探。
func Sign(token, ts, nonce, method, path string) string {
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write([]byte(ts + "\n" + nonce + "\n" + strings.ToUpper(method) + "\n" + path))
	return hex.EncodeToString(mac.Sum(nil))
}

// Equal 常量时间比较签名，避免时序侧信道
func Equal(expected, actual string) bool {
	return hmac.Equal([]byte(expected), []byte(actual))
}
