package api

import "strings"

// 记忆的三个维度。取值故意保持窄：模型只在 Scope 和 Kind 上做选择，
// Tier 由 Kind 推导——让模型自评重要性，它会把什么都标成重要。
const (
	codeMemoryScopeUser    = "user"
	codeMemoryScopeProject = "project"

	codeMemoryKindPreference = "preference"
	codeMemoryKindDecision   = "decision"
	codeMemoryKindFact       = "fact"
	codeMemoryKindBugLesson  = "bug_lesson"

	codeMemoryTierCore    = "core"
	codeMemoryTierWorking = "working"
	codeMemoryTierArchive = "archive"

	codeMemoryStatusActive   = "active"
	codeMemoryStatusArchived = "archived"

	// user 作用域的记忆统一挂在这个模块下：跨项目的习惯没有模块可言，
	// 留空会让注入时的分组出现一个无名组。
	codeMemoryUserModuleKey = "user"
)

// normalizeCodeMemoryToken 把模型可能给出的各种写法归一。
// 结构化输出已经限定了枚举，但模型仍会偶尔给出 "user preference" 这类值，
// 归一比拒绝更划算——丢一条记忆的代价高于容忍一次措辞偏差。
func normalizeCodeMemoryToken(value string) string {
	var builder strings.Builder
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if character >= 'a' && character <= 'z' {
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func normalizeCodeMemoryScope(value string) string {
	switch normalizeCodeMemoryToken(value) {
	case "user", "global", "developer", "crossproject", "userpreference":
		return codeMemoryScopeUser
	default:
		return codeMemoryScopeProject
	}
}

func normalizeCodeMemoryKind(value string) string {
	switch normalizeCodeMemoryToken(value) {
	case "preference", "preferences", "userpreference", "style", "workflow", "habit":
		return codeMemoryKindPreference
	case "decision", "decisions", "convention", "standard", "rule", "architecture":
		return codeMemoryKindDecision
	case "buglesson", "bug", "lesson", "incident", "pitfall", "regression":
		return codeMemoryKindBugLesson
	default:
		return codeMemoryKindFact
	}
}

// codeMemoryTierForKind 由类别推导重要性。
//
// 「踩过的坑」和「定下的规矩」是重复犯错代价最高的两类，进 core；
// 其余进 working。core 在注入时优先且不轻易被挤掉。
func codeMemoryTierForKind(kind string) string {
	switch kind {
	case codeMemoryKindBugLesson, codeMemoryKindDecision:
		return codeMemoryTierCore
	default:
		return codeMemoryTierWorking
	}
}

// preferredCodeMemoryTier 在合并两条记忆时取更高的那一档。
// 合并只会让记忆更重要，不会更次要——否则一条 core 被 working 合掉之后
// 就悄悄从注入里消失了。
func preferredCodeMemoryTier(existing, candidate string) string {
	if existing == codeMemoryTierCore || candidate == codeMemoryTierCore {
		return codeMemoryTierCore
	}
	if existing == codeMemoryTierWorking || candidate == codeMemoryTierWorking {
		return codeMemoryTierWorking
	}
	return codeMemoryTierArchive
}

// normalizeCodeMemoryModuleKey 收敛模块名：小写、只留字母数字和横线、限长。
// 模块名会进 Markdown 标题和分组键，放任自由文本会让分组彻底散掉。
func normalizeCodeMemoryModuleKey(value, scope string) string {
	if scope == codeMemoryScopeUser {
		return codeMemoryUserModuleKey
	}
	var builder strings.Builder
	lastDash := false
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9'):
			builder.WriteRune(character)
			lastDash = false
		case character == '-' || character == '_' || character == ' ' || character == '/':
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		return "general"
	}
	if len([]rune(normalized)) > 48 {
		normalized = strings.Trim(string([]rune(normalized)[:48]), "-")
	}
	return normalized
}
