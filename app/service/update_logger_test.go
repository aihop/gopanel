package service

import (
	"strings"
	"sync"
	"testing"

	"github.com/aihop/gopanel/global"
)

// 日志文件路径是 <TmpDir>/install_logs/，测试里 TmpDir 为空会直接写进源码目录，
// 所以每个用到 logger 的用例都先把 TmpDir 指到临时目录
func useTempLogDir(t *testing.T) {
	t.Helper()
	old := global.CONF.System.TmpDir
	global.CONF.System.TmpDir = t.TempDir()
	t.Cleanup(func() { global.CONF.System.TmpDir = old })
}

// 同一秒内点两次按钮不能再拿到同一个 logger：
// 共用 logger 时先结束的那个会 close channel，另一个还在写日志的任务就会
// "send on closed channel" panic，而它在后台 goroutine 里，panic = 面板进程退出。
func TestNewUpdateLogNameIsUnique(t *testing.T) {
	seen := make(map[string]bool, 200)
	for i := 0; i < 200; i++ {
		name := NewUpdateLogName("gp_agent_ensure")
		if seen[name] {
			t.Fatalf("日志名重复: %s", name)
		}
		seen[name] = true
		if !strings.HasPrefix(name, "gp_agent_ensure_") || !strings.HasSuffix(name, ".log") {
			t.Fatalf("日志名格式不对: %s", name)
		}
	}
}

// Append/SetStatus 与 RemoveUpdateLogger 并发时不能 panic（旧实现是先取 listeners
// 快照再在锁外发送，Remove 在这中间 close 掉 channel 就会 panic）。
// 用 -race 跑这个用例同时能验证没有数据竞争。
func TestUpdateLoggerConcurrentAppendAndRemove(t *testing.T) {
	useTempLogDir(t)
	for round := 0; round < 50; round++ {
		name := NewUpdateLogName("concurrent_probe")
		logger := GetUpdateLogger(name)
		sub := logger.Subscribe()

		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				logger.Append("writing", i)
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				logger.SetStatus("running")
			}
		}()
		go func() {
			defer wg.Done()
			RemoveUpdateLogger(name)
		}()
		wg.Wait()

		// 重复 Remove 必须是幂等的（SafeGo 的兜底回调里也会再调一次）
		RemoveUpdateLogger(name)

		// 订阅方最终一定能读到 channel 关闭而不是永久阻塞
		drained := false
		for range sub {
			drained = true
			_ = drained
		}
	}
}

// 任务已结束后再订阅，必须立刻拿到一个已关闭的 channel，
// 否则 SSE 处理协程会挂在一个永远不会被 close 的 channel 上。
func TestSubscribeAfterRemoveDoesNotBlock(t *testing.T) {
	useTempLogDir(t)
	name := NewUpdateLogName("closed_probe")
	logger := GetUpdateLogger(name)
	RemoveUpdateLogger(name)

	done := make(chan struct{})
	go func() {
		for range logger.Subscribe() {
		}
		close(done)
	}()
	<-done
}

// SafeGo 必须把后台任务的 panic 吃掉：这里如果没兜住，整个测试进程会挂
func TestSafeGoRecoversPanic(t *testing.T) {
	got := make(chan error, 1)
	SafeGo("panic-probe", func() {
		panic("boom")
	}, func(err error) {
		got <- err
	})
	err := <-got
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("没有拿到 panic 信息: %v", err)
	}
}
