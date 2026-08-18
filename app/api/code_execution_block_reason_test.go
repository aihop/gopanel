package api

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func deliverySession(id uint, sourceDir string) *model.AIDevSession {
	return &model.AIDevSession{ID: id, SourceWorkDir: sourceDir, WorkDir: "/tmp/session_" + sourceDir}
}

// 交付槽和执行槽分开之后，跑满 AI 执行不该再挡住交付。
// 合池时这里会卡到超时——那正是实测里最大的一类失败。
func TestDeliveryIsNotStarvedBySaturatedExecutions(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(1, 1)
	busy, err := coordinator.acquireOwned(
		context.Background(), 1, []string{"/repo/other-session-worktree"}, codeExecutionInstruction, false,
	)
	if err != nil {
		t.Fatalf("执行应能占到槽：%v", err)
	}
	defer busy.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	lease, err := coordinator.acquireOwned(ctx, 2, []string{"/repo/source"}, codeExecutionDelivery, true)
	if err != nil {
		t.Fatalf("执行槽被占满不该挡住交付：%v", err)
	}
	lease.Release()
}

// 交付槽自身满了仍然要排队，否则质量检查会把机器压垮。
func TestDeliveryStillRespectsItsOwnCapacity(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(4, 1)
	first, err := coordinator.acquireOwned(
		context.Background(), 1, []string{"/repo/a"}, codeExecutionDelivery, false,
	)
	if err != nil {
		t.Fatalf("首个交付应能占到槽：%v", err)
	}
	defer first.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if _, err := coordinator.acquireOwned(ctx, 2, []string{"/repo/b"}, codeExecutionDelivery, true); err == nil {
		t.Fatal("交付槽已满时第二个交付应排队至超时")
	}
}

