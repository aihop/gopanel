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

// === AI Group APIs ===

func GetAIGroups(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize", "50"))

	groupRepo := repo.NewAIGroupRepo()
	groups, total, err := groupRepo.GetGroups(page, pageSize)
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(fiber.Map{
		"items": groups,
		"total": total,
	}))
}

func CreateAIGroup(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}

	group := &model.AIGroup{
		Name:        req.Name,
		Description: req.Description,
		CreatorID:   claims.UserId,
	}

	groupRepo := repo.NewAIGroupRepo()
	if err := groupRepo.CreateGroup(group); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(group))
}

// === AI Task APIs ===

// 获取历史任务列表
func GetAITasks(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize", "20"))
	projectID, _ := strconv.Atoi(c.Query("projectId", "0"))

	aiRepo := repo.NewAITaskRepo()
	var tasks []*model.AITask
	var total int64
	var err error

	if projectID > 0 {
		tasks, total, err = aiRepo.GetTasksByProjectID(uint(projectID), page, pageSize)
	} else {
		tasks, total, err = aiRepo.GetTasksByUserID(claims.UserId, page, pageSize)
	}

	if err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(fiber.Map{
		"items": tasks,
		"total": total,
	}))
}

// 获取某个任务的历史对话记录
func GetAITaskMessages(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	taskID, _ := strconv.Atoi(c.Params("id"))

	aiRepo := repo.NewAITaskRepo()
	task, err := aiRepo.GetTaskByID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	// 权限校验
	if task.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(err))
	}

	messages, err := aiRepo.GetMessagesByTaskID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(messages))
}

// 重命名任务
func UpdateAITask(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	taskID, _ := strconv.Atoi(c.Params("id"))

	var req struct {
		Title string `json:"title"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
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

// 删除任务
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
