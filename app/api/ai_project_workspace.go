package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
)

const (
	aiProjectWorkspaceManifestName = ".gopanel-project.json"
	maxAIProjectSourceDirs         = 20
)

type aiProjectWorkspaceSource struct {
	Path     string `json:"path"`
	LinkName string `json:"linkName"`
}

type aiProjectWorkspaceManifest struct {
	Version int                        `json:"version"`
	Sources []aiProjectWorkspaceSource `json:"sources"`
}

func aiProjectCodeRoot() string {
	baseDir := filepath.Clean(global.CONF.System.BaseDir)
	if resolvedBaseDir, err := filepath.EvalSymlinks(baseDir); err == nil {
		baseDir = resolvedBaseDir
	}
	return filepath.Join(baseDir, "code")
}

func aiProjectUserRoot(userID uint) string {
	return filepath.Join(aiProjectCodeRoot(), fmt.Sprintf("user_%d", userID))
}

func aiProjectWorkspaceDir(userID, projectID uint) string {
	return filepath.Join(aiProjectUserRoot(userID), fmt.Sprintf("project_%d", projectID))
}

func pathWithinDirectory(baseDir, targetPath string) bool {
	relativePath, err := filepath.Rel(filepath.Clean(baseDir), filepath.Clean(targetPath))
	return err == nil && !filepath.IsAbs(relativePath) && relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))
}

func isManagedAIProjectWorkDir(workDir string, userID uint) bool {
	workDir = filepath.Clean(strings.TrimSpace(workDir))
	if workDir == "." || !filepath.IsAbs(workDir) {
		return false
	}
	if !pathWithinDirectory(aiProjectUserRoot(userID), workDir) || filepath.Dir(workDir) != aiProjectUserRoot(userID) {
		return false
	}
	return isAIProjectWorkspaceDirectory(workDir)
}

func isAnyManagedAIProjectWorkDir(workDir string) bool {
	workDir = filepath.Clean(strings.TrimSpace(workDir))
	if workDir == "." || !filepath.IsAbs(workDir) {
		return false
	}
	userRoot := filepath.Dir(workDir)
	return filepath.Dir(userRoot) == aiProjectCodeRoot() && strings.HasPrefix(filepath.Base(userRoot), "user_") && isAIProjectWorkspaceDirectory(workDir)
}

func isAIProjectWorkspaceDirectory(workDir string) bool {
	if !strings.HasPrefix(filepath.Base(workDir), "project_") {
		return false
	}
	info, err := os.Stat(filepath.Join(workDir, aiProjectWorkspaceManifestName))
	return err == nil && info.Mode().IsRegular()
}

func validateAIProjectWorkDirForClaims(workDir string, claims *token.CustomClaims) error {
	if claims == nil {
		return errors.New("未授权访问项目目录")
	}
	if claims.Role != constant.UserRoleSubAdmin || isManagedAIProjectWorkDir(workDir, claims.UserId) || isManagedAISessionWorkDir(workDir, claims.UserId) {
		return nil
	}
	return service.ValidatePathWithinBase(claims.FileBaseDir, workDir)
}

func normalizeAIProjectSourceDirs(sourceDirs []string, claims *token.CustomClaims) ([]string, error) {
	if len(sourceDirs) == 0 {
		return nil, errors.New("请至少选择一个项目目录")
	}
	if len(sourceDirs) > maxAIProjectSourceDirs {
		return nil, fmt.Errorf("一个项目最多选择 %d 个目录", maxAIProjectSourceDirs)
	}
	codeRoot, err := filepath.Abs(filepath.Clean(aiProjectCodeRoot()))
	if err != nil {
		return nil, errors.New("GoPanel Code 工作区路径无效")
	}
	seen := make(map[string]struct{}, len(sourceDirs))
	normalized := make([]string, 0, len(sourceDirs))
	for _, sourceDir := range sourceDirs {
		resolvedPath, normalizeErr := normalizeAIProjectSourceDir(sourceDir, claims)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if pathWithinDirectory(resolvedPath, codeRoot) || pathWithinDirectory(codeRoot, resolvedPath) {
			return nil, errors.New("不能选择 GoPanel Code 工作区或其上级目录")
		}
		if _, exists := seen[resolvedPath]; exists {
			continue
		}
		seen[resolvedPath] = struct{}{}
		normalized = append(normalized, resolvedPath)
	}
	return normalized, nil
}

