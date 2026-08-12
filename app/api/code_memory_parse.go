package api

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

// 单条记忆的长度上限。提示词里已经要求模型自己控制长度，这里是兜底：
// 偶尔会收到把整段 diff 塞进 content 的响应，存进去会污染之后每一次注入。
const codeMemoryContentMaxRunes = 600
const codeMemoryRationaleMaxRunes = 200
const codeMemorySummaryMaxRunes = 2000

type codeMemoryExtractionItem struct {
	Content    string   `json:"content"`
	Kind       string   `json:"kind"`
	Scope      string   `json:"scope"`
	ModuleKey  string   `json:"module_key"`
	Rationale  string   `json:"rationale"`
	MergeWith  string   `json:"merge_with"`
	Replace    string   `json:"replace"`
	Archive    []string `json:"archive"`
	SkipReason string   `json:"skip_reason"`
}

type codeMemoryExtractionResponse struct {
	UserSummary    string                     `json:"user_summary"`
	WorkingAdd     []codeMemoryExtractionItem `json:"working_add"`
	WorkingArchive []string                   `json:"working_archive"`
	MergedEntryIDs []string                   `json:"merged_entry_ids"`
}

// parseCodeMemoryExtractionResponse 解析模型响应。
//
// 即便要求了「第一个字符必须是 {」，实际仍会收到套着 ```json 围栏、
// 前面带一段 <think> 的响应。与其让整次抽取白跑，不如把 JSON 从中捞出来。
func parseCodeMemoryExtractionResponse(raw string) (codeMemoryExtractionResponse, error) {
	payload, err := extractCodeMemoryJSONObject(raw)
	if err != nil {
		return codeMemoryExtractionResponse{}, err
	}
	var response codeMemoryExtractionResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		return codeMemoryExtractionResponse{}, errors.New("记忆抽取响应不是有效 JSON")
	}
	return normalizeCodeMemoryExtractionResponse(response), nil
}

// extractCodeMemoryJSONObject 从可能带噪音的文本里取出最外层 JSON 对象。
// 用括号配平而不是找首尾大括号：content 里可能带 } 字符。
func extractCodeMemoryJSONObject(raw string) (string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", errors.New("记忆抽取响应为空")
	}
	if index := strings.Index(text, "</think>"); index >= 0 {
		text = strings.TrimSpace(text[index+len("</think>"):])
	}
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(strings.TrimSpace(text), "```")
	start := strings.Index(text, "{")
	if start < 0 {
		return "", errors.New("记忆抽取响应中没有 JSON 对象")
	}
	depth := 0
	inString := false
	escaped := false
	for index := start; index < len(text); index++ {
		character := text[index]
		if inString {
			switch {
			case escaped:
				escaped = false
			case character == '\\':
				escaped = true
			case character == '"':
				inString = false
			}
			continue
		}
		switch character {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : index+1], nil
			}
		}
	}
	return "", errors.New("记忆抽取响应中的 JSON 对象不完整")
}

func normalizeCodeMemoryExtractionResponse(response codeMemoryExtractionResponse) codeMemoryExtractionResponse {
	response.UserSummary = truncateCodeMemoryText(scrubCodeMemoryText(response.UserSummary), codeMemorySummaryMaxRunes)
	items := make([]codeMemoryExtractionItem, 0, len(response.WorkingAdd))
	for _, item := range response.WorkingAdd {
		normalized, ok := normalizeCodeMemoryExtractionItem(item)
		if !ok {
			continue
		}
		items = append(items, normalized)
		if len(items) >= codeMemoryMaxAddPerExtraction {
			break
		}
	}
	response.WorkingAdd = items
	response.WorkingArchive = normalizeCodeMemoryIDList(response.WorkingArchive)
	response.MergedEntryIDs = normalizeCodeMemoryIDList(response.MergedEntryIDs)
	return response
}

func normalizeCodeMemoryExtractionItem(item codeMemoryExtractionItem) (codeMemoryExtractionItem, bool) {
	// 模型给了 skip_reason 就说明它自己判断这条不该存，尊重这个判断。
	if strings.TrimSpace(item.SkipReason) != "" {
		return codeMemoryExtractionItem{}, false
	}
	// 再脱敏一次。记录进去之前已经洗过，但模型可能把洗剩的片段重新拼出来。
	item.Content = truncateCodeMemoryText(scrubCodeMemoryText(item.Content), codeMemoryContentMaxRunes)
	if strings.TrimSpace(item.Content) == "" {
		return codeMemoryExtractionItem{}, false
	}
	item.Scope = normalizeCodeMemoryScope(item.Scope)
	item.Kind = normalizeCodeMemoryKind(item.Kind)
	item.ModuleKey = normalizeCodeMemoryModuleKey(item.ModuleKey, item.Scope)
	item.Rationale = truncateCodeMemoryText(scrubCodeMemoryText(item.Rationale), codeMemoryRationaleMaxRunes)
	item.MergeWith = normalizeCodeMemoryID(item.MergeWith)
	item.Replace = normalizeCodeMemoryID(item.Replace)
	item.Archive = normalizeCodeMemoryIDList(item.Archive)
	return item, true
}

// normalizeCodeMemoryID 只接受能解析成正整数的引用。
// 模型偶尔会回 "none"、"null" 或整条内容，放行它们会让 apply 阶段
// 去更新一个不存在的条目，或更糟——把 0 当成通配。
func normalizeCodeMemoryID(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	id, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || id == 0 {
		return ""
	}
	return strconv.FormatUint(id, 10)
}

func normalizeCodeMemoryIDList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeCodeMemoryID(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func truncateCodeMemoryText(text string, limit int) string {
	trimmed := strings.TrimSpace(text)
	if utf8.RuneCountInString(trimmed) <= limit {
		return trimmed
	}
	return strings.TrimSpace(string([]rune(trimmed)[:limit])) + "…"
}
