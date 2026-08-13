package api

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func withCodeMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "memory.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&model.AICodeMemoryEntry{},
		&model.AICodeMemorySummary{},
		&model.AICodeMemoryExtractionState{},
		&model.AICodeMemoryAuditEvent{},
	); err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
	return database
}

func memoryItem(content, kind, scope, module string) codeMemoryExtractionItem {
	return codeMemoryExtractionItem{Content: content, Kind: kind, Scope: scope, ModuleKey: module}
}

func applyContext() codeMemoryApplyContext {
	return codeMemoryApplyContext{UserID: 3, ProjectID: 9, SessionID: 41}
}

func TestApplyCodeMemoryExtractionAddsEntriesWithDerivedTier(t *testing.T) {
	withCodeMemoryDB(t)
	response := codeMemoryExtractionResponse{
		UserSummary: "偏好中文回复",
		WorkingAdd: []codeMemoryExtractionItem{
			memoryItem("交付默认走 direct 模式", codeMemoryKindDecision, codeMemoryScopeProject, "delivery"),
			memoryItem("npm run lint 会重排全仓", codeMemoryKindBugLesson, codeMemoryScopeProject, "frontend"),
			memoryItem("喜欢先看测试再看实现", codeMemoryKindPreference, codeMemoryScopeUser, "user"),
		},
	}
	result, err := applyCodeMemoryExtraction(response, applyContext())
	if err != nil || result.Added != 3 {
		t.Fatalf("三条都应写入：%#v, %v", result, err)
	}
	var entries []model.AICodeMemoryEntry
	if err := global.DB.Order("id").Find(&entries).Error; err != nil {
		t.Fatal(err)
	}
	if entries[0].Tier != codeMemoryTierCore || entries[1].Tier != codeMemoryTierCore {
		t.Fatalf("decision 和 bug_lesson 应进 core：%s, %s", entries[0].Tier, entries[1].Tier)
	}
	if entries[2].Tier != codeMemoryTierWorking {
		t.Fatalf("preference 应进 working：%s", entries[2].Tier)
	}
	// user 作用域跨项目生效，不能绑定到某个项目上。
	if entries[2].ProjectID != 0 {
		t.Fatalf("user 记忆不该绑定项目：%d", entries[2].ProjectID)
	}
	if entries[0].ProjectID != 9 {
		t.Fatalf("project 记忆应绑定项目：%d", entries[0].ProjectID)
	}
	var summary model.AICodeMemorySummary
	if err := global.DB.Where("user_id = ?", 3).First(&summary).Error; err != nil || summary.Content != "偏好中文回复" {
		t.Fatalf("用户画像未写入：%#v, %v", summary, err)
	}
}

