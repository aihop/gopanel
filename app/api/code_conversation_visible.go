package api

import (
	"strings"

	"github.com/aihop/gopanel/app/model"
)

var injectedConversationMarkers = []string{
	"\n[GoPanel Git 交付约束]",
	"\n[GoPanel 长期记忆]",
	"\n[GoPanel 工作区路径]",
	"[GoPanel Git 交付约束]",
	"[GoPanel 长期记忆]",
	"[GoPanel 工作区路径]",
}

// Codex CLI 会往用户轮次里塞自己的上下文块（工作区根目录、权限档案等）。
// 那是给模型看的脚手架，不是用户说过的话——原样显示出来，用户会以为自己发了一堆 XML。
// 这里按标签名整块剔除，加新标签只需往这个列表里补一行。
var injectedConversationTags = []string{"environment_context"}

// stripTagBlock 剔除 content 里所有 <tag>…</tag> 整块，包括跨行的。
func stripTagBlock(content, tag string) string {
	openTag, closeTag := "<"+tag+">", "</"+tag+">"
	for {
		start := strings.Index(content, openTag)
		if start < 0 {
			return content
		}
		offset := strings.Index(content[start:], closeTag)
		if offset < 0 {
			// 只有开标签没有闭标签（内容被截断）：从开标签起全部丢掉。
			// 宁可少显示一段，也不要把脚手架漏给用户看。
			return content[:start]
		}
		content = content[:start] + content[start+offset+len(closeTag):]
	}
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
	for _, tag := range injectedConversationTags {
		content = stripTagBlock(content, tag)
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
	return collapseConversationDuplicates(visible)
}

func collapseConversationDuplicates(messages []*model.AIMessage) []*model.AIMessage {
	runBacked := make([]*model.AIMessage, 0, len(messages))
	for _, message := range messages {
		if message != nil && message.Role != "user" && message.RunID != 0 {
			runBacked = append(runBacked, message)
		}
	}
	collapsed := make([]*model.AIMessage, 0, len(messages))
	for _, message := range messages {
		if shouldDropConversationDuplicate(message, collapsed, runBacked) {
			continue
		}
		collapsed = append(collapsed, message)
	}
	return collapsed
}

func shouldDropConversationDuplicate(message *model.AIMessage, seen, runBacked []*model.AIMessage) bool {
	content := strings.TrimSpace(message.Content)
	if message.Role == "user" {
		for _, existing := range seen {
			if existing.Role == "user" && strings.TrimSpace(existing.Content) == content && conversationTimesClose(existing, message) {
				return true
			}
		}
		return false
	}
	for _, existing := range runBacked {
		if existing == message || (existing.ID != 0 && existing.ID == message.ID) {
			continue
		}
		target := strings.TrimSpace(existing.Content)
		if content == target {
			return message.RunID == 0 || message.NativeID != ""
		}
		if message.RunID == 0 && isConversationFragment(content, target) {
			return true
		}
	}
	if len(seen) > 0 {
		last := seen[len(seen)-1]
		if last.Role != "user" && strings.TrimSpace(last.Content) == content {
			return true
		}
	}
	return false
}

func isConversationFragment(fragment, complete string) bool {
	if fragment == "" || complete == "" || fragment == complete {
		return fragment == complete
	}
	if strings.HasPrefix(complete, fragment) {
		return true
	}
	return len(fragment) >= 24 && strings.Contains(complete, fragment)
}

func conversationTimesClose(left, right *model.AIMessage) bool {
	if left.CreatedAt.IsZero() || right.CreatedAt.IsZero() {
		return true
	}
	difference := left.CreatedAt.Sub(right.CreatedAt)
	if difference < 0 {
		difference = -difference
	}
	return difference <= nativeHistoryDuplicateWindow
}
