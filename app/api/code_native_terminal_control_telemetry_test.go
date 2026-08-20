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

// 租约必须扛得住浏览器把后台标签页定时器节流到分钟级：15 秒的 ping 被拉到约
// 60 秒时仍要能续上。定成 60 秒时实测就是在这里被掐掉的，改动这个常量前先看
// 上面那段注释里的实测数据。
func TestControlLeaseSurvivesThrottledHeartbeat(t *testing.T) {
	const throttledHeartbeatGap = 60 * time.Second
	if nativeTerminalControlLease <= throttledHeartbeatGap {
		t.Fatalf("租约 %v 必须大于被节流后的心跳间隔 %v，否则心跳永远追不上",
			nativeTerminalControlLease, throttledHeartbeatGap)
	}

	// 光是「大于」还不够，要留出余量：节流后的间隔本身会抖动。
	if nativeTerminalControlLease < 2*throttledHeartbeatGap {
		t.Fatalf("租约 %v 至少要留出两倍余量（%v），否则抖动一下还是会掉",
			nativeTerminalControlLease, 2*throttledHeartbeatGap)
	}

	// 另一头也要卡住：租约越长，控制方硬断线后别人等得越久。
	if nativeTerminalControlLease > 5*time.Minute {
		t.Fatalf("租约 %v 过长，控制方硬断线后接管等待不可接受", nativeTerminalControlLease)
	}
}

// 续租必须真的把过期时间往后推，而不是只重排计时器。
func TestRenewControlLeaseExtendsExpiry(t *testing.T) {
	terminal := &nativeCodeTerminal{
		subscribers:  map[string]*nativeTerminalSubscription{"a": {}},
		controllerID: "a",
	}
	terminal.controlExpiresAt = time.Now().Add(10 * time.Second)
	before := terminal.controlExpiresAt

	if !terminal.renewControlLease("a") {
		t.Fatal("控制方续租应当成功")
	}
	if !terminal.controlExpiresAt.After(before) {
		t.Fatal("续租后过期时间必须往后推")
	}
	if terminal.controlHeartbeats != 1 {
		t.Fatalf("心跳计数应为 1，实际 %d", terminal.controlHeartbeats)
	}

	// 非控制方的 ping 不能续租，否则只读连接能把控制权钉死在别人手上。
	if terminal.renewControlLease("b") {
		t.Fatal("非控制方不该能续租")
	}
}
