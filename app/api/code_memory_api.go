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
	"gorm.io/gorm"
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
	if projectID > 0 {
		if _, err := getCodeProjectWithPermission(uint(projectID), claims); err != nil {
			return c.JSON(e.Fail(err))
		}
	}

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
	// 按注入顺序返回，让界面无需自己排序就能与 AI 读到的保持一致。
	view := codeMemoryListView{Entries: flattenCodeMemoryInInjectionOrder(entries), Total: total}
	var summary model.AICodeMemorySummary
	if err := global.DB.Where("user_id = ?", claims.UserId).First(&summary).Error; err == nil {
		view.Summary = summary.Content
	}
	return c.JSON(e.Succ(view))
}

// CreateCodeMemory 手动添加一条记忆。
//
// 刻意不让用户填 kind / module / tier：人手写的记忆几乎总是「以后就这么办」，
// 也就是 decision。把这三个字段做成表单，界面立刻就不简约了，而用户在
// 这三项上的选择对结果几乎没有影响。唯一值得暴露的是作用范围——
// 它决定这条规矩只管当前项目还是所有项目。
func CreateCodeMemory(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		Content     string `json:"content"`
		ProjectID   uint   `json:"projectId"`
		AllProjects bool   `json:"allProjects"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	// 手写的内容同样要脱敏：用户很可能直接粘一段配置进来。
	content := truncateCodeMemoryText(scrubCodeMemoryText(req.Content), codeMemoryContentMaxRunes)
	if content == "" {
		return c.JSON(e.Fail(errors.New("记忆内容不能为空")))
	}
	scope := codeMemoryScopeProject
	if req.AllProjects {
		scope = codeMemoryScopeUser
	}
	if scope == codeMemoryScopeProject && req.ProjectID == 0 {
		return c.JSON(e.Fail(errors.New("请先选择项目，或改为对所有项目生效")))
	}
	if scope == codeMemoryScopeProject {
		if _, err := getCodeProjectWithPermission(req.ProjectID, claims); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	entry := model.AICodeMemoryEntry{
		UserID: claims.UserId, ProjectID: codeMemoryProjectIDForScope(scope, req.ProjectID),
		Scope: scope, Kind: codeMemoryKindDecision, Tier: codeMemoryTierForKind(codeMemoryKindDecision),
		ModuleKey: normalizeCodeMemoryModuleKey("general", scope),
		Content:   content, Status: codeMemoryStatusActive,
	}
	if err := global.DB.Create(&entry).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(entry))
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
	queued := enqueueCodeMemoryExtraction(session.ID, codeMemoryTriggerManual, true)
	state, statusErr := loadCodeMemoryExtractionStatus(session.ID)
	if statusErr != nil {
		return c.JSON(e.Fail(statusErr))
	}
	return c.JSON(e.Succ(fiber.Map{"queued": queued, "status": state}))
}

func GetCodeSessionMemoryStatus(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return c.JSON(e.Fail(errors.New("会话 ID 无效")))
	}
	if _, err := getAISessionWithPermission(uint(sessionID), claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	state, err := loadCodeMemoryExtractionStatus(uint(sessionID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if state == nil {
		state = &model.AICodeMemoryExtractionState{SessionID: uint(sessionID), Status: codeMemoryExtractionIdle}
	}
	return c.JSON(e.Succ(state))
}

func SaveCodeMemorySummary(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		Content string `json:"content"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	content := truncateCodeMemoryText(scrubCodeMemoryText(req.Content), codeMemorySummaryMaxRunes)
	if content == "" {
		return c.JSON(e.Fail(errors.New("用户画像不能为空")))
	}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var summary model.AICodeMemorySummary
		lookupErr := tx.Where("user_id = ?", claims.UserId).First(&summary).Error
		before := ""
		if lookupErr == nil {
			before = summary.Content
			if before == content {
				return nil
			}
			if err := tx.Model(&summary).Update("content", content).Error; err != nil {
				return err
			}
		} else if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			if err := tx.Create(&model.AICodeMemorySummary{UserID: claims.UserId, Content: content}).Error; err != nil {
				return err
			}
		} else {
			return lookupErr
		}
		return createCodeMemorySummaryAudit(tx, claims.UserId, 0, "summary_update", "manual", before, content, c.IP())
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"content": content}))
}

func DeleteCodeMemorySummary(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var summary model.AICodeMemorySummary
		if err := tx.Where("user_id = ?", claims.UserId).First(&summary).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if err := tx.Delete(&summary).Error; err != nil {
			return err
		}
		return createCodeMemorySummaryAudit(tx, claims.UserId, 0, "summary_clear", "manual", summary.Content, "", c.IP())
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}

func GetCodeMemoryAuditEvents(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var events []model.AICodeMemoryAuditEvent
	if err := global.DB.Where("user_id = ?", claims.UserId).Order("created_at desc").Limit(50).Find(&events).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(events))
}
