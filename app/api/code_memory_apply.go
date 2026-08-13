package api

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type codeMemoryApplyResult struct {
	Added    int `json:"added"`
	Merged   int `json:"merged"`
	Replaced int `json:"replaced"`
	Archived int `json:"archived"`
}

type codeMemoryApplyContext struct {
	UserID    uint
	ProjectID uint
	SessionID uint
}

// applyCodeMemoryExtraction 把一次抽取的结果落库。
//
// 整体放进一个事务：合并和归档是成对的动作，只落一半会留下既没被合并、
// 也没被归档的孤儿条目，下次抽取又会把它当成新的重复项处理。
func applyCodeMemoryExtraction(
	response codeMemoryExtractionResponse,
	context codeMemoryApplyContext,
) (codeMemoryApplyResult, error) {
	result := codeMemoryApplyResult{}
	if global.DB == nil {
		return result, nil
	}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range response.WorkingAdd {
			itemResult, err := applyCodeMemoryItem(tx, item, context)
			if err != nil {
				return err
			}
			result.Added += itemResult.Added
			result.Merged += itemResult.Merged
			result.Replaced += itemResult.Replaced
			result.Archived += itemResult.Archived
		}
		archived, err := archiveCodeMemoryEntries(tx, response.WorkingArchive, context, 0)
		if err != nil {
			return err
		}
		result.Archived += archived
		return applyCodeMemoryUserSummary(tx, response.UserSummary, context)
	})
	return result, err
}

func applyCodeMemoryItem(
	tx *gorm.DB,
	item codeMemoryExtractionItem,
	context codeMemoryApplyContext,
) (codeMemoryApplyResult, error) {
	result := codeMemoryApplyResult{}
	tier := codeMemoryTierForKind(item.Kind)

	// merge_with 指向一条已有记忆：更新它而不是新增，这是判重的主路径。
	if target := loadCodeMemoryTarget(tx, item.MergeWith, context); target != nil {
		target.Content = item.Content
		target.Rationale = item.Rationale
		target.Kind = item.Kind
		target.ModuleKey = item.ModuleKey
		target.Tier = preferredCodeMemoryTier(target.Tier, tier)
		target.Status = codeMemoryStatusActive
		target.SourceSessionID = context.SessionID
		if err := tx.Save(target).Error; err != nil {
			return result, err
		}
		result.Merged++
		archived, err := archiveCodeMemoryEntries(tx, item.Archive, context, target.ID)
		if err != nil {
			return result, err
		}
		result.Archived += archived
		return result, nil
	}

	entry := model.AICodeMemoryEntry{
		UserID: context.UserID, ProjectID: codeMemoryProjectIDForScope(item.Scope, context.ProjectID),
		Scope: item.Scope, Kind: item.Kind, Tier: tier, ModuleKey: item.ModuleKey,
		Content: item.Content, Rationale: item.Rationale,
		Status: codeMemoryStatusActive, SourceSessionID: context.SessionID,
	}
	if err := tx.Create(&entry).Error; err != nil {
		return result, err
	}
	result.Added++

	// replace 是「新的取代旧的」：旧条目归档并指回新条目，保留链路而不是删掉。
	if target := loadCodeMemoryTarget(tx, item.Replace, context); target != nil {
		if err := archiveCodeMemoryEntry(tx, target, entry.ID); err != nil {
			return result, err
		}
		result.Replaced++
	}
	archived, err := archiveCodeMemoryEntries(tx, item.Archive, context, entry.ID)
	if err != nil {
		return result, err
	}
	result.Archived += archived
	return result, nil
}

// codeMemoryProjectIDForScope：user 作用域跨项目生效，不绑定项目。
func codeMemoryProjectIDForScope(scope string, projectID uint) uint {
	if scope == codeMemoryScopeUser {
		return 0
	}
	return projectID
}

// loadCodeMemoryTarget 按 id 取一条属于当前用户的记忆。
//
// 必须同时限制用户和当前作用域：id 是模型给出来的，它有可能引用到别人的
// 记忆，也可能编到同一用户另一个项目。抽取提示词只包含用户级记忆和当前
// 项目记忆，因此目标也只能落在这两个范围内。
func loadCodeMemoryTarget(tx *gorm.DB, rawID string, context codeMemoryApplyContext) *model.AICodeMemoryEntry {
	id, err := strconv.ParseUint(strings.TrimSpace(rawID), 10, 64)
	if err != nil || id == 0 {
		return nil
	}
	var entry model.AICodeMemoryEntry
	if err := tx.Where("id = ? AND user_id = ?", uint(id), context.UserID).
		Where("scope = ? OR (scope = ? AND project_id = ?)",
			codeMemoryScopeUser, codeMemoryScopeProject, context.ProjectID).
		First(&entry).Error; err != nil {
		return nil
	}
	return &entry
}

func archiveCodeMemoryEntries(
	tx *gorm.DB,
	rawIDs []string,
	context codeMemoryApplyContext,
	supersededBy uint,
) (int, error) {
	archived := 0
	for _, rawID := range rawIDs {
		target := loadCodeMemoryTarget(tx, rawID, context)
		if target == nil || target.Status == codeMemoryStatusArchived {
			continue
		}
		if target.ID == supersededBy {
			continue
		}
		if err := archiveCodeMemoryEntry(tx, target, supersededBy); err != nil {
			return archived, err
		}
		archived++
	}
	return archived, nil
}

func archiveCodeMemoryEntry(tx *gorm.DB, entry *model.AICodeMemoryEntry, supersededBy uint) error {
	now := time.Now()
	return tx.Model(entry).Updates(map[string]any{
		"status": codeMemoryStatusArchived, "tier": codeMemoryTierArchive,
		"archived_at": now, "superseded_by": supersededBy,
	}).Error
}

// applyCodeMemoryUserSummary 整体重写用户画像。
// 空字符串表示「保持不变」——这是提示词里约定的语义，
// 当成"清空"会让一次无内容的抽取抹掉长期积累的画像。
func applyCodeMemoryUserSummary(tx *gorm.DB, summary string, context codeMemoryApplyContext) error {
	summary = strings.TrimSpace(summary)
	if summary == "" || context.UserID == 0 {
		return nil
	}
	var existing model.AICodeMemorySummary
	err := tx.Where("user_id = ?", context.UserID).First(&existing).Error
	before := ""
	if err == nil {
		before = existing.Content
		if before == summary {
			return nil
		}
		if err := tx.Model(&existing).Update("content", summary).Error; err != nil {
			return err
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&model.AICodeMemorySummary{UserID: context.UserID, Content: summary}).Error; err != nil {
			return err
		}
	} else {
		return err
	}
	return createCodeMemorySummaryAudit(tx, context.UserID, context.SessionID, "summary_update", "extraction", before, summary, "")
}

func createCodeMemorySummaryAudit(tx *gorm.DB, userID, sessionID uint, action, source, before, after, ip string) error {
	return tx.Create(&model.AICodeMemoryAuditEvent{
		UserID: userID, SessionID: sessionID, Action: action, Source: source,
		Before: before, After: after, IP: ip,
	}).Error
}
