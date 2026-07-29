package api

import (
	"errors"
	"strconv"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func GetCodeSessionHistory(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "0"))
	if page < 1 {
		page = 1
	}
	if limit < 0 || limit > 100 {
		limit = 50
	}
	messages, err := repo.NewAITaskRepo().GetMessagesBySessionID(session.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	runs, total, err := repo.NewAIDevSessionRepo().GetExecutionRunsBySessionID(session.ID, page, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	for _, run := range runs {
		if run != nil {
			run.RawOutput = ""
		}
	}
	return c.JSON(e.Succ(fiber.Map{
		"session":  session,
		"messages": messages,
		"runs":     runs,
		"total":    total,
		"page":     page,
		"limit":    limit,
	}))
}

func GetCodeExecutionRun(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	runID, _ := strconv.Atoi(c.Params("id"))
	run, err := repo.NewAIDevSessionRepo().GetExecutionRunByID(uint(runID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if run.SessionID == 0 {
		return c.JSON(e.Fail(errors.New("该运行记录未关联长期会话")))
	}
	if _, err := getAISessionWithPermission(run.SessionID, claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(run))
}
