package api

import (
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// 闸门的判定结果。分开命名而不是返回 bool，是为了让日志能说清
// 「这次为什么没抽」——否则用户只会看到记忆迟迟不出现，无从判断是坏了还是没到条件。
const (
	codeMemoryGateAllow        = "allow"
	codeMemoryGateLowSignal    = "low_signal"
	codeMemoryGateNotEnoughNew = "insufficient_growth"
)

// 显式触发标记。用户直接打「记住：这个项目一律用 pnpm」就绕过两层闸门立刻抽取。
//
// 这是整套机制里唯一让用户能主动干预的口子：其余判定都是启发式的，
// 总会有它判错的时候，得留一个「我说了算」的说法。
var codeMemoryTriggerMarkers = []string{
	"记住：", "记住:", "#记住", "/记住",
	"remember:", "remember ", "#remember", "/remember",
	"user memory:", "project memory:",
}

// 值得沉淀的内容通常带这些词。命中不代表一定有记忆可抽，
// 但一条都不命中的对话（纯粹的"跑一下测试""看看这个文件"）几乎必然抽不出东西——
// 与其花一次模型调用去确认，不如先挡掉。
var codeMemorySignalTerms = []string{
	// 中文
	"约定", "决定", "以后", "总是", "不要", "必须", "偏好", "习惯",
	"修复", "报错", "错误", "问题", "坑", "配置", "命令", "发布", "规范", "注意",
	// 英文
	"always", "never", "prefer", "decided", "decision", "convention", "rule",
	"standard", "remember", "fix", "bug", "regression", "error", "failed",
	"warning", "config", "setting", "command", "release", "deploy",
}

func codeMemoryTextHasTriggerMarker(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		lowered := strings.ToLower(strings.TrimSpace(line))
		for _, marker := range codeMemoryTriggerMarkers {
			if strings.HasPrefix(lowered, marker) {
				return true
			}
		}
	}
	return false
}

// codeMemoryTextHasSignal 判断这段对话里有没有值得抽的东西。
func codeMemoryTextHasSignal(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if codeMemoryTextHasTriggerMarker(trimmed) {
		return true
	}
	lowered := strings.ToLower(trimmed)
	for _, term := range codeMemorySignalTerms {
		if strings.Contains(lowered, term) {
			return true
		}
	}
	// 关键词之外，出现具体的路径、命令或标识符也算有料：
	// 「这个项目的入口在 cmd/server/main.go」不含任何关键词，但值得记。
	return codeMemoryTextLooksTechnical(trimmed)
}

func codeMemoryTextLooksTechnical(text string) bool {
	for _, token := range strings.Fields(text) {
		trimmed := strings.Trim(token, ",;:\"'()[]{}。，、")
		if len(trimmed) < 4 {
			continue
		}
		// 路径
		if strings.Contains(trimmed, "/") && !strings.HasPrefix(trimmed, "http") {
			return true
		}
		// 带扩展名的文件、包名
		if strings.Contains(trimmed, ".") && !strings.HasSuffix(trimmed, ".") {
			return true
		}
		// 环境变量或常量式标识符
		if trimmed == strings.ToUpper(trimmed) && strings.Contains(trimmed, "_") {
			return true
		}
	}
	return false
}

// evaluateCodeMemoryGate 决定这次要不要真的抽。
//
// 两层：先看内容有没有信号（零成本），再看距上次抽取新增够不够多。
// 显式触发标记两层都绕过——用户主动说了要记，就不该被启发式挡回去。
func evaluateCodeMemoryGate(sessionID uint, transcript string, threshold int, newMessages int) string {
	if codeMemoryTextHasTriggerMarker(transcript) {
		return codeMemoryGateAllow
	}
	if !codeMemoryTextHasSignal(transcript) {
		return codeMemoryGateLowSignal
	}
	if threshold <= 0 {
		return codeMemoryGateAllow
	}
	// 首次抽取时没有基线，直接放行：一个会话总要先有第一次。
	if newMessages < 0 {
		return codeMemoryGateAllow
	}
	if newMessages < threshold {
		return codeMemoryGateNotEnoughNew
	}
	return codeMemoryGateAllow
}

// loadCodeMemoryExtractionState 取会话上次抽到哪条消息。
// 返回 -1 表示从未抽过，调用方据此跳过增量判断。
func loadCodeMemoryExtractionState(sessionID uint) int64 {
	if global.DB == nil || sessionID == 0 {
		return -1
	}
	var state model.AICodeMemoryExtractionState
	if err := global.DB.Where("session_id = ?", sessionID).First(&state).Error; err != nil {
		return -1
	}
	return int64(state.LastMessageID)
}

// saveCodeMemoryExtractionState 记下这次抽到哪条。
// 抽取成功才写：失败后重来应当重新读同一批消息，而不是跳过它们。
func saveCodeMemoryExtractionState(sessionID uint, lastMessageID uint) {
	if global.DB == nil || sessionID == 0 || lastMessageID == 0 {
		return
	}
	state := model.AICodeMemoryExtractionState{SessionID: sessionID, LastMessageID: lastMessageID}
	err := global.DB.Where("session_id = ?", sessionID).
		Assign(map[string]any{"last_message_id": lastMessageID}).
		FirstOrCreate(&state).Error
	if err != nil {
		warnCodeDelivery("Save Code memory extraction state for session %d failed: %v", sessionID, err)
	}
}
