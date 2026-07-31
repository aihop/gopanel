package api

import (
	"errors"
	"strconv"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type codeTokenUsage struct {
	InputTokens       int64 `json:"inputTokens"`
	OutputTokens      int64 `json:"outputTokens"`
	CachedInputTokens int64 `json:"cachedInputTokens"`
	ReasoningTokens   int64 `json:"reasoningTokens"`
	TotalTokens       int64 `json:"totalTokens"`
	Runs              int64 `json:"runs"`
}

type codeDailyTokenUsage struct {
	Date              string `json:"date"`
	InputTokens       int64  `json:"inputTokens"`
	OutputTokens      int64  `json:"outputTokens"`
	CachedInputTokens int64  `json:"cachedInputTokens"`
	ReasoningTokens   int64  `json:"reasoningTokens"`
	TotalTokens       int64  `json:"totalTokens"`
	Runs              int64  `json:"runs"`
}

type codeTokenUsageResponse struct {
	Session codeTokenUsage        `json:"session"`
	Project codeTokenUsage        `json:"project"`
	Daily   []codeDailyTokenUsage `json:"daily"`
}

func sumCodeTokenUsage(query *gorm.DB) (codeTokenUsage, error) {
	var usage codeTokenUsage
	err := query.Select(`COALESCE(SUM(input_tokens), 0) AS input_tokens,
		COALESCE(SUM(output_tokens), 0) AS output_tokens,
		COALESCE(SUM(cached_input_tokens), 0) AS cached_input_tokens,
		COALESCE(SUM(reasoning_tokens), 0) AS reasoning_tokens,
		COALESCE(SUM(total_tokens), 0) AS total_tokens, COUNT(*) AS runs`).Scan(&usage).Error
	return usage, err
}

func loadCodeTokenUsage(session *model.AIDevSession) (codeTokenUsageResponse, error) {
	if session == nil || session.ID == 0 {
		return codeTokenUsageResponse{}, errors.New("开发会话无效")
	}
	sessionUsage, err := sumCodeTokenUsage(global.DB.Model(&model.AIExecutionRun{}).Where("session_id = ?", session.ID))
	if err != nil {
		return codeTokenUsageResponse{}, err
	}
	projectQuery := global.DB.Model(&model.AIExecutionRun{}).Where("session_id = ?", session.ID)
	if session.ProjectID > 0 {
		projectQuery = global.DB.Model(&model.AIExecutionRun{}).
			Joins("JOIN ai_dev_sessions ON ai_dev_sessions.id = ai_execution_runs.session_id").
			Where("ai_dev_sessions.project_id = ?", session.ProjectID)
	}
	projectUsage, err := sumCodeTokenUsage(projectQuery)
	if err != nil {
		return codeTokenUsageResponse{}, err
	}
	var runs []model.AIExecutionRun
	dailyQuery := global.DB.Model(&model.AIExecutionRun{}).Where("session_id = ? AND created_at >= ?", session.ID, time.Now().AddDate(0, 0, -29))
	if session.ProjectID > 0 {
		dailyQuery = global.DB.Model(&model.AIExecutionRun{}).
			Joins("JOIN ai_dev_sessions ON ai_dev_sessions.id = ai_execution_runs.session_id").
			Where("ai_dev_sessions.project_id = ? AND ai_execution_runs.created_at >= ?", session.ProjectID, time.Now().AddDate(0, 0, -29))
	}
	if err := dailyQuery.Order("ai_execution_runs.created_at asc").Find(&runs).Error; err != nil {
		return codeTokenUsageResponse{}, err
	}
	dailyByDate := make(map[string]*codeDailyTokenUsage)
	order := make([]string, 0, 30)
	for _, run := range runs {
		date := run.CreatedAt.Local().Format("2006-01-02")
		day := dailyByDate[date]
		if day == nil {
			day = &codeDailyTokenUsage{Date: date}
			dailyByDate[date] = day
			order = append(order, date)
		}
		day.InputTokens += run.InputTokens
		day.OutputTokens += run.OutputTokens
		day.CachedInputTokens += run.CachedInputTokens
		day.ReasoningTokens += run.ReasoningTokens
		day.TotalTokens += run.TotalTokens
		day.Runs++
	}
	daily := make([]codeDailyTokenUsage, 0, len(order))
	for _, date := range order {
		daily = append(daily, *dailyByDate[date])
	}
	return codeTokenUsageResponse{Session: sessionUsage, Project: projectUsage, Daily: daily}, nil
}

func GetCodeTokenUsage(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return c.JSON(e.Fail(errors.New("会话 ID 无效")))
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	usage, err := loadCodeTokenUsage(session)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(usage))
}
