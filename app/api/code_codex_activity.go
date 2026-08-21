package api

import (
	"encoding/json"
	"strings"
)

// 执行期间对话里原先只有一个转圈的「运行中」：用户既不知道 AI 在做什么，
// 也判断不了该不该继续等。这里把 app-server 的 item 事件翻成「此刻在干什么」。
//
// 只发种类和细节，不发中文文案——具体措辞交给前端 i18n，
// 免得把界面用语固化进后端。

const codexActivityDetailLimit = 120

type codexActivityItem struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Path    string `json:"path"`
	Name    string `json:"name"`
	Query   string `json:"query"`
}

// codexActivityKind 把 item 类型收敛成前端认识的几种活动。
// 认不出的类型返回空：宁可什么都不显示，也不要冒出一句看不懂的状态。
func codexActivityKind(itemType string) string {
	switch strings.TrimSpace(itemType) {
	case "commandExecution":
		return "command"
	case "fileChange":
		return "file"
	case "mcpToolCall", "toolCall":
		return "tool"
	case "webSearch":
		return "search"
	case "reasoning":
		return "thinking"
	}
	return ""
}

// codexActivityDetail 取这条活动最值得显示的一行。
// 命令可能是多行脚本，只取首行并截断——状态栏放不下，全塞进去反而看不清在跑什么。
func codexActivityDetail(item codexActivityItem) string {
	for _, candidate := range []string{item.Command, item.Path, item.Name, item.Query} {
		value := strings.TrimSpace(candidate)
		if value == "" {
			continue
		}
		if index := strings.IndexByte(value, '\n'); index >= 0 {
			value = strings.TrimSpace(value[:index])
		}
		if len([]rune(value)) > codexActivityDetailLimit {
			value = string([]rune(value)[:codexActivityDetailLimit]) + "…"
		}
		return value
	}
	return ""
}

// codexActivityFromNotification 从一条 item 通知里解出活动描述。
//
// 刻意按 "item/" 前缀匹配而不是枚举事件名：Codex 的事件名会随版本增减，
// 写死一份清单只会在它变动时悄悄失效。解不出结构或认不出类型就返回空，
// 调用方据此跳过。
func codexActivityFromNotification(method string, params json.RawMessage) (kind string, detail string) {
	if !strings.HasPrefix(method, "item/") || strings.HasSuffix(method, "/delta") {
		return "", ""
	}
	var payload struct {
		Item codexActivityItem `json:"item"`
	}
	if json.Unmarshal(params, &payload) != nil {
		return "", ""
	}
	kind = codexActivityKind(payload.Item.Type)
	if kind == "" {
		return "", ""
	}
	return kind, codexActivityDetail(payload.Item)
}
