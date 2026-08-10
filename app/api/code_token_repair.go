package api

import (
	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

const (
	codeTokenUsagePending     = "pending"
	codeTokenUsageRecorded    = "recorded"
	codeTokenUsageRecovered   = "recovered"
	codeTokenUsageUnavailable = "unavailable"
)

type CodeTokenRepairResult struct {
	Recorded    int64
	Recovered   int64
	Unavailable int64
}

func RepairLegacyCodeTokenUsage(db *gorm.DB) (CodeTokenRepairResult, error) {
	result := CodeTokenRepairResult{}
	lastID := uint(0)
	for {
		var runs []model.AIExecutionRun
		if err := db.Select(
			"id", "executor_id", "raw_output", "input_tokens", "output_tokens",
			"cached_input_tokens", "reasoning_tokens", "total_tokens",
		).Where("id > ? AND (token_usage_status = ? OR token_usage_status = '')", lastID, codeTokenUsagePending).
			Order("id").Limit(100).Find(&runs).Error; err != nil {
			return result, err
		}
		if len(runs) == 0 {
			return result, nil
		}
		for index := range runs {
			run := &runs[index]
			lastID = run.ID
			status := codeTokenUsageRecorded
			updates := map[string]any{"token_usage_status": status}
			if run.TotalTokens <= 0 && run.InputTokens <= 0 && run.OutputTokens <= 0 {
				parsed := parseCodeExecutorOutput(run.ExecutorID, []byte(run.RawOutput), "")
				if parsed.TokenUsageReported {
					status = codeTokenUsageRecovered
					updates = map[string]any{
						"input_tokens": parsed.InputTokens, "output_tokens": parsed.OutputTokens,
						"cached_input_tokens": parsed.CachedInputTokens, "reasoning_tokens": parsed.ReasoningTokens,
						"total_tokens": parsed.TotalTokens, "token_usage_status": status,
					}
				} else {
					status = codeTokenUsageUnavailable
					updates["token_usage_status"] = status
				}
			}
			if err := db.Model(&model.AIExecutionRun{}).Where("id = ? AND (token_usage_status = ? OR token_usage_status = '')", run.ID, codeTokenUsagePending).
				Updates(updates).Error; err != nil {
				return result, err
			}
			switch status {
			case codeTokenUsageRecorded:
				result.Recorded++
			case codeTokenUsageRecovered:
				result.Recovered++
			case codeTokenUsageUnavailable:
				result.Unavailable++
			}
		}
	}
}
