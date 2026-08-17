package api

import (
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeDesktopSummary struct {
	Attention int64 `json:"attention"`
	Running   int64 `json:"running"`
	Queued    int64 `json:"queued"`
}

func loadCodeDesktopSummary(userID uint) (codeDesktopSummary, error) {
	var summary codeDesktopSummary
	err := global.DB.Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN bucket = 'attention' THEN 1 ELSE 0 END), 0) AS attention,
			COALESCE(SUM(CASE WHEN bucket = 'running' THEN 1 ELSE 0 END), 0) AS running,
			COALESCE(SUM(CASE WHEN bucket = 'queued' THEN 1 ELSE 0 END), 0) AS queued
		FROM (
			SELECT tasks.id, CASE
				WHEN tasks.status IN ('pending_approval', 'failed')
					OR delivery.status IN ('failed', 'partial', 'conflict') THEN 'attention'
				WHEN tasks.status IN ('running', 'delivering') OR delivery.status = 'running' THEN 'running'
				WHEN tasks.status = 'queued' OR delivery.status = 'queued' THEN 'queued'
				ELSE 'idle'
			END AS bucket
			FROM ai_tasks AS tasks
			LEFT JOIN ai_code_delivery_jobs AS delivery
				ON delivery.session_id = tasks.session_id
					AND delivery.user_id = tasks.user_id
					AND (delivery.task_id = 0 OR delivery.task_id = tasks.id)
			WHERE tasks.user_id = ?
				AND tasks.archived_at IS NULL
				AND COALESCE(tasks.agent_name, '') <> 'terminal'
		) AS task_states
	`, userID).Scan(&summary).Error
	return summary, err
}

func GetCodeDesktopSummary(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	summary, err := loadCodeDesktopSummary(claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(summary))
}
