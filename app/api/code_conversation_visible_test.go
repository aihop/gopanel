package api

import (
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func TestStripInjectedConversationPrompt(t *testing.T) {
	user := "检查登录接口"
	wrapped := user + "\n\n[GoPanel Git 交付约束]\n不要 git push\n\n[GoPanel 长期记忆]\n- 用 Go"
	if got := stripInjectedConversationPrompt(wrapped); got != user {
		t.Fatalf("stripped = %q, want %q", got, user)
	}
	if got := stripInjectedConversationPrompt(user + "\n\n@attach src/main.go"); got != user+"\n\n@attach src/main.go" {
		t.Fatalf("attachments must stay visible to the parser: %q", got)
	}
}

func TestConversationHistoryMessagesHidesDuplicates(t *testing.T) {
	now := time.Now()
	messages := conversationHistoryMessages([]*model.AIMessage{
		{ID: 1, CreatedAt: now, Role: "user", Content: "修登录"},
		{ID: 2, CreatedAt: now.Add(time.Millisecond), Role: "user", Content: "修登录", NativeID: "native-user"},
		{ID: 3, CreatedAt: now.Add(time.Second), Role: "agent", RunID: 9, Content: "已经修好登录接口"},
		{ID: 4, CreatedAt: now.Add(2 * time.Second), Role: "agent", NativeID: "chunk", Content: "已经修好"},
		{ID: 5, CreatedAt: now.Add(3 * time.Second), Role: "agent", NativeID: "full", Content: "已经修好登录接口"},
	})
	if len(messages) != 2 {
		t.Fatalf("expected one user and one agent, got %#v", messages)
	}
	if messages[0].Content != "修登录" || messages[1].RunID != 9 {
		t.Fatalf("unexpected collapsed messages: %#v", messages)
	}
}

func TestConversationHistoryMessagesHidesSystemPrompt(t *testing.T) {
	messages := conversationHistoryMessages([]*model.AIMessage{
		{ID: 1, Role: "user", Content: "修测试\n\n[GoPanel 长期记忆]\nfoo"},
		{ID: 2, Role: "developer", Content: "注入的系统提示"},
		{ID: 3, Role: "agent", Content: "已经修好了"},
		{ID: 4, Role: "user", Content: "[GoPanel Git 交付约束]\n不要 push"},
	})
	if len(messages) != 2 {
		t.Fatalf("messages = %#v", messages)
	}
	if messages[0].Content != "修测试" || messages[1].Role != "agent" {
		t.Fatalf("unexpected visible messages: %#v", messages)
	}
}

// Codex 会在用户轮次里注入 <environment_context>（工作区根目录、权限档案等）。
// 那是给模型看的脚手架，露到对话里用户会以为自己发了一堆 XML。
func TestStripInjectedConversationPromptRemovesCodexEnvironmentContext(t *testing.T) {
	envBlock := "<environment_context>\n<current_date>2026-08-20</current_date>\n" +
		"<workspace_roots>/Users/hugh/.gopanel/code/user_1/project_2</workspace_roots>" +
		"<permission_profile type=\"managed\"><file_system type=\"restricted\">:root</file_system>" +
		"</permission_profile>\n</environment_context>"

	t.Run("整块剔除后保留真正的用户输入", func(t *testing.T) {
		if got := stripInjectedConversationPrompt(envBlock + "\n\n把登录接口改成 JWT"); got != "把登录接口改成 JWT" {
			t.Fatalf("环境上下文应被完整剔除，实际 %q", got)
		}
	})

	// 只有脚手架、没有正文时要返回空——上层据此把这条消息整个丢掉。
	t.Run("只有脚手架时返回空", func(t *testing.T) {
		if got := stripInjectedConversationPrompt(envBlock); got != "" {
			t.Fatalf("应返回空以便整条丢弃，实际 %q", got)
		}
	})

	// 内容被截断、只剩开标签时，宁可少显示也不能把脚手架漏出来。
	t.Run("闭标签缺失时从开标签起全部丢掉", func(t *testing.T) {
		if got := stripInjectedConversationPrompt("先看这个\n<environment_context>\n<current_date>2026"); got != "先看这个" {
			t.Fatalf("残缺标签应整段丢弃，实际 %q", got)
		}
	})

	// 不能误伤：普通文本里出现尖括号不该被动。
	t.Run("不误伤普通尖括号", func(t *testing.T) {
		normal := "泛型写成 List<String> 就行"
		if got := stripInjectedConversationPrompt(normal); got != normal {
			t.Fatalf("普通文本不该被改动，实际 %q", got)
		}
	})
}
