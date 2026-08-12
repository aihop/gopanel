package api

import (
	"strings"
	"testing"
)

// 即便系统提示词要求「第一个字符必须是 {」，实际仍会收到套围栏、
// 带 <think> 的响应。整次抽取因此白跑不值得。
func TestParseCodeMemoryExtractionResponseSurvivesNoisyOutput(t *testing.T) {
	payload := `{"user_summary":"偏好 pnpm","working_add":[{"content":"交付默认走 direct 模式","kind":"decision","scope":"project","module_key":"delivery","rationale":"减少 PR 环节","merge_with":"","replace":"","archive":[],"skip_reason":""}],"working_archive":[],"merged_entry_ids":[]}`
	noisy := map[string]string{
		"裸 JSON":      payload,
		"markdown 围栏": "```json\n" + payload + "\n```",
		"think 前缀":    "<think>让我想想该存什么</think>\n" + payload,
		"前后空白":        "\n\n  " + payload + "  \n",
	}
	for name, raw := range noisy {
		response, err := parseCodeMemoryExtractionResponse(raw)
		if err != nil {
			t.Fatalf("%s 应能解析：%v", name, err)
		}
		if len(response.WorkingAdd) != 1 || response.WorkingAdd[0].ModuleKey != "delivery" {
			t.Fatalf("%s 解析结果不对：%#v", name, response)
		}
	}
}

// content 里带 } 时，靠首尾大括号截取会把 JSON 切断。
func TestParseCodeMemoryExtractionResponseHandlesBracesInContent(t *testing.T) {
	raw := `{"user_summary":"","working_add":[{"content":"错误信息是 map[string]any{} 解码失败","kind":"bug_lesson","scope":"project","module_key":"api","rationale":"","merge_with":"","replace":"","archive":[],"skip_reason":""}],"working_archive":[],"merged_entry_ids":[]}`
	response, err := parseCodeMemoryExtractionResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.WorkingAdd) != 1 || !strings.Contains(response.WorkingAdd[0].Content, "map[string]any{}") {
		t.Fatalf("含大括号的内容被截断：%#v", response)
	}
}

func TestParseCodeMemoryExtractionResponseRejectsUnusableOutput(t *testing.T) {
	for name, raw := range map[string]string{
		"空响应":      "",
		"纯文本":      "我觉得这次没什么值得记住的",
		"截断的 JSON": `{"user_summary":"abc","working_add":[`,
	} {
		if _, err := parseCodeMemoryExtractionResponse(raw); err == nil {
			t.Fatalf("%s 应报错", name)
		}
	}
}

// 模型自己给了 skip_reason 就说明它判断这条不该存，要尊重。
func TestParseCodeMemoryExtractionResponseDropsSkippedAndEmptyItems(t *testing.T) {
	raw := `{"user_summary":"","working_add":[
		{"content":"这条应被跳过","kind":"fact","scope":"project","module_key":"a","rationale":"","merge_with":"","replace":"","archive":[],"skip_reason":"临时任务"},
		{"content":"   ","kind":"fact","scope":"project","module_key":"b","rationale":"","merge_with":"","replace":"","archive":[],"skip_reason":""},
		{"content":"保留这条","kind":"fact","scope":"project","module_key":"c","rationale":"","merge_with":"","replace":"","archive":[],"skip_reason":""}
	],"working_archive":[],"merged_entry_ids":[]}`
	response, err := parseCodeMemoryExtractionResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.WorkingAdd) != 1 || response.WorkingAdd[0].Content != "保留这条" {
		t.Fatalf("跳过项和空内容应被丢弃：%#v", response.WorkingAdd)
	}
}

