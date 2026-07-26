// Package nodesign 提供主控与被控节点之间的请求签名算法。
// 单独成包是为了让 app/middleware（校验方）和 app/service（签名方）都能引用而不产生循环依赖。
package nodesign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// SkewSeconds 允许的时间偏移，主控与节点时钟不同步时留出余量
const SkewSeconds = 300

// Sign 只读摘要接口用的签名：覆盖 ts + nonce + method + path。
// 已经部署到线上节点，签名内容不能再改——否则新主控配旧节点会全部签名失败。
func Sign(token, ts, nonce, method, path string) string {
	return mac(token, ts+"\n"+nonce+"\n"+strings.ToUpper(method)+"\n"+path)
}

// SignBody 代理接口用的签名：在 Sign 的基础上再覆盖查询串与请求体哈希。
//
// 代理会执行写操作，节点常跑在明文 HTTP 上（内网）。只签路径的话，
// 中间人能在签名有效的前提下改 body 或 query——比如把 ?containerID=A 改成 B，
// 或把 {"operation":"stop"} 改成 {"operation":"remove"}。
//
// 主控与节点必须同版本：这里的签名内容一变，旧版节点会全部校验失败。
func SignBody(token, ts, nonce, method, path, rawQuery, bodyHash string) string {
	return mac(token, ts+"\n"+nonce+"\n"+strings.ToUpper(method)+"\n"+path+"\n"+rawQuery+"\n"+bodyHash)
}

// BodyHash 请求体的 sha256 十六进制串，空体固定为空串的哈希
func BodyHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func mac(token, payload string) string {
	m := hmac.New(sha256.New, []byte(token))
	m.Write([]byte(payload))
	return hex.EncodeToString(m.Sum(nil))
}

// Equal 常量时间比较签名，避免时序侧信道
func Equal(expected, actual string) bool {
	return hmac.Equal([]byte(expected), []byte(actual))
}

// nonceStore 已用过的 nonce。
// 只读接口重放无害（重复读一次摘要），但代理会执行写操作——
// 重放一次"删除容器"是真实损失，所以代理路径必须查重。
type nonceStore struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

// maxNonces 上限，防止被大量伪造 nonce 撑爆内存。
// 超过就整体清空：最坏情况是短时间内放过一次重放，比 OOM 可接受。
const maxNonces = 20000

var usedNonces = &nonceStore{seen: make(map[string]time.Time)}

// ConsumeNonce 记录并检查 nonce 是否已被用过。
// 返回 false 表示这是重放请求，调用方应拒绝。
func ConsumeNonce(nonce string) bool {
	if strings.TrimSpace(nonce) == "" {
		return false
	}
	now := time.Now()
	usedNonces.mu.Lock()
	defer usedNonces.mu.Unlock()

	// 顺手清理过期项：超出时间窗的 nonce 已经会被时间戳校验挡住，不必再留
	for key, at := range usedNonces.seen {
		if now.Sub(at) > time.Duration(SkewSeconds)*time.Second {
			delete(usedNonces.seen, key)
		}
	}
	if len(usedNonces.seen) >= maxNonces {
		usedNonces.seen = make(map[string]time.Time)
	}
	if _, exists := usedNonces.seen[nonce]; exists {
		return false
	}
	usedNonces.seen[nonce] = now
	return true
}
