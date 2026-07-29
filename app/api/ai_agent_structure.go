package api

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const maxAIStructureEntries = 500
const maxAISessionFileSize = 2 * 1024 * 1024

var ignoredAIStructureNames = map[string]struct{}{
	".git":                  {},
	".gopanel-project.json": {},
	"node_modules":          {},
}

type aiStructureEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsDir     bool   `json:"isDir"`
	Extension string `json:"extension"`
}

type aiStructureResult struct {
	Path      string             `json:"path"`
	Entries   []aiStructureEntry `json:"entries"`
	Truncated bool               `json:"truncated"`
}

func aiStructureRoots(workDir string, sourceDirs []string) ([]string, error) {
	root, err := filepath.Abs(filepath.Clean(strings.TrimSpace(workDir)))
	if err != nil {
		return nil, errors.New("项目工作目录无效")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, errors.New("项目工作目录不存在或无法访问")
	}
	roots := []string{filepath.Clean(resolvedRoot)}
	if isAIProjectWorkspaceDirectory(root) {
		for _, sourceDir := range sourceDirs {
			resolvedSource, resolveErr := filepath.EvalSymlinks(sourceDir)
			if resolveErr == nil {
				roots = append(roots, filepath.Clean(resolvedSource))
			}
		}
	}
	return roots, nil
}

func isPathWithinAnyRoot(target string, roots []string) bool {
	for _, root := range roots {
		if pathWithinDirectory(root, target) {
			return true
		}
	}
	return false
}

func resolveAIStructurePath(workDir, relativePath string, sourceDirs []string) (string, string, []string, error) {
	roots, err := aiStructureRoots(workDir, sourceDirs)
	if err != nil {
		return "", "", nil, err
	}
	relativePath = strings.ReplaceAll(strings.TrimSpace(relativePath), "\\", "/")
	cleanRelative := path.Clean(relativePath)
	if cleanRelative == "." {
		cleanRelative = ""
	}
	if path.IsAbs(cleanRelative) || cleanRelative == ".." || strings.HasPrefix(cleanRelative, "../") {
		return "", "", nil, errors.New("目录路径无效")
	}
	root, err := filepath.Abs(filepath.Clean(strings.TrimSpace(workDir)))
	if err != nil {
		return "", "", nil, errors.New("项目工作目录无效")
	}
	resolvedTarget, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(cleanRelative)))
	if err != nil {
		return "", "", nil, errors.New("目录不存在或无法访问")
	}
	if !isPathWithinAnyRoot(filepath.Clean(resolvedTarget), roots) {
		return "", "", nil, errors.New("目录超出当前项目范围")
	}
	info, err := os.Stat(resolvedTarget)
	if err != nil || !info.IsDir() {
		return "", "", nil, errors.New("请求路径不是目录")
	}
	return filepath.Clean(resolvedTarget), cleanRelative, roots, nil
}

func resolveAISessionFilePath(workDir, relativePath string, sourceDirs []string) (string, string, error) {
	roots, err := aiStructureRoots(workDir, sourceDirs)
	if err != nil {
		return "", "", err
	}
	relativePath = strings.ReplaceAll(strings.TrimSpace(relativePath), "\\", "/")
	cleanRelative := path.Clean(relativePath)
	if cleanRelative == "." || path.IsAbs(cleanRelative) || cleanRelative == ".." || strings.HasPrefix(cleanRelative, "../") {
		return "", "", errors.New("文件路径无效")
	}
	root, err := filepath.Abs(filepath.Clean(strings.TrimSpace(workDir)))
	if err != nil {
		return "", "", errors.New("项目工作目录无效")
	}
	resolvedTarget, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(cleanRelative)))
	if err != nil || !isPathWithinAnyRoot(filepath.Clean(resolvedTarget), roots) {
		return "", "", errors.New("文件不存在或超出当前项目范围")
	}
	info, err := os.Stat(resolvedTarget)
	if err != nil || !info.Mode().IsRegular() {
		return "", "", errors.New("请求路径不是普通文件")
	}
	if info.Size() > maxAISessionFileSize {
		return "", "", errors.New("文件超过 2 MB，无法在代码编辑器中打开")
	}
	return filepath.Clean(resolvedTarget), cleanRelative, nil
}

