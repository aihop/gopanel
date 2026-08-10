package api

import (
	"errors"
	"strconv"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
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
	// taskId 可选：指定后只返回该任务的对话，不指定则返回整个会话的（保持既有行为）。
	taskID, _ := strconv.Atoi(c.Query("taskId", "0"))
	taskRepo := repo.NewAITaskRepo()
	var messages []*model.AIMessage
	if taskID > 0 {
		messages, err = taskRepo.GetMessagesBySessionAndTaskID(session.ID, uint(taskID))
	} else {
		messages, err = taskRepo.GetMessagesBySessionID(session.ID)
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if session.AgentName == "codex" {
		if err := repairNativeCodexSessionBinding(session); err != nil {
			return c.JSON(e.Fail(err))
		}
		nativeMessages, nativeErr := getNativeCodexMessages(session)
		if nativeErr != nil {
			return c.JSON(e.Fail(nativeErr))
		}
		// 固化到库里：之后 rollout 文件被清理或 codex 再改格式，历史都不会丢。
		// 固化失败不影响本次展示，只记日志。
		if persistErr := persistNativeCodexMessages(session.ID, nativeMessages); persistErr != nil {
			global.LOG.Warnf("Persist native Codex history for session %d failed: %v", session.ID, persistErr)
		}
		// rollout 文件本身没有任务概念，整份记录归属会话当前任务。
		// 查看更早的任务时不该把它混进去。
		if taskID == 0 || uint(taskID) == session.LastTaskID {
			messages = mergeCodeHistoryMessages(messages, nativeMessages)
		}
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
