package api

import "testing"

// 让模型自评重要性，它会把什么都标成重要。所以 tier 由 kind 推导：
// 「踩过的坑」和「定下的规矩」是重复犯错代价最高的两类。
func TestCodeMemoryTierIsDerivedFromKind(t *testing.T) {
	core := map[string]string{
		codeMemoryKindBugLesson:  codeMemoryTierCore,
		codeMemoryKindDecision:   codeMemoryTierCore,
		codeMemoryKindFact:       codeMemoryTierWorking,
		codeMemoryKindPreference: codeMemoryTierWorking,
	}
	for kind, expected := range core {
		if actual := codeMemoryTierForKind(kind); actual != expected {
			t.Fatalf("%s 应推导为 %s，实际 %s", kind, expected, actual)
		}
	}
}

// 合并只能让记忆更重要，不能更次要——否则一条 core 被 working 合掉之后
// 就悄悄从注入里消失了。
func TestPreferredCodeMemoryTierNeverDowngrades(t *testing.T) {
	cases := []struct{ existing, candidate, expected string }{
		{codeMemoryTierCore, codeMemoryTierWorking, codeMemoryTierCore},
		{codeMemoryTierWorking, codeMemoryTierCore, codeMemoryTierCore},
		{codeMemoryTierWorking, codeMemoryTierArchive, codeMemoryTierWorking},
		{codeMemoryTierArchive, codeMemoryTierWorking, codeMemoryTierWorking},
		{codeMemoryTierArchive, codeMemoryTierArchive, codeMemoryTierArchive},
	}
	for _, testCase := range cases {
		actual := preferredCodeMemoryTier(testCase.existing, testCase.candidate)
		if actual != testCase.expected {
			t.Fatalf("%s + %s 应为 %s，实际 %s",
				testCase.existing, testCase.candidate, testCase.expected, actual)
		}
	}
}

// 结构化输出限定了枚举，但模型仍会给出别的写法。归一比拒绝划算：
// 丢一条记忆的代价高于容忍一次措辞偏差。
func TestNormalizeCodeMemoryScopeAndKindAcceptVariants(t *testing.T) {
	userScopes := []string{"user", "USER", "global", "cross_project", "developer"}
	for _, value := range userScopes {
		if normalizeCodeMemoryScope(value) != codeMemoryScopeUser {
			t.Fatalf("%q 应归一为 user", value)
		}
	}
	// 拿不准一律落到 project：project 记忆的影响面比 user 小，
	// 归错方向的代价也就更小。
	for _, value := range []string{"project", "repo", "", "什么鬼"} {
		if normalizeCodeMemoryScope(value) != codeMemoryScopeProject {
			t.Fatalf("%q 应归一为 project", value)
		}
	}
	kinds := map[string]string{
		"preference": codeMemoryKindPreference, "style": codeMemoryKindPreference,
		"decision": codeMemoryKindDecision, "convention": codeMemoryKindDecision,
		"bug_lesson": codeMemoryKindBugLesson, "regression": codeMemoryKindBugLesson,
		"fact": codeMemoryKindFact, "": codeMemoryKindFact,
	}
	for value, expected := range kinds {
		if actual := normalizeCodeMemoryKind(value); actual != expected {
			t.Fatalf("%q 应归一为 %s，实际 %s", value, expected, actual)
		}
	}
}

// 模块名会进 Markdown 标题和分组键，放任自由文本会让分组彻底散掉。
func TestNormalizeCodeMemoryModuleKeyCollapsesFreeText(t *testing.T) {
	cases := map[string]string{
		"Frontend":         "frontend",
		"  Git Delivery ":  "git-delivery",
		"app/api/delivery": "app-api-delivery",
		"前端":               "general",
		"":                 "general",
		"---":              "general",
	}
	for input, expected := range cases {
		if actual := normalizeCodeMemoryModuleKey(input, codeMemoryScopeProject); actual != expected {
			t.Fatalf("%q 应归一为 %q，实际 %q", input, expected, actual)
		}
	}
	// user 作用域没有模块可言，留空会让注入时出现一个无名分组。
	if actual := normalizeCodeMemoryModuleKey("whatever", codeMemoryScopeUser); actual != codeMemoryUserModuleKey {
		t.Fatalf("user 作用域应固定模块名，实际 %q", actual)
	}
	long := normalizeCodeMemoryModuleKey("abcdefghij0123456789abcdefghij0123456789abcdefghij0123456789", codeMemoryScopeProject)
	if len([]rune(long)) > 48 {
		t.Fatalf("模块名应限长，实际 %d", len([]rune(long)))
	}
}
