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

func canManageAIProject(project *model.AIGroup, claims *token.CustomClaims) bool {
	return project != nil && claims != nil && (project.CreatorID == claims.UserId || claims.Role == constant.UserRoleSuper)
}

func existingAIProjectDirectory(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "", errors.New("默认项目目录无效")
	}
	resolvedPath, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolvedPath)
	if err != nil || !info.IsDir() {
		return "", errors.New("默认项目目录不可访问")
	}
	return resolvedPath, nil
}

func aiProjectDirectoryDefaults(claims *token.CustomClaims, userHome string) (string, string, error) {
	if claims != nil && claims.Role == constant.UserRoleSubAdmin {
		baseDir, err := existingAIProjectDirectory(claims.FileBaseDir)
		if err != nil {
			return "", "", errors.New("当前账号未配置有效的工作目录")
		}
		return baseDir, baseDir, nil
	}
	rootDir := string(filepath.Separator)
	if volume := filepath.VolumeName(userHome); volume != "" {
		rootDir = volume + string(filepath.Separator)
	}
	resolvedRoot, err := existingAIProjectDirectory(rootDir)
	if err != nil {
		return "", "", err
	}
	defaultDir, err := existingAIProjectDirectory(userHome)
	if err != nil {
		defaultDir = resolvedRoot
	}
	return defaultDir, resolvedRoot, nil
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
	userHome, _ := os.UserHomeDir()
	defaultWorkDir, directoryRoot, err := aiProjectDirectoryDefaults(claims, userHome)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"items":          groups,
		"total":          total,
		"defaultWorkDir": defaultWorkDir,
		"directoryRoot":  directoryRoot,
	}))
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

func UpdateAIGroup(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目 ID 无效")))
	}
	groupRepo := repo.NewAIGroupRepo()
	project, err := groupRepo.GetGroupByID(uint(projectID))
	if err != nil {
		return c.JSON(e.Fail(errors.New("项目不存在")))
	}
	if !canManageAIProject(project, claims) {
		return c.JSON(e.Fail(errors.New("无权修改该项目")))
	}
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
	project.Name = name
	project.Description = strings.TrimSpace(req.Description)
	project.WorkDir = workDir
	if err := groupRepo.UpdateGroup(project); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(project))
}
