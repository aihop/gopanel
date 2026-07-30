package api

import (
	"context"
	"errors"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"strconv"
)

func GetAITasks(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	page, limit = normalizeCodePage(page, limit, 20)
	projectID, _ := strconv.Atoi(c.Query("projectId", "0"))
	aiRepo := repo.NewAITaskRepo()
	var tasks []*model.AITask
	var total int64
	var err error
	if projectID > 0 {
		tasks, total, err = aiRepo.GetTasksByProjectAndUserID(uint(projectID), claims.UserId, page, limit)
	} else {
		tasks, total, err = aiRepo.GetTasksByUserID(claims.UserId, page, limit)
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": tasks, "total": total}))
}
func GetAITaskMessages(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	taskID, _ := strconv.Atoi(c.Params("id"))
	aiRepo := repo.NewAITaskRepo()
	task, err := aiRepo.GetTaskByID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if task.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("无权访问该 AI 任务")))
	}
	messages, err := aiRepo.GetMessagesByTaskID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(messages))
}
func UpdateAITask(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	taskID, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Title string `json:"title"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	aiRepo := repo.NewAITaskRepo()
	task, err := aiRepo.GetTaskByID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if task.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("无权修改该 AI 任务")))
	}
	task.Title = req.Title
	if err := aiRepo.UpdateTask(task); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
func DeleteAITask(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	taskID, _ := strconv.Atoi(c.Params("id"))
	aiRepo := repo.NewAITaskRepo()
	task, err := aiRepo.GetTaskByID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if task.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("无权删除该 AI 任务")))
	}
	if task.SessionID > 0 {
		sessionRepo := repo.NewAIDevSessionRepo()
		session, sessionErr := sessionRepo.GetSessionByID(task.SessionID)
		if sessionErr != nil {
			return c.JSON(e.Fail(sessionErr))
		}
		if _, err := sessionRepo.CancelQueuedInstructions(task.SessionID); err != nil {
			return c.JSON(e.Fail(err))
		}
		backgroundCodeRunner.cancel(task.SessionID)
		stopContext, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelStop()
		codeExecutions.cancelAndWait(stopContext, codeExecutionWorkspaceKeys(session))
		if !backgroundCodeRunner.wait(stopContext, task.SessionID) || stopContext.Err() != nil {
			return c.JSON(e.Fail(errors.New("停止 Code 会话超时，请稍后重试")))
		}
		if err := aiRepo.DeleteTaskAndSession(uint(taskID), session.ID); err != nil {
			return c.JSON(e.Fail(err))
		}
		if err := cleanupCodeSessionWorktree(session); err != nil {
			global.LOG.Warnf("Cleanup Code worktree %d skipped: %v", session.ID, err)
		}
		return c.JSON(e.Succ())
	}
	if err := aiRepo.DeleteTask(uint(taskID)); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
