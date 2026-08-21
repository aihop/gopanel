package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// 执行期间对话里原先只有一个转圈的「运行中」。这些事件是唯一能回答
// 「此刻在干什么」的来源，认错或认漏都会让状态栏继续空着。
func TestCodexActivityFromNotification(t *testing.T) {
	cases := []struct {
		name   string
		method string
		params string
		kind   string
		detail string
	}{
		{"命令执行", "item/started", `{"item":{"type":"commandExecution","command":"go test ./..."}}`, "command", "go test ./..."},
		{"文件改动", "item/completed", `{"item":{"type":"fileChange","path":"app/api/foo.go"}}`, "file", "app/api/foo.go"},
		{"工具调用", "item/started", `{"item":{"type":"mcpToolCall","name":"search"}}`, "tool", "search"},
		{"思考中", "item/started", `{"item":{"type":"reasoning"}}`, "thinking", ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			kind, detail := codexActivityFromNotification(testCase.method, json.RawMessage(testCase.params))
			if kind != testCase.kind || detail != testCase.detail {
				t.Fatalf("期望 (%q,%q)，实际 (%q,%q)", testCase.kind, testCase.detail, kind, detail)
			}
		})
	}
}

func TestCodexActivityStaysSilentWhenUnrecognised(t *testing.T) {
	// 文字增量走正文那条路，不该同时被当成活动状态——否则每个 token 都会刷一次状态栏。
	t.Run("忽略文字增量", func(t *testing.T) {
		kind, _ := codexActivityFromNotification("item/agentMessage/delta", json.RawMessage(`{"delta":"hi"}`))
		if kind != "" {
			t.Fatalf("文字增量不该产生活动状态，实际 %q", kind)
		}
	})

	// 认不出的类型宁可不显示，也不要冒出一句看不懂的状态。
	t.Run("认不出的类型返回空", func(t *testing.T) {
		kind, _ := codexActivityFromNotification("item/started", json.RawMessage(`{"item":{"type":"somethingNew"}}`))
		if kind != "" {
			t.Fatalf("未知类型应静默，实际 %q", kind)
		}
	})

	t.Run("非 item 事件不参与", func(t *testing.T) {
		kind, _ := codexActivityFromNotification("turn/completed", json.RawMessage(`{}`))
		if kind != "" {
			t.Fatalf("非 item 事件不该产生活动状态，实际 %q", kind)
		}
	})

	t.Run("载荷解不开时不报错也不显示", func(t *testing.T) {
		kind, _ := codexActivityFromNotification("item/started", json.RawMessage(`不是 JSON`))
		if kind != "" {
			t.Fatalf("坏载荷应静默，实际 %q", kind)
		}
	})
}

// 命令常常是多行脚本，状态栏放不下；全塞进去反而看不清在跑什么。
func TestCodexActivityDetailKeepsOneReadableLine(t *testing.T) {
	item := codexActivityItem{Type: "commandExecution", Command: "cd /repo && \\\n  go build ./..."}
	if detail := codexActivityDetail(item); detail != "cd /repo && \\" {
		t.Fatalf("应只取首行，实际 %q", detail)
	}

	long := codexActivityItem{Type: "commandExecution", Command: strings.Repeat("字", 200)}
	detail := codexActivityDetail(long)
	if !strings.HasSuffix(detail, "…") {
		t.Fatalf("超长命令应截断并加省略号，实际 %q", detail)
	}
	if runes := []rune(detail); len(runes) != codexActivityDetailLimit+1 {
		t.Fatalf("截断长度应按字符算（中文不能截半个），实际 %d 个字符", len(runes))
	}
}