// 判重是整个引擎能不能用的关键：不合并的话，每轮都会把同一条偏好
// 重新写一遍，几轮之后注入的全是同义重复。
func TestApplyCodeMemoryExtractionMergesIntoExistingEntry(t *testing.T) {
	withCodeMemoryDB(t)
	existing := model.AICodeMemoryEntry{
		UserID: 3, ProjectID: 9, Scope: codeMemoryScopeProject, Kind: codeMemoryKindFact,
		Tier: codeMemoryTierWorking, ModuleKey: "delivery", Content: "旧的说法",
		Status: codeMemoryStatusActive,
	}
	if err := global.DB.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	item := memoryItem("更准确的说法", codeMemoryKindDecision, codeMemoryScopeProject, "delivery")
	item.MergeWith = strconv.FormatUint(uint64(existing.ID), 10)
	result, err := applyCodeMemoryExtraction(codeMemoryExtractionResponse{
		WorkingAdd: []codeMemoryExtractionItem{item},
	}, applyContext())
	if err != nil || result.Merged != 1 || result.Added != 0 {
		t.Fatalf("应合并而不是新增：%#v, %v", result, err)
	}
	var reloaded model.AICodeMemoryEntry
	if err := global.DB.First(&reloaded, existing.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.Content != "更准确的说法" {
		t.Fatalf("合并应更新内容：%q", reloaded.Content)
	}
	// 合并只能让记忆更重要，不能更次要。
	if reloaded.Tier != codeMemoryTierCore {
		t.Fatalf("合并后应升到 core：%s", reloaded.Tier)
	}
	var count int64
	global.DB.Model(&model.AICodeMemoryEntry{}).Count(&count)
	if count != 1 {
		t.Fatalf("不该产生新条目，实际 %d 条", count)
	}
}

func TestApplyCodeMemoryExtractionReplacesAndArchives(t *testing.T) {
	withCodeMemoryDB(t)
	stale := model.AICodeMemoryEntry{
		UserID: 3, ProjectID: 9, Scope: codeMemoryScopeProject, Kind: codeMemoryKindFact,
		Tier: codeMemoryTierWorking, ModuleKey: "git", Content: "过时的事实", Status: codeMemoryStatusActive,
	}
	duplicate := stale
	duplicate.Content = "重复的事实"
	if err := global.DB.Create(&stale).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&duplicate).Error; err != nil {
		t.Fatal(err)
	}
	item := memoryItem("新的事实", codeMemoryKindFact, codeMemoryScopeProject, "git")
	item.Replace = strconv.FormatUint(uint64(stale.ID), 10)
	item.Archive = []string{strconv.FormatUint(uint64(duplicate.ID), 10)}
	result, err := applyCodeMemoryExtraction(codeMemoryExtractionResponse{
		WorkingAdd: []codeMemoryExtractionItem{item},
	}, applyContext())
	if err != nil || result.Added != 1 || result.Replaced != 1 || result.Archived != 1 {
		t.Fatalf("应新增并归档两条旧记忆：%#v, %v", result, err)
	}
	var reloaded model.AICodeMemoryEntry
	if err := global.DB.First(&reloaded, stale.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != codeMemoryStatusArchived || reloaded.SupersededBy == 0 {
		t.Fatalf("被取代的记忆应归档并指向新条目：%#v", reloaded)
	}
	// 归档而不是删除：留一条记录才能看出这条曾经存在过。
	var count int64
	global.DB.Model(&model.AICodeMemoryEntry{}).Count(&count)
	if count != 3 {
		t.Fatalf("旧记忆应保留为归档，实际 %d 条", count)
	}
}

// id 是模型给出来的，它会编号码。少了 user_id 过滤，一次幻觉就能改写别人的记忆库。
func TestApplyCodeMemoryExtractionIgnoresOtherUsersEntries(t *testing.T) {
	withCodeMemoryDB(t)
	foreign := model.AICodeMemoryEntry{
		UserID: 99, ProjectID: 9, Scope: codeMemoryScopeProject, Kind: codeMemoryKindFact,
		Tier: codeMemoryTierWorking, ModuleKey: "git", Content: "别人的记忆", Status: codeMemoryStatusActive,
	}
	if err := global.DB.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	item := memoryItem("我的记忆", codeMemoryKindFact, codeMemoryScopeProject, "git")
	item.MergeWith = strconv.FormatUint(uint64(foreign.ID), 10)
	item.Archive = []string{strconv.FormatUint(uint64(foreign.ID), 10)}
	result, err := applyCodeMemoryExtraction(codeMemoryExtractionResponse{
		WorkingAdd: []codeMemoryExtractionItem{item},
	}, applyContext())
	if err != nil {
		t.Fatal(err)
	}
	if result.Merged != 0 || result.Archived != 0 || result.Added != 1 {
		t.Fatalf("跨用户引用应被忽略并退化为新增：%#v", result)
	}
	var reloaded model.AICodeMemoryEntry
	if err := global.DB.First(&reloaded, foreign.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.Content != "别人的记忆" || reloaded.Status != codeMemoryStatusActive {
		t.Fatalf("别人的记忆被改写了：%#v", reloaded)
	}
}

func TestApplyCodeMemoryExtractionIgnoresOtherProjectEntries(t *testing.T) {
	withCodeMemoryDB(t)
	foreignProject := model.AICodeMemoryEntry{
		UserID: 3, ProjectID: 77, Scope: codeMemoryScopeProject, Kind: codeMemoryKindFact,
		Tier: codeMemoryTierWorking, ModuleKey: "git", Content: "另一个项目的记忆", Status: codeMemoryStatusActive,
	}
	if err := global.DB.Create(&foreignProject).Error; err != nil {
		t.Fatal(err)
	}
	item := memoryItem("当前项目的记忆", codeMemoryKindFact, codeMemoryScopeProject, "git")
	item.MergeWith = strconv.FormatUint(uint64(foreignProject.ID), 10)
	item.Archive = []string{strconv.FormatUint(uint64(foreignProject.ID), 10)}
	result, err := applyCodeMemoryExtraction(codeMemoryExtractionResponse{
		WorkingAdd: []codeMemoryExtractionItem{item},
	}, applyContext())
	if err != nil {
		t.Fatal(err)
	}
	if result.Merged != 0 || result.Archived != 0 || result.Added != 1 {
		t.Fatalf("跨项目引用应被忽略并退化为新增：%#v", result)
	}
	var reloaded model.AICodeMemoryEntry
	if err := global.DB.First(&reloaded, foreignProject.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.Content != "另一个项目的记忆" || reloaded.Status != codeMemoryStatusActive {
		t.Fatalf("另一个项目的记忆被改写了：%#v", reloaded)
	}
}

func TestApplyCodeMemoryExtractionCanReferenceUserMemoryAcrossProjects(t *testing.T) {
	withCodeMemoryDB(t)
	userMemory := model.AICodeMemoryEntry{
		UserID: 3, ProjectID: 0, Scope: codeMemoryScopeUser, Kind: codeMemoryKindPreference,
		Tier: codeMemoryTierWorking, ModuleKey: codeMemoryUserModuleKey, Content: "旧偏好", Status: codeMemoryStatusActive,
	}
	if err := global.DB.Create(&userMemory).Error; err != nil {
		t.Fatal(err)
	}
	item := memoryItem("新偏好", codeMemoryKindPreference, codeMemoryScopeUser, codeMemoryUserModuleKey)
	item.MergeWith = strconv.FormatUint(uint64(userMemory.ID), 10)
	result, err := applyCodeMemoryExtraction(codeMemoryExtractionResponse{
		WorkingAdd: []codeMemoryExtractionItem{item},
	}, applyContext())
	if err != nil || result.Merged != 1 || result.Added != 0 {
		t.Fatalf("当前项目抽取应能合并用户级记忆：%#v, %v", result, err)
	}
}

// 空摘要表示「保持不变」。当成清空会让一次无内容的抽取抹掉长期积累的画像。
func TestApplyCodeMemoryExtractionKeepsSummaryWhenEmpty(t *testing.T) {
	withCodeMemoryDB(t)
	if err := global.DB.Create(&model.AICodeMemorySummary{UserID: 3, Content: "已有画像"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := applyCodeMemoryExtraction(codeMemoryExtractionResponse{UserSummary: "   "}, applyContext()); err != nil {
		t.Fatal(err)
	}
	var summary model.AICodeMemorySummary
	if err := global.DB.Where("user_id = ?", 3).First(&summary).Error; err != nil {
		t.Fatal(err)
	}
	if summary.Content != "已有画像" {
		t.Fatalf("空摘要不该清空已有画像：%q", summary.Content)
	}
}

func TestApplyCodeMemoryExtractionAuditsSummaryChanges(t *testing.T) {
	withCodeMemoryDB(t)
	if _, err := applyCodeMemoryExtraction(codeMemoryExtractionResponse{UserSummary: "偏好先看结论"}, applyContext()); err != nil {
		t.Fatal(err)
	}
	var event model.AICodeMemoryAuditEvent
	if err := global.DB.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.UserID != 3 || event.SessionID != 41 || event.Action != "summary_update" || event.Source != "extraction" {
		t.Fatalf("画像审计上下文不完整：%#v", event)
	}
	if event.Before != "" || event.After != "偏好先看结论" {
		t.Fatalf("画像审计内容不正确：%#v", event)
	}
}

func TestFinishCodeMemoryExtractionStatusAdvancesCursorOnlyOnSuccess(t *testing.T) {
	withCodeMemoryDB(t)
	if err := updateCodeMemoryExtractionStatus(41, map[string]any{"last_message_id": 10}); err != nil {
		t.Fatal(err)
	}
	result := codeMemoryApplyResult{Added: 1, Merged: 2, Replaced: 3, Archived: 4}
	if err := finishCodeMemoryExtractionStatus(41, codeMemoryExtractionFailed, "provider failed", result, 25); err != nil {
		t.Fatal(err)
	}
	failed, err := loadCodeMemoryExtractionStatus(41)
	if err != nil {
		t.Fatal(err)
	}
	if failed.LastMessageID != 10 || failed.Status != codeMemoryExtractionFailed || failed.Reason != "provider failed" {
		t.Fatalf("失败状态不应推进游标：%#v", failed)
	}
	if failed.Added != 1 || failed.Merged != 2 || failed.Replaced != 3 || failed.Archived != 4 || failed.CompletedAt == nil {
		t.Fatalf("失败状态应保留结果与完成时间：%#v", failed)
	}
	if err := finishCodeMemoryExtractionStatus(41, codeMemoryExtractionSuccess, "", result, 25); err != nil {
		t.Fatal(err)
	}
	succeeded, err := loadCodeMemoryExtractionStatus(41)
	if err != nil {
		t.Fatal(err)
	}
	if succeeded.LastMessageID != 25 || succeeded.Status != codeMemoryExtractionSuccess {
		t.Fatalf("成功状态应推进游标：%#v", succeeded)
	}
}

func TestLoadCodeMemoryExtractionStateTreatsQueuedRowAsFirstExtraction(t *testing.T) {
	withCodeMemoryDB(t)
	if err := queueCodeMemoryExtractionStatus(41, codeMemoryTriggerAutomatic); err != nil {
		t.Fatal(err)
	}
	if cursor := loadCodeMemoryExtractionState(41); cursor != -1 {
		t.Fatalf("尚未成功抽取的排队状态不应成为游标基线：%d", cursor)
	}
	if err := finishCodeMemoryExtractionStatus(41, codeMemoryExtractionSuccess, "", codeMemoryApplyResult{}, 25); err != nil {
		t.Fatal(err)
	}
	if cursor := loadCodeMemoryExtractionState(41); cursor != 25 {
		t.Fatalf("成功抽取后应返回已保存游标：%d", cursor)
	}
}

func TestRenderCodeMemoryContextGroupsAndPrioritizes(t *testing.T) {
	entries := []model.AICodeMemoryEntry{
		{Scope: codeMemoryScopeProject, Kind: codeMemoryKindFact, Tier: codeMemoryTierWorking, ModuleKey: "git", Content: "零散事实"},
		{Scope: codeMemoryScopeProject, Kind: codeMemoryKindBugLesson, Tier: codeMemoryTierCore, ModuleKey: "git", Content: "踩过的坑"},
		{Scope: codeMemoryScopeUser, Kind: codeMemoryKindPreference, Tier: codeMemoryTierWorking, ModuleKey: codeMemoryUserModuleKey, Content: "跨项目习惯"},
	}
	rendered := renderCodeMemoryContext(entries, "用户画像内容")
	if !strings.Contains(rendered, "用户画像内容") {
		t.Fatalf("画像应被渲染：%s", rendered)
	}
	// user 分组排最前：跨项目习惯适用于所有任务。
	userIndex := strings.Index(rendered, "跨项目习惯")
	gitIndex := strings.Index(rendered, "踩过的坑")
	if userIndex < 0 || gitIndex < 0 || userIndex > gitIndex {
		t.Fatalf("user 分组应排在前面：%s", rendered)
	}
	// 同组内 core 在前。
	if strings.Index(rendered, "踩过的坑") > strings.Index(rendered, "零散事实") {
		t.Fatalf("core 应排在 working 前面：%s", rendered)
	}
}

func TestRenderCodeMemoryContextIsEmptyWithoutMemories(t *testing.T) {
	if rendered := renderCodeMemoryContext(nil, "  "); rendered != "" {
		t.Fatalf("没有记忆时不该注入任何内容：%q", rendered)
	}
}

// 灌太多会把真正的任务指令挤到上下文边缘，反而降低执行质量。
func TestRenderCodeMemoryContextTruncatesOversizedInjection(t *testing.T) {
	entries := make([]model.AICodeMemoryEntry, 0, 40)
	for index := 0; index < 40; index++ {
		entries = append(entries, model.AICodeMemoryEntry{
			Scope: codeMemoryScopeProject, Kind: codeMemoryKindFact, Tier: codeMemoryTierWorking,
			ModuleKey: "m", Content: strings.Repeat("很长的记忆内容", 40),
		})
	}
	rendered := renderCodeMemoryContext(entries, "")
	if len([]rune(rendered)) > codeMemoryInjectMaxRunes+32 {
		t.Fatalf("注入内容应被截断，实际 %d", len([]rune(rendered)))
	}
	if !strings.Contains(rendered, "记忆已截断") {
		t.Fatalf("截断应有提示：%s", rendered[len(rendered)-80:])
	}
}
