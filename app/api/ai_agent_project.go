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
		if err := validateAIProjectWorkDirForClaims(resolvedPath, claims); err != nil {
			return "", err
		}
	}
	return resolvedPath, nil
}

func canManageAIProject(project *model.AIProject, claims *token.CustomClaims) bool {
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

func GetAIProjects(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	page, limit = normalizeCodePage(page, limit, 50)
	projectRepo := repo.NewAIProjectRepo()
	projects, total, err := projectRepo.GetProjects(claims.UserId, claims.Role == constant.UserRoleSuper, page, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := projectRepo.LoadExecutionSummaries(projects, claims.UserId, claims.Role == constant.UserRoleSuper); err != nil {
		return c.JSON(e.Fail(err))
	}
	userHome, _ := os.UserHomeDir()
	defaultWorkDir, directoryRoot, err := aiProjectDirectoryDefaults(claims, userHome)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"items":          projects,
		"total":          total,
		"defaultWorkDir": defaultWorkDir,
		"directoryRoot":  directoryRoot,
	}))
}
func CreateAIProject(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		WorkDir            string   `json:"workDir"`
		SourceDirs         []string `json:"sourceDirs"`
		PrimaryRepository  *string  `json:"primaryRepository"`
		DeliveryBranch     *string  `json:"deliveryBranch"`
		RequireQualityGate bool     `json:"requireQualityGate"`
		MonthlyTokenBudget int64    `json:"monthlyTokenBudget"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return c.JSON(e.Fail(errors.New("项目名称不能为空")))
	}
	requestedDirs := req.SourceDirs
	if len(requestedDirs) == 0 && strings.TrimSpace(req.WorkDir) != "" {
		requestedDirs = []string{req.WorkDir}
	}
	sourceDirs, err := normalizeAIProjectSourceDirs(requestedDirs, claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.MonthlyTokenBudget < 0 {
		return c.JSON(e.Fail(errors.New("Token 月度预算不能为负数")))
	}
	primaryRepository, deliveryBranch := "", ""
	if req.PrimaryRepository != nil {
		primaryRepository = strings.TrimSpace(*req.PrimaryRepository)
	}
	if req.DeliveryBranch != nil {
		deliveryBranch = strings.TrimSpace(*req.DeliveryBranch)
	}
	project := &model.AIProject{
		Name: name, Description: strings.TrimSpace(req.Description), SourceDirs: sourceDirs,
		CreatorID: claims.UserId, PrimaryRepository: primaryRepository,
		DeliveryBranch: deliveryBranch, RequireQualityGate: true,
		MonthlyTokenBudget: req.MonthlyTokenBudget,
	}
	if err := applyCodeProjectDeliveryPolicy(project, sourceDirs); err != nil {
		return c.JSON(e.Fail(err))
	}
	projectRepo := repo.NewAIProjectRepo()
	if err := projectRepo.CreateProject(project); err != nil {
		return c.JSON(e.Fail(err))
	}
	workDir, err := syncAIProjectWorkspace(project, sourceDirs)
	if err != nil {
		_ = projectRepo.DeleteProject(project.ID)
		_ = os.RemoveAll(aiProjectWorkspaceDir(project.CreatorID, project.ID))
		return c.JSON(e.Fail(err))
	}
	project.WorkDir = workDir
	if err := projectRepo.UpdateProject(project); err != nil {
		_ = projectRepo.DeleteProject(project.ID)
		_ = os.RemoveAll(aiProjectWorkspaceDir(project.CreatorID, project.ID))
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(project))
}

func UpdateAIProject(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目 ID 无效")))
	}
	projectRepo := repo.NewAIProjectRepo()
	project, err := projectRepo.GetProjectByID(uint(projectID))
	if err != nil {
		return c.JSON(e.Fail(errors.New("项目不存在")))
	}
	if !canManageAIProject(project, claims) {
		return c.JSON(e.Fail(errors.New("无权修改该项目")))
	}
	var req struct {
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		WorkDir            string   `json:"workDir"`
		SourceDirs         []string `json:"sourceDirs"`
		PrimaryRepository  *string  `json:"primaryRepository"`
		DeliveryBranch     *string  `json:"deliveryBranch"`
		RequireQualityGate bool     `json:"requireQualityGate"`
		MonthlyTokenBudget int64    `json:"monthlyTokenBudget"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return c.JSON(e.Fail(errors.New("项目名称不能为空")))
	}
	requestedDirs := req.SourceDirs
	if len(requestedDirs) == 0 && strings.TrimSpace(req.WorkDir) != "" {
		requestedDirs = []string{req.WorkDir}
	}
	sourceDirs, err := normalizeAIProjectSourceDirs(requestedDirs, claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	workDir, err := syncAIProjectWorkspace(project, sourceDirs)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	project.Name = name
	project.Description = strings.TrimSpace(req.Description)
	project.WorkDir = workDir
	project.SourceDirs = sourceDirs
	if req.PrimaryRepository != nil {
		project.PrimaryRepository = strings.TrimSpace(*req.PrimaryRepository)
	}
	if req.DeliveryBranch != nil {
		project.DeliveryBranch = strings.TrimSpace(*req.DeliveryBranch)
	}
	if req.MonthlyTokenBudget < 0 {
		return c.JSON(e.Fail(errors.New("Token 月度预算不能为负数")))
	}
	project.RequireQualityGate = true
	project.MonthlyTokenBudget = req.MonthlyTokenBudget
	if err := applyCodeProjectDeliveryPolicy(project, sourceDirs); err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := projectRepo.UpdateProject(project); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(project))
}
