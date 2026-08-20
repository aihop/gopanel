package api

import (
	"strings"

	"github.com/aihop/gopanel/app/model"
)

var injectedConversationMarkers = []string{
	"\n[GoPanel Git 交付约束]",
	"\n[GoPanel 长期记忆]",
	"[GoPanel Git 交付约束]",
	"[GoPanel 长期记忆]",
}

func stripInjectedConversationPrompt(content string) string {
	cut := -1
	for _, marker := range injectedConversationMarkers {
		index := strings.Index(content, marker)
		if index >= 0 && (cut < 0 || index < cut) {
			cut = index
		}
	}
	if cut >= 0 {
		content = content[:cut]
	}
	return strings.TrimSpace(content)
}

func conversationHistoryMessages(messages []*model.AIMessage) []*model.AIMessage {
	visible := make([]*model.AIMessage, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		role := strings.TrimSpace(message.Role)
		if role == "system" || role == "developer" {
			continue
		}
		stored := *message
		if role == "user" {
			stored.Content = stripInjectedConversationPrompt(stored.Content)
			if stored.Content == "" {
				continue
			}
		}
		visible = append(visible, &stored)
	}
	return visible
}
