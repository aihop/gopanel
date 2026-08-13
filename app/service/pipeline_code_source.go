package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

var codeSnapshotExcludedNames = map[string]struct{}{
	".git": {}, ".gopanel-project.json": {}, ".gopanel_artifact": {}, ".gopanel_shims": {},
	".env": {}, "node_modules": {}, "__MACOSX": {}, ".data": {},
}

func pipelineSourceType(pipeline *model.Pipeline) string {
	if pipeline != nil && strings.EqualFold(strings.TrimSpace(pipeline.SourceType), "code") {
		return "code"
	}
	return "git"
}

func (s *PipelineService) prepareCodeProjectSnapshot(ctx context.Context, logger *PipelineLogger, pipeline *model.Pipeline, workspace, expectedCommit string) (string, string, error) {
	if s.db == nil || pipeline == nil || pipeline.CodeProjectID == 0 {
		return "", "", errors.New("流水线未绑定 Code 项目")
	}
	var project model.AIProject
	if err := s.db.First(&project, pipeline.CodeProjectID).Error; err != nil {
		return "", "", fmt.Errorf("读取 Code 项目失败: %w", err)
	}
	if len(project.SourceDirs) == 0 {
		return "", "", errors.New("Code 项目没有可复制的源目录")
	}
	primarySource, commitHash, err := inspectCodeSnapshotCommit(ctx, &project)
	if err != nil {
		return "", "", err
	}
	if expectedCommit != "" && !pipelineCommitEqual(commitHash, expectedCommit) {
		return "", "", fmt.Errorf("Code 项目基准提交已变化: expected %s, got %s", expectedCommit, commitHash)
	}
	baseDir := filepath.Dir(workspace)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", "", err
	}
	tempDir, err := os.MkdirTemp(baseDir, ".workspace-snapshot-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tempDir)
	logger.Info("正在冻结 Code 项目快照: %s", project.Name)
	if err := copyCodeProjectSources(ctx, project.SourceDirs, tempDir); err != nil {
		return "", "", err
	}
	digest, _, err := pipelineDirectoryDigest(tempDir)
	if err != nil {
		return "", "", fmt.Errorf("计算 Code 项目快照摘要失败: %w", err)
	}
	if err := replacePipelineWorkspace(workspace, tempDir); err != nil {
		return "", "", err
	}
	logger.Info("Code 项目快照已就绪: project=%d, primary=%s, commit=%s, digest=%s", project.ID, filepath.Base(primarySource), commitHash, digest)
	return commitHash, digest, nil
}

func inspectCodeSnapshotCommit(ctx context.Context, project *model.AIProject) (string, string, error) {
	primary := strings.TrimSpace(project.PrimaryRepository)
	if primary == "" {
		primary = strings.TrimSpace(project.SourceDirs[0])
	}
	matched := false
	for _, sourceDir := range project.SourceDirs {
		if codeSnapshotPathWithin(sourceDir, primary) {
			matched = true
			break
		}
	}
	if !matched {
		return "", "", errors.New("Code 项目主仓库不在项目源目录中")
	}
	command := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	command.Dir = primary
	output, err := command.Output()
	if err != nil {
		return primary, "", nil
	}
	commit := strings.ToLower(strings.TrimSpace(string(output)))
	if !pipelineCommitPattern.MatchString(commit) {
		return "", "", errors.New("Code 项目基准提交无效")
	}
	return primary, commit, nil
}

func codeSnapshotPathWithin(baseDir, target string) bool {
	relative, err := filepath.Rel(filepath.Clean(baseDir), filepath.Clean(target))
	return err == nil && !filepath.IsAbs(relative) && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func copyCodeProjectSources(ctx context.Context, sourceDirs []string, destination string) error {
	multiSource := len(sourceDirs) > 1
	usedNames := make(map[string]struct{}, len(sourceDirs))
	for _, rawSource := range sourceDirs {
		if err := ctx.Err(); err != nil {
			return err
		}
		source, err := filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(rawSource)))
		if err != nil {
			return fmt.Errorf("Code 项目源目录不可访问: %s", rawSource)
		}
		info, err := os.Stat(source)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("Code 项目源目录无效: %s", rawSource)
		}
		target := destination
		if multiSource {
			name := uniqueCodeSnapshotName(filepath.Base(source), usedNames)
			target = filepath.Join(destination, name)
			if err := os.MkdirAll(target, info.Mode()); err != nil {
				return err
			}
		}
		if err := validateCodeSnapshotSymlinks(source); err != nil {
			return err
		}
		if err := copyPipelineTree(source, target, codeSnapshotExcludedNames); err != nil {
			return fmt.Errorf("复制 Code 项目目录 %s 失败: %w", filepath.Base(source), err)
		}
	}
	return nil
}

func validateCodeSnapshotSymlinks(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root || shouldSkipPipelineReleaseEntry(path, info, codeSnapshotExcludedNames) {
			if path != root && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return nil
		}
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}
		if filepath.IsAbs(target) {
			return fmt.Errorf("Code 项目包含绝对符号链接，无法安全快照: %s", path)
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(path), target))
		if !codeSnapshotPathWithin(root, resolved) {
			return fmt.Errorf("Code 项目符号链接指向源目录外，无法安全快照: %s", path)
		}
		return nil
	})
}

func uniqueCodeSnapshotName(name string, used map[string]struct{}) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "source"
	}
	candidate := name
	for suffix := 2; ; suffix++ {
		if _, exists := used[candidate]; !exists {
			used[candidate] = struct{}{}
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", name, suffix)
	}
}

func replacePipelineWorkspace(workspace, snapshot string) error {
	backup := workspace + ".previous"
	if err := os.RemoveAll(backup); err != nil {
		return err
	}
	workspaceExists := false
	if _, err := os.Stat(workspace); err == nil {
		workspaceExists = true
		if err := os.Rename(workspace, backup); err != nil {
			return fmt.Errorf("备份流水线工作区失败: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(snapshot, workspace); err != nil {
		if workspaceExists {
			_ = os.Rename(backup, workspace)
		}
		return fmt.Errorf("启用 Code 项目快照失败: %w", err)
	}
	if workspaceExists {
		_ = os.RemoveAll(backup)
	}
	return nil
}
