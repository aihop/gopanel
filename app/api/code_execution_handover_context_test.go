package api

import (
	"testing"
	"time"
)

// 终端一族的生命周期挂在 done channel 上，而追 rollout 的 follower 收的是 context。
// 这个桥接错了的话，follower 要么永远不退出（终端关了还在追磁盘文件、每 200ms 一次），
// 要么一上来就退出（对话页永远看不到终端产出）。
func TestContextFromDone(t *testing.T) {
	t.Run("done 未关闭时保持存活", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)
		ctx := contextFromDone(done)
		select {
		case <-ctx.Done():
			t.Fatal("done 还没关，context 不该结束")
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("done 关闭后随之取消", func(t *testing.T) {
		done := make(chan struct{})
		ctx := contextFromDone(done)
		close(done)
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
			t.Fatal("done 已关闭，context 应当被取消，否则 follower 会一直追下去")
		}
	})

	// 终端可能在 follower 起来之前就退出了，这种情况也要立刻收敛。
	t.Run("传入已关闭的 done 立即取消", func(t *testing.T) {
		done := make(chan struct{})
		close(done)
		ctx := contextFromDone(done)
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
			t.Fatal("传入已关闭的 channel 应立即取消")
		}
	})
}
