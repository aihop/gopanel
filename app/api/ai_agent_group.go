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

func GetAIGroups(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	groupRepo := repo.NewAIGroupRepo()
	groups, total, err := groupRepo.GetGroups(page, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": groups, "total": total}))
}
func CreateAIGroup(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	group := &model.AIGroup{Name: req.Name, Description: req.Description, CreatorID: claims.UserId}
	groupRepo := repo.NewAIGroupRepo()
	if err := groupRepo.CreateGroup(group); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(group))
}
