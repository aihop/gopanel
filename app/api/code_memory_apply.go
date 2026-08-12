package api

import (
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
		return applyCodeMemoryUserSummary(tx, response.UserSummary, context.UserID)
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
// 必须带 user_id 过滤：id 是模型给出来的，它有可能引用到别人的记忆
// （提示词里只会出现本人的条目，但模型会编号码）。少了这个条件，
// 一次幻觉就能改写别人的记忆库。
func loadCodeMemoryTarget(tx *gorm.DB, rawID string, context codeMemoryApplyContext) *model.AICodeMemoryEntry {
	id, err := strconv.ParseUint(strings.TrimSpace(rawID), 10, 64)
	if err != nil || id == 0 {
		return nil
	}
	var entry model.AICodeMemoryEntry
	if err := tx.Where("id = ? AND user_id = ?", uint(id), context.UserID).First(&entry).Error; err != nil {
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
func applyCodeMemoryUserSummary(tx *gorm.DB, summary string, userID uint) error {
	if strings.TrimSpace(summary) == "" || userID == 0 {
		return nil
	}
	var existing model.AICodeMemorySummary
	err := tx.Where("user_id = ?", userID).First(&existing).Error
	if err != nil {
		return tx.Create(&model.AICodeMemorySummary{UserID: userID, Content: summary}).Error
	}
	return tx.Model(&existing).Update("content", summary).Error
}
