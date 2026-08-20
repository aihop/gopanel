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
