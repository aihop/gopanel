package api

import (
	"errors"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type codeTokenBudget struct {
	LimitTokens     int64   `json:"limitTokens"`
	UsedTokens      int64   `json:"usedTokens"`
	RemainingTokens int64   `json:"remainingTokens"`
	UsagePercent    float64 `json:"usagePercent"`
	Exceeded        bool    `json:"exceeded"`
	Unlimited       bool    `json:"unlimited"`
	Complete        bool    `json:"complete"`
	UnavailableRuns int64   `json:"unavailableRuns"`
	PendingRuns     int64   `json:"pendingRuns"`
}

func codeMonthStart(now time.Time) time.Time {
	local := now.Local()
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, local.Location())
}

func loadCodeTokenBudget(projectID uint, now time.Time) (codeTokenBudget, error) {
	if projectID == 0 {
		return codeTokenBudget{Unlimited: true, Complete: true}, nil
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(projectID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return codeTokenBudget{Unlimited: true, Complete: true}, nil
	}
	if err != nil {
		return codeTokenBudget{}, err
	}
	usage, err := sumCodeTokenUsage(global.DB.Model(&model.AIExecutionRun{}).
		Joins("JOIN ai_dev_sessions ON ai_dev_sessions.id = ai_execution_runs.session_id").
		Where("ai_dev_sessions.project_id = ? AND ai_execution_runs.created_at >= ?", projectID, codeMonthStart(now)))
	if err != nil {
		return codeTokenBudget{}, err
	}
	used := usage.TotalTokens
	budget := codeTokenBudget{
		LimitTokens: project.MonthlyTokenBudget, UsedTokens: used, Unlimited: project.MonthlyTokenBudget <= 0,
		Complete: usage.Complete, UnavailableRuns: usage.UnavailableRuns, PendingRuns: usage.PendingRuns,
	}
	if budget.Unlimited {
		return budget, nil
	}
	budget.RemainingTokens = max(project.MonthlyTokenBudget-used, 0)
	budget.UsagePercent = float64(used) / float64(project.MonthlyTokenBudget) * 100
	budget.Exceeded = used >= project.MonthlyTokenBudget
	return budget, nil
}

func validateCodeTokenBudget(session *model.AIDevSession) error {
	if session == nil || session.ProjectID == 0 {
		return nil
	}
	budget, err := loadCodeTokenBudget(session.ProjectID, time.Now())
	if err != nil {
		return err
	}
	if budget.Exceeded {
		return errors.New("项目本月 Token 预算已用尽，请调整预算后继续执行")
	}
	return nil
}
