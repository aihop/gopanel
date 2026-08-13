package api

import (
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

const (
	codeMemoryExtractionIdle    = "idle"
	codeMemoryExtractionQueued  = "queued"
	codeMemoryExtractionRunning = "running"
	codeMemoryExtractionSuccess = "success"
	codeMemoryExtractionSkipped = "skipped"
	codeMemoryExtractionFailed  = "failed"

	codeMemoryTriggerAutomatic = "automatic"
	codeMemoryTriggerManual    = "manual"

	codeMemoryExtractionReasonFailed = "extraction_failed"
)

type codeMemorySkipError struct {
	reason string
}

func (err codeMemorySkipError) Error() string { return err.reason }

func loadCodeMemoryExtractionStatus(sessionID uint) (*model.AICodeMemoryExtractionState, error) {
	if global.DB == nil || sessionID == 0 {
		return nil, nil
	}
	var state model.AICodeMemoryExtractionState
	if err := global.DB.Where("session_id = ?", sessionID).First(&state).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

func updateCodeMemoryExtractionStatus(sessionID uint, updates map[string]any) error {
	if global.DB == nil || sessionID == 0 {
		return nil
	}
	state := model.AICodeMemoryExtractionState{SessionID: sessionID}
	return global.DB.Where("session_id = ?", sessionID).Assign(updates).FirstOrCreate(&state).Error
}

func queueCodeMemoryExtractionStatus(sessionID uint, trigger string) error {
	return updateCodeMemoryExtractionStatus(sessionID, map[string]any{
		"status": codeMemoryExtractionQueued, "trigger": trigger, "reason": "",
		"added": 0, "merged": 0, "replaced": 0, "archived": 0,
		"started_at": nil, "completed_at": nil,
	})
}

func startCodeMemoryExtractionStatus(sessionID uint, trigger string) error {
	now := time.Now()
	return updateCodeMemoryExtractionStatus(sessionID, map[string]any{
		"status": codeMemoryExtractionRunning, "trigger": trigger, "reason": "",
		"started_at": now, "completed_at": nil,
	})
}

func finishCodeMemoryExtractionStatus(sessionID uint, status, reason string, result codeMemoryApplyResult, lastMessageID uint) error {
	now := time.Now()
	updates := map[string]any{
		"status": status, "reason": truncateCodeMemoryStatusReason(reason),
		"added": result.Added, "merged": result.Merged,
		"replaced": result.Replaced, "archived": result.Archived,
		"completed_at": now,
	}
	if status == codeMemoryExtractionSuccess && lastMessageID > 0 {
		updates["last_message_id"] = lastMessageID
	}
	return updateCodeMemoryExtractionStatus(sessionID, updates)
}

func truncateCodeMemoryStatusReason(reason string) string {
	runes := []rune(strings.TrimSpace(reason))
	if len(runes) > 500 {
		runes = runes[:500]
	}
	return string(runes)
}
