package middleware

import "testing"

// SSE 接口漏加后缀 = 连接静默 401，前端只看到 onerror，极难排查。
// /host/disk/scan/stream 就是这么坏掉的，这里锁住允许名单。
func TestIsQueryTokenAllowedPath(t *testing.T) {
	allowed := []string{
		"/api/pipeline/logs",
		"/api/file/compress/logs",
		"/api/ai/terminal",
		"/api/host/disk/scan/stream",
	}
	for _, p := range allowed {
		if !isQueryTokenAllowedPath(p) {
			t.Errorf("%s 应允许用 ?token= 鉴权（EventSource 无法设置请求头）", p)
		}
	}

	// 普通接口不能开这个口子：URL 里的 token 会进访问日志和 Referer
	denied := []string{"/api/host/disk/clean", "/api/file/batch/del", "/api/host/disk/scan"}
	for _, p := range denied {
		if isQueryTokenAllowedPath(p) {
			t.Errorf("%s 不应允许 ?token= 鉴权", p)
		}
	}
}
