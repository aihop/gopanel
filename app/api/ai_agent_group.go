package api

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func normalizeAIProjectWorkDir(workDir string, claims *token.CustomClaims) (string, error) {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" || !filepath.IsAbs(workDir) {
		return "", errors.New("项目路径必须是有效的绝对目录")
	}
	resolvedPath, err := filepath.EvalSymlinks(filepath.Clean(workDir))
	if err != nil {
		return "", errors.New("项目路径不存在或无法访问")
	}
	info, err := os.Stat(resolvedPath)
	if err != nil || !info.IsDir() {
		return "", errors.New("项目路径必须是可访问的目录")
	}
	if claims.Role == constant.UserRoleSubAdmin {
		if err := service.ValidatePathWithinBase(claims.FileBaseDir, resolvedPath); err != nil {
			return "", err
		}
	}
	return resolvedPath, nil
}

func GetAIGroups(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	groupRepo := repo.NewAIGroupRepo()
	groups, total, err := groupRepo.GetGroups(claims.UserId, claims.Role == constant.UserRoleSuper, page, limit)
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
		WorkDir     string `json:"workDir"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return c.JSON(e.Fail(errors.New("项目名称不能为空")))
	}
	workDir, err := normalizeAIProjectWorkDir(req.WorkDir, claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	group := &model.AIGroup{Name: name, Description: strings.TrimSpace(req.Description), WorkDir: workDir, CreatorID: claims.UserId}
	groupRepo := repo.NewAIGroupRepo()
	if err := groupRepo.CreateGroup(group); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(group))
}
