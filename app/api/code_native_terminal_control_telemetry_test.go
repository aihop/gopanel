package api

import (
	"testing"
	"time"
)

// 租约实测大量「空闲 60 秒自动过期」，但客户端每 15 秒就发一次 ping。
// 过期审计必须能区分这两种完全不同的原因，否则只能靠猜。
func TestControlTelemetryDistinguishesHeartbeatFailureModes(t *testing.T) {
	acquiredAt := time.Date(2026, 8, 20, 20, 43, 3, 0, time.UTC)
	expiredAt := acquiredAt.Add(60 * time.Second)

	// 心跳一次都没到：连接早就断了，或者 ping 根本没走到续租那条路。
	t.Run("心跳从未到达", func(t *testing.T) {
		terminal := &nativeCodeTerminal{controlAcquiredAt: acquiredAt}
		meta := terminal.controlTelemetryLocked(expiredAt)
		if meta["heartbeats"] != 0 {
			t.Fatalf("心跳数应为 0，实际 %v", meta["heartbeats"])
		}
		if _, exists := meta["sinceLastHeartbeatMs"]; exists {
			t.Fatal("从未收到心跳时不该有 sinceLastHeartbeatMs，那会让人误以为心跳来过")
		}
		if meta["heldMs"] != int64(60000) {
			t.Fatalf("持有时长应为 60000ms，实际 %v", meta["heldMs"])
		}
	})

	// 心跳来过又突然停了：典型是浏览器把后台标签页的定时器节流到分钟级。
	t.Run("心跳中断", func(t *testing.T) {
		terminal := &nativeCodeTerminal{
			controlAcquiredAt:    acquiredAt,
			controlHeartbeats:    3,
			controlLastHeartbeat: acquiredAt.Add(45 * time.Second),
		}
		meta := terminal.controlTelemetryLocked(expiredAt)
		if meta["heartbeats"] != 3 {
			t.Fatalf("心跳数应为 3，实际 %v", meta["heartbeats"])
		}
		if meta["sinceLastHeartbeatMs"] != int64(15000) {
			t.Fatalf("距上次心跳应为 15000ms，实际 %v", meta["sinceLastHeartbeatMs"])
		}
	})
}

// 换了控制者要重新起算，否则过期时读到的是上一任的心跳数，
// 诊断数据会指向完全错误的方向。
func TestTakeControlResetsHeartbeatTelemetry(t *testing.T) {
	terminal := &nativeCodeTerminal{
		subscribers:          map[string]*nativeTerminalSubscription{"a": {}, "b": {}},
		controllerID:         "a",
		controlHeartbeats:    7,
		controlLastHeartbeat: time.Now(),
		cols:                 80,
		rows:                 24,
	}
	// 上一任租约已过期，接管才是合法的——没过期时拒绝接管是另一条规则，不在这里验。
	terminal.controlExpiresAt = time.Now().Add(-time.Second)
	if granted, reason := terminal.takeControl("b", 0, 0); !granted {
		t.Fatalf("应能接管：%s", reason)
	}
	if terminal.controlHeartbeats != 0 {
		t.Fatalf("换控制者后心跳数应清零，实际 %d", terminal.controlHeartbeats)
	}
	if !terminal.controlLastHeartbeat.IsZero() {
		t.Fatal("换控制者后上次心跳时间应清空")
	}
}
