package api

import (
	"testing"

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
