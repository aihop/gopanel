package api

import (
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"strconv"
)

func GetAITasks(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	projectID, _ := strconv.Atoi(c.Query("projectId", "0"))
	aiRepo := repo.NewAITaskRepo()
	var tasks []*model.AITask
	var total int64
	var err error
	if projectID > 0 {
		tasks, total, err = aiRepo.GetTasksByProjectID(uint(projectID), page, limit)
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
		return c.JSON(e.Fail(err))
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
		return c.JSON(e.Fail(err))
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
		return c.JSON(e.Fail(err))
	}
	if err := aiRepo.DeleteTask(uint(taskID)); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
