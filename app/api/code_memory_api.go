package api

import (
	"errors"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeMemoryListView struct {
	Entries []model.AICodeMemoryEntry `json:"entries"`
	Summary string                    `json:"summary"`
	Total   int64                     `json:"total"`
}

// GetCodeMemories 列出当前用户的记忆。
//
// 记忆必须可见可删：抽取由模型完成，它一定会记错东西。没有纠正入口的话，
// 一条错误记忆会在之后每次执行时被反复注入，比不记还糟。
func GetCodeMemories(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, _ := strconv.ParseUint(strings.TrimSpace(c.Query("projectId")), 10, 64)
	includeArchived := strings.TrimSpace(c.Query("includeArchived")) == "true"

	query := global.DB.Model(&model.AICodeMemoryEntry{}).Where("user_id = ?", claims.UserId)
	if !includeArchived {
		query = query.Where("status = ?", codeMemoryStatusActive)
	}
	if projectID > 0 {
		query = query.Where("scope = ? OR (scope = ? AND project_id = ?)",
			codeMemoryScopeUser, codeMemoryScopeProject, uint(projectID))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	var entries []model.AICodeMemoryEntry
	if err := query.
		Order("CASE tier WHEN 'core' THEN 0 WHEN 'working' THEN 1 ELSE 2 END").
		Order("updated_at DESC").Limit(200).Find(&entries).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	view := codeMemoryListView{Entries: entries, Total: total}
	var summary model.AICodeMemorySummary
	if err := global.DB.Where("user_id = ?", claims.UserId).First(&summary).Error; err == nil {
		view.Summary = summary.Content
	}
	return c.JSON(e.Succ(view))
}

// DeleteCodeMemory 归档一条记忆。
//
// 归档而不是物理删除：被删掉的记忆下次抽取可能又被原样写回去，
// 留一条归档记录至少能看出「这条曾经存在过」。
func DeleteCodeMemory(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	entryID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || entryID == 0 {
		return c.JSON(e.Fail(errors.New("记忆参数无效")))
	}
	var entry model.AICodeMemoryEntry
	if err := global.DB.Where("id = ? AND user_id = ?", uint(entryID), claims.UserId).
		First(&entry).Error; err != nil {
		return c.JSON(e.Fail(errors.New("记忆不存在或无权操作")))
	}
	if err := archiveCodeMemoryEntry(global.DB, &entry, 0); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}

// ExtractCodeSessionMemory 手动触发一次抽取。
// 抽取平时挂在执行结束后，这个入口是给「这次聊出了重要结论，立刻记下来」用的。
func ExtractCodeSessionMemory(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return c.JSON(e.Fail(errors.New("会话 ID 无效")))
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	enqueueCodeMemoryExtraction(session.ID)
	return c.JSON(e.Succ(nil))
}
