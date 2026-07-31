package api

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func normalizeCodeSessionTitle(value string) (string, error) {
	title := strings.TrimSpace(value)
	if title == "" {
		return "", errors.New("会话名称不能为空")
	}
	if utf8.RuneCountInString(title) > 255 {
		return "", errors.New("会话名称不能超过 255 个字符")
	}
	return title, nil
}

func UpdateCodeSessionTitle(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var req struct {
		Title string `json:"title"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	title, err := normalizeCodeSessionTitle(req.Title)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if title == session.Title {
		return c.JSON(e.Succ(session))
	}
	if err := repo.NewAIDevSessionRepo().UpdateSessionTitle(session.ID, title); err != nil {
		return c.JSON(e.Fail(err))
	}
	session.Title = title
	createAITimelineEvent(repo.NewAIDevSessionRepo(), &model.AITimelineEvent{
		SessionID: session.ID,
		TaskID:    session.LastTaskID,
		EventType: "session_title_changed",
		Stage:     session.CurrentStage,
		Title:     "会话名称已修改",
		Content:   title,
		Status:    "info",
	})
	return c.JSON(e.Succ(session))
}

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
