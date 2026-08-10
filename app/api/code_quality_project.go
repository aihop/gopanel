package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const codeProjectQualityCheckLimit = 20

type codeQualityPreflightItem struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Label     string `json:"label"`
	Command   string `json:"command"`
	WorkDir   string `json:"workDir"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

func PreflightCodeProjectQualityChecks(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var request struct {
		SourceDirs    []string                      `json:"sourceDirs"`
		QualityChecks []model.AIProjectQualityCheck `json:"qualityChecks"`
	}
	if err := c.Bind().JSON(&request); err != nil {
		return c.JSON(e.Fail(err))
	}
	sourceDirs, err := normalizeAIProjectSourceDirs(request.SourceDirs, claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	repositories, err := codeProjectQualityRepositories(sourceDirs)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	roots := make([]codeDeliveryQualityRoot, 0, len(repositories))
	for _, resolved := range repositories {
		roots = append(roots, codeDeliveryQualityRoot{WorkDir: resolved, IdentityDir: resolved, RuntimeDir: resolved})
	}
	checks, err := normalizeCodeProjectQualityChecks(request.QualityChecks, sourceDirs)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	detected := detectCodeQualityChecksInRoots(sourceDirs)
	detected = append(detected, configuredCodeQualityChecks(checks, roots)...)
	items := make([]codeQualityPreflightItem, 0, len(detected))
	ready := len(detected) > 0
	for _, check := range detected {
		item := codeQualityPreflightItem{
			ID: check.ID, Kind: check.Kind, Label: check.Label, Command: check.Command, WorkDir: check.WorkDir,
		}
		if _, _, resolveErr := resolveCodeQualityCommand(check); resolveErr != nil {
			item.Reason = fmt.Sprintf("质量检查命令不可用：%s", check.Executable)
			ready = false
		} else {
			item.Available = true
		}
		items = append(items, item)
	}
	return c.JSON(e.Succ(fiber.Map{"ready": ready, "items": items}))
}

func normalizeCodeProjectQualityChecks(
	checks []model.AIProjectQualityCheck,
	sourceDirs []string,
) ([]model.AIProjectQualityCheck, error) {
	if len(checks) > codeProjectQualityCheckLimit {
		return nil, fmt.Errorf("质量检查最多配置 %d 项", codeProjectQualityCheckLimit)
	}
	repositoryPaths, err := codeProjectQualityRepositories(sourceDirs)
	if err != nil {
		return nil, err
	}
	repositories := make(map[string]string, len(repositoryPaths))
	for _, repository := range repositoryPaths {
		repositories[filepath.Clean(repository)] = repository
	}
	result := make([]model.AIProjectQualityCheck, 0, len(checks))
	for index := range checks {
		check := checks[index]
		check.Name = strings.TrimSpace(check.Name)
		check.Kind = strings.TrimSpace(check.Kind)
		check.Repository = strings.TrimSpace(check.Repository)
		check.WorkDir = strings.TrimSpace(check.WorkDir)
		check.Command = strings.TrimSpace(check.Command)
		if check.Name == "" || check.Command == "" {
			return nil, fmt.Errorf("第 %d 项质量检查缺少名称或命令", index+1)
		}
		if check.Kind == "" {
			check.Kind = "test"
		}
		if !validCodeQualityKind(check.Kind) {
			return nil, fmt.Errorf("质量检查 %s 的类型无效", check.Name)
		}
		resolvedRepository, err := filepath.EvalSymlinks(filepath.Clean(check.Repository))
		if err != nil {
			return nil, fmt.Errorf("质量检查 %s 的仓库不可用", check.Name)
		}
		resolvedSource, ok := repositories[filepath.Clean(resolvedRepository)]
		if !ok {
			return nil, fmt.Errorf("质量检查 %s 的仓库不属于当前项目", check.Name)
		}
		check.Repository = resolvedSource
		if check.WorkDir == "" {
			check.WorkDir = "."
		}
		if filepath.IsAbs(check.WorkDir) || !isSafeRelativeCodeQualityPath(check.WorkDir) {
			return nil, fmt.Errorf("质量检查 %s 的工作目录必须位于仓库内", check.Name)
		}
		workDir := filepath.Join(check.Repository, filepath.Clean(check.WorkDir))
		resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
		if err != nil || !isPathInside(resolvedWorkDir, check.Repository) {
			return nil, fmt.Errorf("质量检查 %s 的工作目录不存在或越出仓库", check.Name)
		}
		if _, err := parseCodeQualityCommand(check.Command); err != nil {
			return nil, fmt.Errorf("质量检查 %s：%w", check.Name, err)
		}
		check.WorkDir = filepath.ToSlash(filepath.Clean(check.WorkDir))
		result = append(result, check)
	}
	return result, nil
}

func codeProjectQualityRepositories(sourceDirs []string) ([]string, error) {
	candidates, err := discoverCodeRepositoryCandidates(sourceDirs)
	if err != nil {
		return nil, err
	}
	repositories := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		repositories = append(repositories, candidate.SourceDir)
	}
	return repositories, nil
}

func validCodeQualityKind(kind string) bool {
	return kind == "test" || kind == "lint" || kind == "typecheck" || kind == "build"
}

func isSafeRelativeCodeQualityPath(path string) bool {
	cleaned := filepath.Clean(path)
	return cleaned != ".." && !strings.HasPrefix(cleaned, ".."+string(filepath.Separator))
}

func configuredCodeQualityChecks(
	configured []model.AIProjectQualityCheck,
	roots []codeDeliveryQualityRoot,
) []codeQualityCheck {
	checks := make([]codeQualityCheck, 0, len(configured))
	for _, item := range configured {
		for _, root := range roots {
			identity := strings.TrimSpace(root.RuntimeDir)
			if identity == "" {
				identity = root.IdentityDir
			}
			if filepath.Clean(identity) != filepath.Clean(item.Repository) {
				continue
			}
			parts, err := parseCodeQualityCommand(item.Command)
			if err != nil || len(parts) == 0 {
				continue
			}
			workDir := filepath.Join(root.WorkDir, filepath.FromSlash(item.WorkDir))
			resolvedRoot, rootErr := filepath.EvalSymlinks(root.WorkDir)
			resolvedWorkDir, workDirErr := filepath.EvalSymlinks(workDir)
			if rootErr != nil || workDirErr != nil || !isPathInside(resolvedWorkDir, resolvedRoot) {
				continue
			}
			check := newCodeQualityCheck(item.Kind, item.Name, resolvedWorkDir, resolvedRoot, parts[0], parts[1:]...)
			check.Command = item.Command
			check.LocalScript = strings.ContainsAny(parts[0], `/\`)
			checks = append(checks, check)
			break
		}
	}
	return checks
}

func loadConfiguredCodeQualityChecks(projectID uint, roots []codeDeliveryQualityRoot) []codeQualityCheck {
	if projectID == 0 {
		return nil
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(projectID)
	if err != nil {
		return nil
	}
	return configuredCodeQualityChecks(project.QualityChecks, roots)
}

func codeSessionQualityRoots(session *model.AIDevSession, paths []string) []codeDeliveryQualityRoot {
	roots := make([]codeDeliveryQualityRoot, 0, len(paths))
	if session.IsolationMode == codeIsolationMultiWorktree {
		repositories, _ := loadCodeSessionRepositories(session.ID)
		for _, path := range paths {
			root := codeDeliveryQualityRoot{WorkDir: path, IdentityDir: path, RuntimeDir: path}
			for _, repository := range repositories {
				if filepath.Clean(repository.WorktreeDir) == filepath.Clean(path) {
					root.IdentityDir, root.RuntimeDir = repository.SourceDir, repository.SourceDir
					break
				}
			}
			roots = append(roots, root)
		}
		return roots
	}
	for _, path := range paths {
		runtimeDir := path
		if strings.TrimSpace(session.SourceWorkDir) != "" {
			runtimeDir = session.SourceWorkDir
		}
		roots = append(roots, codeDeliveryQualityRoot{WorkDir: path, IdentityDir: runtimeDir, RuntimeDir: runtimeDir})
	}
	return roots
}

func mergeCodeQualityChecks(detected, configured []codeQualityCheck) []codeQualityCheck {
	configuredKeys := make(map[string]struct{}, len(configured))
	for _, check := range configured {
		configuredKeys[codeQualityCheckMergeKey(check)] = struct{}{}
	}
	result := make([]codeQualityCheck, 0, len(detected)+len(configured))
	for _, check := range detected {
		if _, replaced := configuredKeys[codeQualityCheckMergeKey(check)]; !replaced {
			result = append(result, check)
		}
	}
	return append(result, configured...)
}

func codeQualityCheckMergeKey(check codeQualityCheck) string {
	return filepath.Clean(check.workDirPath) + "\x00" + strings.TrimSpace(check.Command)
}

func parseCodeQualityCommand(command string) ([]string, error) {
	if strings.ContainsAny(command, "\r\n\x00") {
		return nil, errors.New("命令只能填写单行内容")
	}
	var parts []string
	var current strings.Builder
	quote := rune(0)
	escaped := false
	flush := func() {
		if current.Len() > 0 {
			parts = append(parts, current.String())
			current.Reset()
		}
	}
	for _, char := range strings.TrimSpace(command) {
		if escaped {
			current.WriteRune(char)
			escaped = false
			continue
		}
		if char == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			} else {
				current.WriteRune(char)
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		if unicode.IsSpace(char) {
			flush()
			continue
		}
		current.WriteRune(char)
	}
	if escaped || quote != 0 {
		return nil, errors.New("命令中的引号或转义不完整")
	}
	flush()
	if len(parts) == 0 {
		return nil, errors.New("命令不能为空")
	}
	for _, part := range parts {
		if part == "&&" || part == "||" || part == ";" || part == "|" || part == ">" || part == "<" {
			return nil, errors.New("每项只能配置一个命令，请将多个步骤拆成多条检查")
		}
	}
	return parts, nil
}

func resolveCodeQualityCommand(check codeQualityCheck) (string, []string, error) {
	if !check.LocalScript {
		return resolveCodeExecutorCommand(check.Executable)
	}
	commandPath := check.Executable
	if !filepath.IsAbs(commandPath) {
		commandPath = filepath.Join(check.workDirPath, commandPath)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(commandPath))
	if err != nil || !isPathInside(resolved, check.workDirPath) {
		return "", nil, os.ErrNotExist
	}
	info, err := os.Stat(resolved)
	if err != nil || info.IsDir() || info.Mode().Perm()&0111 == 0 {
		return "", nil, os.ErrPermission
	}
	return resolved, codeExecutorEnvironment(resolved, codeExecutorSearchDirs()), nil
}