// 放开条数后模型会把任务流水也当记忆存进来，几轮下来注入的上下文就废了。
func TestParseCodeMemoryExtractionResponseCapsAddCount(t *testing.T) {
	var items []string
	for index := 0; index < 10; index++ {
		items = append(items, `{"content":"记忆条目","kind":"fact","scope":"project","module_key":"m","rationale":"","merge_with":"","replace":"","archive":[],"skip_reason":""}`)
	}
	raw := `{"user_summary":"","working_add":[` + strings.Join(items, ",") + `],"working_archive":[],"merged_entry_ids":[]}`
	response, err := parseCodeMemoryExtractionResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.WorkingAdd) != codeMemoryMaxAddPerExtraction {
		t.Fatalf("单次写入应被截到 %d 条，实际 %d", codeMemoryMaxAddPerExtraction, len(response.WorkingAdd))
	}
}

// 模型偶尔会回 "none"、"null" 或整条内容当引用。放行它们会让 apply
// 去更新不存在的条目，或把 0 当成通配。
func TestParseCodeMemoryExtractionResponseRejectsBogusEntryReferences(t *testing.T) {
	raw := `{"user_summary":"","working_add":[
		{"content":"内容","kind":"fact","scope":"project","module_key":"m","rationale":"","merge_with":"none","replace":"0","archive":["null","12","12","abc"],"skip_reason":""}
	],"working_archive":["-1","7"],"merged_entry_ids":["0",""]}`
	response, err := parseCodeMemoryExtractionResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	item := response.WorkingAdd[0]
	if item.MergeWith != "" || item.Replace != "" {
		t.Fatalf("非法引用应被清空：merge=%q replace=%q", item.MergeWith, item.Replace)
	}
	if len(item.Archive) != 1 || item.Archive[0] != "12" {
		t.Fatalf("archive 应只留合法且去重的 id：%#v", item.Archive)
	}
	if len(response.WorkingArchive) != 1 || response.WorkingArchive[0] != "7" {
		t.Fatalf("working_archive 过滤不对：%#v", response.WorkingArchive)
	}
	if len(response.MergedEntryIDs) != 0 {
		t.Fatalf("merged_entry_ids 应被清空：%#v", response.MergedEntryIDs)
	}
}

// 记忆会长期存库并反复注入，模型把整段 diff 塞进 content 时必须截断。
func TestParseCodeMemoryExtractionResponseTruncatesOversizedContent(t *testing.T) {
	huge := strings.Repeat("很长的内容", 400)
	raw := `{"user_summary":"","working_add":[
		{"content":"` + huge + `","kind":"fact","scope":"project","module_key":"m","rationale":"","merge_with":"","replace":"","archive":[],"skip_reason":""}
	],"working_archive":[],"merged_entry_ids":[]}`
	response, err := parseCodeMemoryExtractionResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if runes := []rune(response.WorkingAdd[0].Content); len(runes) != codeMemoryContentMaxRunes+1 {
		t.Fatalf("内容应被截断到 %d(+省略号)，实际 %d", codeMemoryContentMaxRunes, len(runes))
	}
}

// 抽取结果会直接落库，模型有可能把记录里洗剩的片段重新拼出来。
func TestParseCodeMemoryExtractionResponseScrubsSecretsInOutput(t *testing.T) {
	raw := `{"user_summary":"用户把 sk-abcdefghijklmnopqrstuvwxyz012345 写在了配置里","working_add":[
		{"content":"部署要用 ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123 这个令牌","kind":"fact","scope":"project","module_key":"deploy","rationale":"","merge_with":"","replace":"","archive":[],"skip_reason":""}
	],"working_archive":[],"merged_entry_ids":[]}`
	response, err := parseCodeMemoryExtractionResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(response.UserSummary, "sk-abcdefghijklmnopqrstuvwxyz012345") {
		t.Fatalf("摘要里的密钥未脱敏：%q", response.UserSummary)
	}
	if strings.Contains(response.WorkingAdd[0].Content, "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123") {
		t.Fatalf("记忆内容里的密钥未脱敏：%q", response.WorkingAdd[0].Content)
	}
}