func normalizeAIProjectSourceDir(sourceDir string, claims *token.CustomClaims) (string, error) {
	sourceDir = strings.TrimSpace(sourceDir)
	if sourceDir == "" || !filepath.IsAbs(sourceDir) {
		return "", errors.New("项目目录必须是有效的绝对目录")
	}
	resolvedPath, err := filepath.EvalSymlinks(filepath.Clean(sourceDir))
	if err != nil {
		return "", fmt.Errorf("项目目录不存在或无法访问：%s", sourceDir)
	}
	info, err := os.Stat(resolvedPath)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("项目路径必须是可访问的目录：%s", sourceDir)
	}
	if claims != nil && claims.Role == constant.UserRoleSubAdmin {
		if err := service.ValidatePathWithinBase(claims.FileBaseDir, resolvedPath); err != nil {
			return "", err
		}
	}
	return resolvedPath, nil
}

func readAIProjectWorkspaceManifest(workspaceDir string) (aiProjectWorkspaceManifest, error) {
	var manifest aiProjectWorkspaceManifest
	content, err := os.ReadFile(filepath.Join(workspaceDir, aiProjectWorkspaceManifestName))
	if errors.Is(err, os.ErrNotExist) {
		return manifest, nil
	}
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return manifest, errors.New("项目工作区元数据损坏")
	}
	return manifest, nil
}

func writeAIProjectWorkspaceManifest(workspaceDir string, manifest aiProjectWorkspaceManifest) error {
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(workspaceDir, ".gopanel-project-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if err := tempFile.Chmod(0600); err != nil {
		tempFile.Close()
		return err
	}
	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, filepath.Join(workspaceDir, aiProjectWorkspaceManifestName))
}

func nextAIProjectLinkName(sourceDir string, usedNames map[string]struct{}) string {
	baseName := filepath.Base(sourceDir)
	if baseName == "" || baseName == "." || baseName == string(filepath.Separator) || baseName == aiProjectWorkspaceManifestName {
		baseName = "source"
	}
	name := baseName
	for suffix := 2; ; suffix++ {
		if _, exists := usedNames[name]; !exists {
			usedNames[name] = struct{}{}
			return name
		}
		name = fmt.Sprintf("%s-%d", baseName, suffix)
	}
}

func buildAIProjectWorkspaceSources(sourceDirs []string, previous aiProjectWorkspaceManifest, reservedNames map[string]struct{}) []aiProjectWorkspaceSource {
	previousNames := make(map[string]string, len(previous.Sources))
	usedNames := make(map[string]struct{}, len(reservedNames)+1)
	usedNames[aiProjectWorkspaceManifestName] = struct{}{}
	for name := range reservedNames {
		usedNames[name] = struct{}{}
	}
	for _, source := range previous.Sources {
		if source.Path != "" && source.LinkName != "" && filepath.Base(source.LinkName) == source.LinkName {
			previousNames[source.Path] = source.LinkName
		}
	}
	result := make([]aiProjectWorkspaceSource, 0, len(sourceDirs))
	for _, sourceDir := range sourceDirs {
		linkName := previousNames[sourceDir]
		if linkName != "" {
			if _, exists := usedNames[linkName]; exists {
				linkName = ""
			} else {
				usedNames[linkName] = struct{}{}
			}
		}
		if linkName == "" {
			linkName = nextAIProjectLinkName(sourceDir, usedNames)
		}
		result = append(result, aiProjectWorkspaceSource{Path: sourceDir, LinkName: linkName})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].LinkName < result[j].LinkName })
	return result
}