func getAISessionSourceDirs(sessionProjectID uint, claims *token.CustomClaims) ([]string, error) {
	if sessionProjectID == 0 {
		return nil, nil
	}
	project, err := repo.NewAIGroupRepo().GetGroupByID(sessionProjectID)
	if err != nil || !canManageAIProject(project, claims) {
		return nil, errors.New("无权访问该项目目录")
	}
	return project.SourceDirs, nil
}

func getAISessionFileContext(c fiber.Ctx) (string, []string, error) {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return "", nil, errors.New("会话 ID 无效")
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return "", nil, err
	}
	if err := validateAIProjectWorkDirForClaims(session.WorkDir, claims); err != nil {
		return "", nil, err
	}
	sourceDirs, err := getAISessionSourceDirs(session.ProjectID, claims)
	if err != nil {
		return "", nil, err
	}
	return session.WorkDir, sourceDirs, nil
}

func readAISessionFile(workDir, relativePath string, sourceDirs []string) (fiber.Map, error) {
	target, cleanRelative, err := resolveAISessionFilePath(workDir, relativePath, sourceDirs)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(content) || strings.IndexByte(string(content), 0) >= 0 {
		return nil, errors.New("该文件不是可编辑的 UTF-8 文本文件")
	}
	return fiber.Map{
		"path": cleanRelative, "content": string(content),
		"extension": strings.TrimPrefix(strings.ToLower(filepath.Ext(cleanRelative)), "."), "size": len(content),
	}, nil
}

func writeAISessionFile(workDir, relativePath, content string, sourceDirs []string) error {
	if len([]byte(content)) > maxAISessionFileSize || !utf8.ValidString(content) {
		return errors.New("文件内容必须是 2 MB 以内的 UTF-8 文本")
	}
	target, _, err := resolveAISessionFilePath(workDir, relativePath, sourceDirs)
	if err != nil {
		return err
	}
	return os.WriteFile(target, []byte(content), 0)
}

func GetAISessionFile(c fiber.Ctx) error {
	workDir, sourceDirs, err := getAISessionFileContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	result, err := readAISessionFile(workDir, c.Query("path"), sourceDirs)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

func SaveAISessionFile(c fiber.Ctx) error {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	workDir, sourceDirs, err := getAISessionFileContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := writeAISessionFile(workDir, req.Path, req.Content, sourceDirs); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"path": path.Clean(req.Path), "size": len([]byte(req.Content))}))
}

func listAISessionStructure(workDir, relativePath string, sourceDirs []string) (aiStructureResult, error) {
	target, cleanRelative, roots, err := resolveAIStructurePath(workDir, relativePath, sourceDirs)
	if err != nil {
		return aiStructureResult{}, err
	}
	directoryEntries, err := os.ReadDir(target)
	if err != nil {
		return aiStructureResult{}, err
	}
	result := aiStructureResult{Path: cleanRelative, Entries: make([]aiStructureEntry, 0, len(directoryEntries))}
	for _, entry := range directoryEntries {
		if _, ignored := ignoredAIStructureNames[entry.Name()]; ignored {
			continue
		}
		entryTarget, resolveErr := filepath.EvalSymlinks(filepath.Join(target, entry.Name()))
		if resolveErr != nil || !isPathWithinAnyRoot(filepath.Clean(entryTarget), roots) {
			continue
		}
		info, statErr := os.Stat(entryTarget)
		if statErr != nil {
			continue
		}
		result.Entries = append(result.Entries, aiStructureEntry{
			Name:      entry.Name(),
			Path:      path.Join(cleanRelative, entry.Name()),
			IsDir:     info.IsDir(),
			Extension: strings.TrimPrefix(strings.ToLower(filepath.Ext(entry.Name())), "."),
		})
	}
	sort.SliceStable(result.Entries, func(left, right int) bool {
		if result.Entries[left].IsDir != result.Entries[right].IsDir {
			return result.Entries[left].IsDir
		}
		return strings.ToLower(result.Entries[left].Name) < strings.ToLower(result.Entries[right].Name)
	})
	if len(result.Entries) > maxAIStructureEntries {
		result.Entries = result.Entries[:maxAIStructureEntries]
		result.Truncated = true
	}
	return result, nil
}

func GetAISessionStructure(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return c.JSON(e.Fail(errors.New("会话 ID 无效")))
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := validateAIProjectWorkDirForClaims(session.WorkDir, claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	sourceDirs, err := getAISessionSourceDirs(session.ProjectID, claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	result, err := listAISessionStructure(session.WorkDir, c.Query("path"), sourceDirs)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}
