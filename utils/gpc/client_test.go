package gpc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Do 的契约：失败时也必须返回非 nil 的 *Response。
// 调用方（api.AgentEnsure / service.UpdateGpAgent）在错误分支里会读 resp.Output
// 打日志，而它们跑在后台 goroutine 里 —— 一次 nil 解引用就是整个面板进程退出。
func TestDoNeverReturnsNilResponse(t *testing.T) {
	cases := []struct {
		name   string
		socket string
		action string
	}{
		{name: "空 action", action: ""},
		// 用户机器上的真实场景：helper socket 不存在，dial 直接失败
		{name: "socket 不存在", socket: filepath.Join(t.TempDir(), "missing.sock"), action: "GOPANEL_AGENT_ENSURE"},
		{name: "socket 是普通文件", socket: mustFile(t), action: "GOPANEL_AGENT_INSTALL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.socket != "" {
				t.Setenv("GPC_SOCKET_PATH", tc.socket)
			}
			resp, err := Do(context.Background(), tc.action, nil)
			if err == nil {
				t.Fatalf("这些用例都应该失败，却成功了: %+v", resp)
			}
			if resp == nil {
				t.Fatal("Do 在失败时返回了 nil *Response —— 调用方读 resp.Output 会 panic")
			}
			// 调用方最常见的写法，必须不 panic
			_ = resp.Output
		})
	}
}

func mustFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "not-a-socket")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatalf("准备测试文件失败: %v", err)
	}
	return p
}
