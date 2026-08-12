package api

import (
	"strings"
	"testing"
)

// 「记住：」是整套启发式里唯一让用户说了算的口子，
// 它必须能穿过低信号和增量两层闸门。
func TestCodeMemoryTriggerMarkerBypassesBothGates(t *testing.T) {
	for _, text := range []string{
		"记住：这个项目一律用 pnpm",
		"记住:不要动 main 分支",
		"#记住 交付前先跑测试",
		"/记住 用 tab 缩进",
		"Remember: always run gofmt",
		"REMEMBER: never force push",
		"  记住：前面有空格也算",
		"前面一行\n记住：第二行也认",
	} {
		if !codeMemoryTextHasTriggerMarker(text) {
			t.Fatalf("应识别为显式触发：%q", text)
		}
		// 内容再没信号、增量再不够，也要放行。
		if gate := evaluateCodeMemoryGate(1, text, 100, 0); gate != codeMemoryGateAllow {
			t.Fatalf("显式触发应绕过闸门：%q -> %s", text, gate)
		}
	}
}

func TestCodeMemoryTriggerMarkerDoesNotFireOnOrdinaryText(t *testing.T) {
	for _, text := range []string{
		"我记住了这个问题",            // 「记住」不在行首
		"请帮我看看这个报错",           // 有信号但不是显式触发
		"remembering the fix", // 词形不同
		"",
	} {
		if codeMemoryTextHasTriggerMarker(text) {
			t.Fatalf("不该识别为显式触发：%q", text)
		}
	}
}

// 一条信号都没有的对话（"跑一下测试""看看这个文件"）几乎必然抽不出东西，
// 花一次模型调用去确认它没东西可抽是纯粹的浪费。
func TestCodeMemoryLowSignalTextIsGatedBeforeAnyModelCall(t *testing.T) {
	lowSignal := []string{
		"好的",
		"继续",
		"user: 你好\n\nagent: 你好，有什么可以帮你",
		"再试一次",
	}
	for _, text := range lowSignal {
		if codeMemoryTextHasSignal(text) {
			t.Fatalf("不该判为有信号：%q", text)
		}
		if gate := evaluateCodeMemoryGate(1, text, 0, -1); gate != codeMemoryGateLowSignal {
			t.Fatalf("低信号应被挡下：%q -> %s", text, gate)
		}
	}
}

func TestCodeMemorySignalDetectionAcceptsDurableContent(t *testing.T) {
	signal := []string{
		"以后这个项目都用 pnpm",
		"这个报错是因为并发槽满了",
		"约定：交付前必须过质量门禁",
		"We decided to always use --no-ff",
		// 不含任何关键词，但有具体路径，值得记。
		"入口在 cmd/server/main.go",
		"设置 GOPANEL_CODE_MAX_DELIVERY_CONCURRENCY 可以放开并发",
	}
	for _, text := range signal {
		if !codeMemoryTextHasSignal(text) {
			t.Fatalf("应判为有信号：%q", text)
		}
	}
}

// 闸门的核心价值：一个 8 轮的会话不该触发 8 次模型调用。
func TestCodeMemoryGrowthGateSkipsUntilEnoughNewMessages(t *testing.T) {
	const threshold = 8
	text := "以后这个项目都用 pnpm"
	if gate := evaluateCodeMemoryGate(1, text, threshold, 3); gate != codeMemoryGateNotEnoughNew {
		t.Fatalf("新增不足应跳过：%s", gate)
	}
	if gate := evaluateCodeMemoryGate(1, text, threshold, 7); gate != codeMemoryGateNotEnoughNew {
		t.Fatalf("差一条也应跳过：%s", gate)
	}
	if gate := evaluateCodeMemoryGate(1, text, threshold, 8); gate != codeMemoryGateAllow {
		t.Fatalf("达到阈值应放行：%s", gate)
	}
	// 首次抽取没有基线（-1），一个会话总要先有第一次。
	if gate := evaluateCodeMemoryGate(1, text, threshold, -1); gate != codeMemoryGateAllow {
		t.Fatalf("首次抽取应放行：%s", gate)
	}
	// 阈值为 0 表示关掉增量闸门，回到每次执行都抽。
	if gate := evaluateCodeMemoryGate(1, text, 0, 1); gate != codeMemoryGateAllow {
		t.Fatalf("阈值为 0 应放行：%s", gate)
	}
}

// 闸门结果要能说清「为什么没抽」，否则用户只看到记忆迟迟不出现，
// 无从判断是坏了还是没到条件。
func TestCodeMemoryGateResultsAreDistinguishable(t *testing.T) {
	results := map[string]bool{
		codeMemoryGateAllow:        true,
		codeMemoryGateLowSignal:    true,
		codeMemoryGateNotEnoughNew: true,
	}
	if len(results) != 3 {
		t.Fatal("三种判定结果必须互不相同")
	}
	for result := range results {
		if strings.TrimSpace(result) == "" {
			t.Fatal("判定结果不能为空字符串")
		}
	}
}

func TestNormalizeCodeMemoryGrowthThresholdClampsRange(t *testing.T) {
	cases := map[int]int{
		-1:  codeMemoryDefaultGrowthThreshold,
		0:   0, // 0 是合法的：显式表达「每次都抽」
		8:   8,
		100: 100,
		999: 100,
	}
	for input, expected := range cases {
		if actual := normalizeCodeMemoryGrowthThreshold(input); actual != expected {
			t.Fatalf("%d 应收敛为 %d，实际 %d", input, expected, actual)
		}
	}
}
