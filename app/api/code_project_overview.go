package api

import (
	"errors"
	"strconv"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type codeProjectLatestRun struct {
	ID          uint       `json:"id"`
	SessionID   uint       `json:"sessionId"`
	TaskID      uint       `json:"taskId"`
	ExecutorID  string     `json:"executorId"`
	Model       string     `json:"model"`
	Status      string     `json:"status"`
	DurationMS  int64      `json:"durationMs"`
	TotalTokens int64      `json:"totalTokens"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type codeProjectOverview struct {
	ProjectID        uint                             `json:"projectId"`
	TaskCount        int64                            `json:"taskCount"`
	ExecutionSummary *model.AIProjectExecutionSummary `json:"executionSummary"`
	TokenUsage       codeTokenUsage                   `json:"tokenUsage"`
	Budget           codeTokenBudget                  `json:"budget"`
	LatestRun        *codeProjectLatestRun            `json:"latestRun"`
}

func loadCodeProjectOverview(project *model.AIGroup, userID uint, includeAll bool) (codeProjectOverview, error) {
	if project == nil || project.ID == 0 {
		return codeProjectOverview{}, errors.New("项目无效")
	}
	if err := repo.NewAIGroupRepo().LoadExecutionSummaries([]*model.AIGroup{project}, userID, includeAll); err != nil {
		return codeProjectOverview{}, err
	}
	projectRuns := global.DB.Model(&model.AIExecutionRun{}).
		Joins("JOIN ai_dev_sessions ON ai_dev_sessions.id = ai_execution_runs.session_id").
		Where("ai_dev_sessions.project_id = ?", project.ID)
	if !includeAll {
		projectRuns = projectRuns.Where("ai_dev_sessions.user_id = ?", userID)
	}
	usage, err := sumCodeTokenUsage(projectRuns)
	if err != nil {
		return codeProjectOverview{}, err
	}
	budget, err := loadCodeTokenBudget(project.ID, time.Now())
	if err != nil {
		return codeProjectOverview{}, err
	}
	var run model.AIExecutionRun
	latestRun := (*codeProjectLatestRun)(nil)
	latestRunQuery := global.DB.Model(&model.AIExecutionRun{}).
		Joins("JOIN ai_dev_sessions ON ai_dev_sessions.id = ai_execution_runs.session_id").
		Where("ai_dev_sessions.project_id = ?", project.ID)
	if !includeAll {
		latestRunQuery = latestRunQuery.Where("ai_dev_sessions.user_id = ?", userID)
	}
	err = latestRunQuery.Order("ai_execution_runs.created_at desc").First(&run).Error
	if err == nil {
		latestRun = &codeProjectLatestRun{
			ID: run.ID, SessionID: run.SessionID, TaskID: run.TaskID, ExecutorID: run.ExecutorID,
			Model: run.Model, Status: run.Status, DurationMS: run.DurationMS, TotalTokens: run.TotalTokens,
			CreatedAt: run.CreatedAt, CompletedAt: run.CompletedAt,
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return codeProjectOverview{}, err
	}
	return codeProjectOverview{
		ProjectID: project.ID, TaskCount: project.TaskCount, ExecutionSummary: project.ExecutionSummary,
		TokenUsage: usage, Budget: budget, LatestRun: latestRun,
	}, nil
}

func GetCodeProjectOverview(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	project, err := getCodeProjectWithPermission(uint(projectID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	overview, err := loadCodeProjectOverview(project, claims.UserId, claims.Role == constant.UserRoleSuper)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(overview))
}