// 归还时必须还回原来的池子。还错池会让另一个池凭空多出容量，
// 几轮之后并发上限就完全失控。
func TestLeaseReturnsSlotToItsOwnPool(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(1, 1)
	delivery, err := coordinator.acquireOwned(
		context.Background(), 1, []string{"/repo/a"}, codeExecutionDelivery, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	delivery.Release()
	if len(coordinator.deliveryCapacity) != 0 {
		t.Fatalf("交付槽应被归还，实际占用 %d", len(coordinator.deliveryCapacity))
	}
	if len(coordinator.capacity) != 0 {
		t.Fatalf("交付归还不该动到执行池，实际占用 %d", len(coordinator.capacity))
	}
	execution, err := coordinator.acquireOwned(
		context.Background(), 2, []string{"/repo/b"}, codeExecutionInstruction, false,
	)
	if err != nil {
		t.Fatalf("执行池容量被交付污染了：%v", err)
	}
	execution.Release()
}

func TestDeliveryBlockReasonPointsAtTheActualHolder(t *testing.T) {
	t.Run("本会话占用", func(t *testing.T) {
		coordinator := newCodeExecutionCoordinator(2, 2)
		session := deliverySession(7, "/repo/own")
		lease, err := coordinator.acquireOwned(
			context.Background(), session.ID, []string{"/repo/own"}, codeExecutionInstruction, false,
		)
		if err != nil {
			t.Fatal(err)
		}
		defer lease.Release()
		if reason := coordinator.deliveryBlockReason(session); !errors.Is(reason, errCodeDeliveryWorkspaceBusy) {
			t.Fatalf("应指向本会话：%v", reason)
		}
	})

	t.Run("其他会话占用", func(t *testing.T) {
		coordinator := newCodeExecutionCoordinator(2, 2)
		session := deliverySession(7, "/repo/shared")
		lease, err := coordinator.acquireOwned(
			context.Background(), 99, []string{"/repo/shared"}, codeExecutionDelivery, false,
		)
		if err != nil {
			t.Fatal(err)
		}
		defer lease.Release()
		reason := coordinator.deliveryBlockReason(session)
		if !errors.Is(reason, errCodeDeliveryRepositoryBusy) {
			t.Fatalf("应指向其他会话：%v", reason)
		}
		// 占用方的会话号要能看见，否则用户不知道该等谁。
		if !strings.Contains(reason.Error(), "#99") {
			t.Fatalf("提示里应包含占用方会话号：%v", reason)
		}
	})

	t.Run("并发槽满", func(t *testing.T) {
		coordinator := newCodeExecutionCoordinator(2, 1)
		lease, err := coordinator.acquireOwned(
			context.Background(), 99, []string{"/repo/elsewhere"}, codeExecutionDelivery, false,
		)
		if err != nil {
			t.Fatal(err)
		}
		defer lease.Release()
		// 键不冲突，只有槽位耗尽——不能再让用户去停自己的会话。
		reason := coordinator.deliveryBlockReason(deliverySession(7, "/repo/mine"))
		if !errors.Is(reason, errCodeDeliveryCapacityBusy) {
			t.Fatalf("应指向并发上限：%v", reason)
		}
		if errors.Is(reason, errCodeDeliveryWorkspaceBusy) {
			t.Fatal("并发槽满不该被报成本会话占用")
		}
	})
}

func TestCodeDeliveryFailureCodesDistinguishBlockSources(t *testing.T) {
	cases := map[string]error{
		"workspace_busy":  errCodeDeliveryWorkspaceBusy,
		"repository_busy": errCodeDeliveryRepositoryBusy,
		"capacity_busy":   errCodeDeliveryCapacityBusy,
	}
	for expected, err := range cases {
		if actual := codeDeliveryFailureCode(codeDeliveryStageQueued, err); actual != expected {
			t.Fatalf("%v 应映射为 %q，实际 %q", err, expected, actual)
		}
	}
}

func TestCodeDeliveryConcurrencyFallsBackToASafeDefault(t *testing.T) {
	t.Setenv("GOPANEL_CODE_MAX_DELIVERY_CONCURRENCY", "")
	if value := codeDeliveryConcurrency(); value != 2 {
		t.Fatalf("默认交付并发应为 2，实际 %d", value)
	}
	t.Setenv("GOPANEL_CODE_MAX_DELIVERY_CONCURRENCY", "0")
	if value := codeDeliveryConcurrency(); value != 2 {
		t.Fatalf("非法值应回落到默认值，实际 %d", value)
	}
	t.Setenv("GOPANEL_CODE_MAX_DELIVERY_CONCURRENCY", "999")
	if value := codeDeliveryConcurrency(); value != 32 {
		t.Fatalf("应封顶到 32，实际 %d", value)
	}
	t.Setenv("GOPANEL_CODE_MAX_DELIVERY_CONCURRENCY", "6")
	if value := codeDeliveryConcurrency(); value != 6 {
		t.Fatalf("应读取环境变量，实际 %d", value)
	}
}

// 两个终端在同一个目录各干各的，本来就是日常操作。面板在自己界面里拦一道
// 也挡不住风险——SSH 上去开两个终端跑同一个 CLI 谁也拦不住，拦只会让面板
// 比裸终端更难用。
func TestInteractiveTerminalsShareTheSameWorkspace(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(2, 2)
	first, err := coordinator.acquireOwned(
		context.Background(), 1, []string{"/project/src"}, codeExecutionInteractive, false,
	)
	if err != nil {
		t.Fatalf("首个终端应能打开：%v", err)
	}
	defer first.Release()

	second, err := coordinator.acquireOwned(
		context.Background(), 2, []string{"/project/src"}, codeExecutionInteractive, false,
	)
	if err != nil {
		t.Fatalf("同一工作区的第二个终端也应能打开：%v", err)
	}
	defer second.Release()
}

// 放行终端之后仍要能看见「还有终端在跑」：交付据此决定要不要等。
// 单值 map 时后来的终端会覆盖先来的登记，先来的那条就被遗忘了。
func TestConcurrentTerminalsRemainVisibleToDelivery(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(2, 2)
	first, err := coordinator.acquireOwned(
		context.Background(), 1, []string{"/project/src"}, codeExecutionInteractive, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := coordinator.acquireOwned(
		context.Background(), 2, []string{"/project/src"}, codeExecutionInteractive, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	// 后开的先关：先开的那条必须仍然登记在案。
	second.Release()
	if !coordinator.hasSessionKind(1, codeExecutionInteractive) {
		t.Fatal("后开的终端关闭后，先开的那条不该被一起遗忘")
	}
	first.Release()
	if coordinator.hasSessionKind(1, codeExecutionInteractive) {
		t.Fatal("全部关闭后不该还留着登记")
	}
}

// 交付要把提交合进源仓库，那是一次真正的原子写；并发写坏的是 Git 对象
// 和分支指针，不是「两个人各改各的文件」。所以这条边界必须保留。
func TestDeliveryStillExcludesRunningWork(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(2, 2)
	terminal, err := coordinator.acquireOwned(
		context.Background(), 1, []string{"/repo"}, codeExecutionInteractive, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Release()
	if _, err := coordinator.acquireOwned(
		context.Background(), 2, []string{"/repo"}, codeExecutionDelivery, false,
	); err == nil {
		t.Fatal("终端还开着时交付不该直接放行")
	}
	// 反向也要挡住：交付进行中不能再起终端去动同一棵树。
	delivery := newCodeExecutionCoordinator(2, 2)
	held, err := delivery.acquireOwned(context.Background(), 1, []string{"/repo"}, codeExecutionDelivery, false)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	if _, err := delivery.acquireOwned(
		context.Background(), 2, []string{"/repo"}, codeExecutionInteractive, false,
	); err == nil {
		t.Fatal("交付进行中不该允许新终端占用同一工作区")
	}
}

func TestCodeExecutionCoexistsOnlyForTerminals(t *testing.T) {
	if !codeExecutionCoexists(codeExecutionInteractive, codeExecutionInteractive) {
		t.Fatal("终端之间应可共存")
	}
	for _, kind := range []string{codeExecutionDelivery, codeExecutionInstruction, codeExecutionMutation, codeExecutionQuality} {
		if codeExecutionCoexists(codeExecutionInteractive, kind) {
			t.Fatalf("终端不应与 %s 共存", kind)
		}
		if codeExecutionCoexists(kind, codeExecutionInteractive) {
			t.Fatalf("%s 不应与终端共存", kind)
		}
	}
}
