package api

import (
	"fmt"
	"strconv"
	"strings"
)

// 抽取是一次确定性的压缩作业，不是一轮对话。系统提示词把这一点说死：
// 只回 JSON、不解释、不调工具、不推断记录之外的东西。模型一旦开始"聊天"，
// 后面的解析就全是特例处理。
const codeMemoryExtractionSystemPrompt = `You extract and compact durable software-engineering memory from AI coding sessions.

Return JSON only.
Do not include markdown fences.
Do not include <think> blocks, reasoning text, analysis, explanations, or prose.
The first non-whitespace character of the response must be "{".
Do not call tools, request scans, browse files, or infer facts outside the provided transcript and existing memory.
Treat this as a deterministic memory compaction job, not a chat response.`

// 一次最多写入几条。上限是刻意压低的：抽取的价值在于「少而准」，
// 放开之后模型会把任务流水也当成记忆存进来，几轮下来注入的上下文就废了。
const codeMemoryMaxAddPerExtraction = 3

// 用户画像的长度预算，单位是 token 的粗略估计。
const codeMemorySummaryTokenBudget = 400

type codeMemoryPromptEntry struct {
	ID        uint
	Scope     string
	Kind      string
	Tier      string
	ModuleKey string
	Content   string
}

// codeMemoryExtractionSchema 是结构化输出的 JSON Schema。
//
// 所有可选字段都列进 required，用空字符串/空数组表示「不使用」：
// 多数 provider 的 structured output 要求键集稳定，让模型自由增删键
// 会换来一堆解析不了的响应。
func codeMemoryExtractionSchema() map[string]any {
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content":     map[string]any{"type": "string"},
			"kind":        map[string]any{"type": "string", "enum": []string{codeMemoryKindPreference, codeMemoryKindDecision, codeMemoryKindFact, codeMemoryKindBugLesson}},
			"scope":       map[string]any{"type": "string", "enum": []string{codeMemoryScopeUser, codeMemoryScopeProject}},
			"module_key":  map[string]any{"type": "string"},
			"rationale":   map[string]any{"type": "string"},
			"merge_with":  map[string]any{"type": "string"},
			"replace":     map[string]any{"type": "string"},
			"archive":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"skip_reason": map[string]any{"type": "string"},
		},
		"required": []string{"content", "kind", "scope", "module_key", "rationale", "merge_with", "replace", "archive", "skip_reason"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"user_summary":     map[string]any{"type": "string"},
			"working_add":      map[string]any{"type": "array", "items": item},
			"working_archive":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"merged_entry_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"user_summary", "working_add", "working_archive", "merged_entry_ids"},
	}
}

type codeMemoryPromptInput struct {
	Transcript      string
	ProjectName     string
	UserSummary     string
	UserMemories    []codeMemoryPromptEntry
	ProjectMemories []codeMemoryPromptEntry
	OutputLanguage  string
}

// buildCodeMemoryExtractionPrompt 拼出抽取用的用户消息。
//
// 已有记忆要一并给出去，否则模型没法判重：不给的话每轮都会把同一条
// 「用 pnpm 不用 npm」重新写一遍，几轮之后记忆里全是同义重复。
func buildCodeMemoryExtractionPrompt(input codeMemoryPromptInput) string {
	language := input.OutputLanguage
	if strings.TrimSpace(language) == "" {
		language = "Chinese"
	}
	projectName := strings.TrimSpace(input.ProjectName)
	if projectName == "" {
		projectName = "(unnamed)"
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "Memory extraction schema: gopanel-memory-v1\nProject: %s\n\n", projectName)
	fmt.Fprintf(&builder, "Existing user summary:\n%s\n\n", renderCodeMemorySummaryForPrompt(input.UserSummary))
	fmt.Fprintf(&builder, "Relevant user memories:\n%s\n\n", renderCodeMemoryEntriesForPrompt(input.UserMemories))
	fmt.Fprintf(&builder, "Relevant project memories:\n%s\n\n", renderCodeMemoryEntriesForPrompt(input.ProjectMemories))
	fmt.Fprintf(&builder, "Transcript:\n<transcript>\n%s\n</transcript>\n\n", input.Transcript)
	builder.WriteString("Return minified JSON only, with no markdown and no line breaks:\n")
	builder.WriteString(`{"user_summary":"","working_add":[],"working_archive":[],"merged_entry_ids":[]}` + "\n\n")
	builder.WriteString(codeMemoryExtractionRules(language))
	return builder.String()
}

