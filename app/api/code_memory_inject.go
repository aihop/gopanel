package api

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// 注入的条数上限。记忆的价值在于「提醒」，不是「复述整个项目」——
// 灌太多会把真正的任务指令挤到上下文边缘，反而降低执行质量。
const (
	codeMemoryInjectMaxEntries = 24
	codeMemoryInjectMaxRunes   = 4000
)

// loadCodeMemoryForInjection 取本次执行要回灌的记忆。
//
// 取 user 作用域（跨项目）+ 本项目的 project 作用域，core 优先、
// 同档按最近更新排。core 先取满是刻意的：坑和规矩比零散事实更值得占位置。
func loadCodeMemoryForInjection(userID, projectID uint) ([]model.AICodeMemoryEntry, string) {
	if global.DB == nil || userID == 0 {
		return nil, ""
	}
	var entries []model.AICodeMemoryEntry
	query := global.DB.Where("user_id = ? AND status = ?", userID, codeMemoryStatusActive).
		Where("scope = ? OR (scope = ? AND project_id = ?)",
			codeMemoryScopeUser, codeMemoryScopeProject, projectID)
	// core 排前面：ORDER BY 里直接用 tier 的字典序会把 archive 排到最前，
	// 所以用 CASE 显式给权重。
	if err := query.
		Order("CASE tier WHEN 'core' THEN 0 WHEN 'working' THEN 1 ELSE 2 END").
		Order("updated_at DESC").
		Limit(codeMemoryInjectMaxEntries).Find(&entries).Error; err != nil {
		warnCodeDelivery("Load Code memory for user %d project %d failed: %v", userID, projectID, err)
		return nil, ""
	}
	var summary model.AICodeMemorySummary
	if err := global.DB.Where("user_id = ?", userID).First(&summary).Error; err != nil {
		return entries, ""
	}
	return entries, summary.Content
}

// renderCodeMemoryContext 把记忆渲染成注入用的文本。
//
// 按模块分组而不是平铺：AI 读到「delivery 模块有这三条约定」比读到
// 二十条零散事实更容易用上。分组内保持 core 在前。
func renderCodeMemoryContext(entries []model.AICodeMemoryEntry, summary string) string {
	if len(entries) == 0 && strings.TrimSpace(summary) == "" {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("\n\n[GoPanel 长期记忆]\n")
	builder.WriteString("以下是此前会话沉淀下来的约定与教训。与当前任务冲突时以用户的当前指令为准。\n")
	if trimmed := strings.TrimSpace(summary); trimmed != "" {
		builder.WriteString("\n## 用户画像\n")
		builder.WriteString(trimmed)
		builder.WriteString("\n")
	}
	for _, group := range groupCodeMemoryByModule(entries) {
		fmt.Fprintf(&builder, "\n## %s\n", group.module)
		for _, entry := range group.entries {
			fmt.Fprintf(&builder, "- [%s] %s\n", entry.Kind, strings.TrimSpace(entry.Content))
		}
	}
	rendered := builder.String()
	if len([]rune(rendered)) > codeMemoryInjectMaxRunes {
		rendered = string([]rune(rendered)[:codeMemoryInjectMaxRunes]) + "\n…（记忆已截断）\n"
	}
	return rendered
}

// flattenCodeMemoryInInjectionOrder 按注入时的真实顺序摊平记忆。
//
// 界面照这个顺序展示，「你看到的就是 AI 看到的」才成立。界面自己另排一套
// 顺序的话，用户就没法从列表推理 AI 为什么这么干——而那正是他打开这个
// 列表的唯一原因。
func flattenCodeMemoryInInjectionOrder(entries []model.AICodeMemoryEntry) []model.AICodeMemoryEntry {
	ordered := make([]model.AICodeMemoryEntry, 0, len(entries))
	for _, group := range groupCodeMemoryByModule(entries) {
		ordered = append(ordered, group.entries...)
	}
	return ordered
}

type codeMemoryModuleGroup struct {
	module  string
	entries []model.AICodeMemoryEntry
}

func groupCodeMemoryByModule(entries []model.AICodeMemoryEntry) []codeMemoryModuleGroup {
	byModule := make(map[string][]model.AICodeMemoryEntry)
	for _, entry := range entries {
		module := entry.ModuleKey
		if module == "" {
			module = "general"
		}
		byModule[module] = append(byModule[module], entry)
	}
	groups := make([]codeMemoryModuleGroup, 0, len(byModule))
	for module, moduleEntries := range byModule {
		sort.SliceStable(moduleEntries, func(left, right int) bool {
			return codeMemoryTierWeight(moduleEntries[left].Tier) < codeMemoryTierWeight(moduleEntries[right].Tier)
		})
		groups = append(groups, codeMemoryModuleGroup{module: module, entries: moduleEntries})
	}
	// user 分组固定排最前：跨项目的习惯适用于所有任务。
	sort.SliceStable(groups, func(left, right int) bool {
		if (groups[left].module == codeMemoryUserModuleKey) != (groups[right].module == codeMemoryUserModuleKey) {
			return groups[left].module == codeMemoryUserModuleKey
		}
		return groups[left].module < groups[right].module
	})
	return groups
}

func codeMemoryTierWeight(tier string) int {
	switch tier {
	case codeMemoryTierCore:
		return 0
	case codeMemoryTierWorking:
		return 1
	default:
		return 2
	}
}

// codeMemoryPrompt 把记忆接到执行提示词后面。
//
// 与 codeManagedDeliveryPrompt 并列：两者都是"在用户指令之外补充平台约束"，
// 放在同一层，顺序上记忆在后——离用户指令越远的内容权重越低，
// 交付约束是硬规则，记忆是软提示。
func codeMemoryPrompt(session *model.AIDevSession, prompt string) string {
	if session == nil || session.UserID == 0 {
		return prompt
	}
	entries, summary := loadCodeMemoryForInjection(session.UserID, session.ProjectID)
	context := renderCodeMemoryContext(entries, summary)
	if context == "" {
		return prompt
	}
	return strings.TrimRight(prompt, "\n") + context
}