func syncAIProjectWorkspace(project *model.AIProject, sourceDirs []string) (string, error) {
	if len(sourceDirs) == 1 {
		return sourceDirs[0], nil
	}
	workspaceDir := aiProjectWorkspaceDir(project.CreatorID, project.ID)
	if err := os.MkdirAll(workspaceDir, 0750); err != nil {
		return "", fmt.Errorf("创建项目工作区失败：%w", err)
	}
	previous, err := readAIProjectWorkspaceManifest(workspaceDir)
	if err != nil {
		return "", err
	}
	managedNames := make(map[string]struct{}, len(previous.Sources))
	for _, source := range previous.Sources {
		managedNames[source.LinkName] = struct{}{}
	}
	reservedNames := make(map[string]struct{})
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if _, managed := managedNames[entry.Name()]; !managed {
			reservedNames[entry.Name()] = struct{}{}
		}
	}
	desiredSources := buildAIProjectWorkspaceSources(sourceDirs, previous, reservedNames)
	desiredByName := make(map[string]aiProjectWorkspaceSource, len(desiredSources))
	previousByName := make(map[string]aiProjectWorkspaceSource, len(previous.Sources))
	for _, source := range previous.Sources {
		previousByName[source.LinkName] = source
	}
	for _, source := range desiredSources {
		desiredByName[source.LinkName] = source
		linkPath := filepath.Join(workspaceDir, source.LinkName)
		info, statErr := os.Lstat(linkPath)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
			return "", fmt.Errorf("工作区中已存在同名文件：%s", source.LinkName)
		}
		resolvedTarget, evalErr := filepath.EvalSymlinks(linkPath)
		previousSource, wasManaged := previousByName[source.LinkName]
		if evalErr != nil || !wasManaged || filepath.Clean(resolvedTarget) != filepath.Clean(previousSource.Path) {
			return "", fmt.Errorf("工作区中的链接已被修改：%s", source.LinkName)
		}
		if filepath.Clean(resolvedTarget) != filepath.Clean(source.Path) {
			if err := os.Remove(linkPath); err != nil {
				return "", err
			}
		}
	}
	for _, source := range desiredSources {
		linkPath := filepath.Join(workspaceDir, source.LinkName)
		if _, err := os.Lstat(linkPath); errors.Is(err, os.ErrNotExist) {
			if err := os.Symlink(source.Path, linkPath); err != nil {
				return "", fmt.Errorf("创建目录链接失败：%w", err)
			}
		}
	}
	for _, source := range previous.Sources {
		if _, keep := desiredByName[source.LinkName]; keep {
			continue
		}
		linkPath := filepath.Join(workspaceDir, source.LinkName)
		info, statErr := os.Lstat(linkPath)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
			return "", fmt.Errorf("无法安全移除已修改的工作区链接：%s", source.LinkName)
		}
		if err := os.Remove(linkPath); err != nil {
			return "", err
		}
	}
	manifest := aiProjectWorkspaceManifest{Version: 1, Sources: desiredSources}
	if err := writeAIProjectWorkspaceManifest(workspaceDir, manifest); err != nil {
		return "", fmt.Errorf("保存项目工作区元数据失败：%w", err)
	}
	return workspaceDir, nil
}

func aiProjectSessionWorkDir(project *model.AIProject, claims *token.CustomClaims) (string, error) {
	if project == nil {
		return "", errors.New("项目不存在")
	}
	if len(project.SourceDirs) == 1 {
		return normalizeAIProjectWorkDir(project.SourceDirs[0], claims)
	}
	return normalizeAIProjectWorkDir(project.WorkDir, claims)
}

func aiProjectWorkspaceSourceDirs(workspaceDir string) []string {
	manifest, err := readAIProjectWorkspaceManifest(workspaceDir)
	if err != nil {
		return nil
	}
	sourceDirs := make([]string, 0, len(manifest.Sources))
	for _, source := range manifest.Sources {
		if source.Path != "" {
			sourceDirs = append(sourceDirs, source.Path)
		}
	}
	return sourceDirs
}