func codeMemoryExtractionRules(language string) string {
	return strings.Join([]string{
		"Rules:",
		"- If nothing durable should be stored, return the exact empty JSON shape above.",
		fmt.Sprintf("- Add at most %d working_add items total. Prefer the highest value durable memories only.", codeMemoryMaxAddPerExtraction),
		"- Each working_add item must include content, kind, scope, and module_key. scope must be exactly user or project; module_key must be a non-empty concise module name.",
		"- Optional item fields: rationale, merge_with, replace, archive, skip_reason. The schema requires stable keys, so use empty strings for unused optional string fields and [] for unused optional arrays.",
		"- merge_with and replace must be a single existing entry id string, not an array. If multiple existing memories are duplicates, set merge_with to the best target id and put the other duplicate ids in archive.",
		"- Use merge_with for semantic duplicates, replace when the new memory supersedes an old entry, archive for stale or duplicate entry ids, skip_reason for candidates that should not be stored.",
		fmt.Sprintf("- Write user_summary, working_add.content, rationale, and skip_reason in %s. Preserve code identifiers, file paths, commands, URLs, API names, branch names, model/tool names, and quoted error text exactly. JSON keys and enum values must remain in English.", language),
		"- Keep each working_add.content concise: target 120-220 Chinese characters or 60-110 English words. Summarize naturally; do not hard-truncate, cut mid-sentence, or drop critical qualifiers to hit a number.",
		"- Keep rationale short: one brief sentence.",
		fmt.Sprintf("- user_summary <= about %d tokens; an empty string means keep the existing summary unchanged. If it would exceed the budget, rewrite it compactly instead of truncating.", codeMemorySummaryTokenBudget),
		"- Extract only durable engineering memory. Omit temporary tasks, logs, timestamps, greetings, tool output, generic knowledge, and assistant-invented preferences.",
		"- scope=user only for explicit cross-project user habits/preferences; user entries must use module_key=\"user\".",
		"- Repository facts, commands, release flow, UI decisions, bugs, diagnostics, paths, APIs, and conventions must be scope=project with a concise module_key such as frontend, delivery, git, worktree, quality, release, mobile, or general.",
		"- Ambiguous or low-value information must be omitted.",
		"- kind must be exactly one of four: preference (how the user wants you to work), decision (a chosen approach, architecture, or standing convention), bug_lesson (what broke and the fix or guard), fact (a durable concrete fact: path, command, config, API). Pick the single best fit; do not store low-value items just to fill a kind.",
		"- Do not output a tier field; importance is derived automatically.",
	}, "\n")
}

func renderCodeMemorySummaryForPrompt(summary string) string {
	if strings.TrimSpace(summary) == "" {
		return "(none)"
	}
	return strings.TrimSpace(summary)
}

// renderCodeMemoryEntriesForPrompt 把已有记忆渲染成模型能引用的形式。
// id 必须显式给出：merge_with / replace / archive 都靠它指向具体条目。
func renderCodeMemoryEntriesForPrompt(entries []codeMemoryPromptEntry) string {
	if len(entries) == 0 {
		return "(none)"
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("- id=%s [%s/%s/%s/%s] %s",
			strconv.FormatUint(uint64(entry.ID), 10),
			entry.Scope, entry.Tier, entry.Kind, entry.ModuleKey,
			strings.TrimSpace(entry.Content),
		))
	}
	return strings.Join(lines, "\n")
}
