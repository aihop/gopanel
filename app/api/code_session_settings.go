package api

import (
	"strconv"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func UpdateCodeSessionApprovalPolicy(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var req struct {
		ApprovalPolicy string `json:"approvalPolicy"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	changed, err := updateCodeApprovalPolicy(session, req.ApprovalPolicy)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := validateCodeExecutorApprovalPolicy(session.AgentName, session.ApprovalPolicy); err != nil {
		return c.JSON(e.Fail(err))
	}
	if !changed {
		return c.JSON(e.Succ(session))
	}
	sessionRepo := repo.NewAIDevSessionRepo()
	if err := sessionRepo.UpdateSessionApprovalPolicy(session.ID, session.ApprovalPolicy); err != nil {
		return c.JSON(e.Fail(err))
	}
	createAITimelineEvent(sessionRepo, &model.AITimelineEvent{
		SessionID: session.ID,
		TaskID:    session.LastTaskID,
		EventType: "approval_policy_changed",
		Stage:     session.CurrentStage,
		Title:     "确认方式已调整",
		Content:   session.ApprovalPolicy,
		Status:    "info",
	})
	return c.JSON(e.Succ(session))
}
