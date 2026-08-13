package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/container"
)

func (s *PipelineService) stepRunner(ctx context.Context, logger *PipelineLogger, p *model.Pipeline, workspaceDir string) (int, string, string, error) {
	if ctx.Err() != nil {
		return 0, "", "", ctx.Err()
	}
	codeRoot, err := resolveRunnerCodeRoot(logger, p, workspaceDir)
	if err != nil {
		return 0, "", "", err
	}
	var runnerCfg map[string]interface{}
	if strings.TrimSpace(p.RunnerConfig) != "" {
		_ = json.Unmarshal([]byte(p.RunnerConfig), &runnerCfg)
	}
	if runnerCfg == nil {
		runnerCfg = map[string]interface{}{}
	}
	s.logRunnerProjectProfile(logger, codeRoot, runnerCfg)
	if err := validateRunnerModeSource(codeRoot, runnerCfg); err != nil {
		return 0, "", "", err
	}
	previousContainerID := ""
	if prev, err := s.recordRepo.LatestRunnerContainerID(p.ID); err == nil {
		previousContainerID = strings.TrimSpace(prev)
	}
	progress := func(format string, a ...interface{}) {
		logger.Info("[Runner] "+format, a...)
	}
	alias := fmt.Sprintf("pipeline-%s", p.PipelineKey)
	deployRequest := pipelineRunnerDeployRequest{
		Alias:               alias,
		CodeRoot:            codeRoot,
		PipelineKey:         strings.TrimSpace(p.PipelineKey),
		PipelineVersion:     strings.TrimSpace(p.Version),
		PreviousContainerID: previousContainerID,
		Config:              runnerCfg,
	}
	hostPort, containerID, err := deployPipelineRunner(ctx, deployRequest, progress)
	if err != nil {
		return 0, "", "", err
	}
	return hostPort, containerID, codeRoot, nil
}

func (s *PipelineService) cleanupUnpublishedRunner(ctx context.Context, logger *PipelineLogger, pipelineID uint, containerID string) {
	containerID = strings.TrimSpace(containerID)
	if containerID == "" {
		return
	}
	previousContainerID, _ := s.recordRepo.LatestRunnerContainerID(pipelineID)
	previousContainerID = strings.TrimSpace(previousContainerID)
	if cleanupErr := cleanupPreviousContainer(containerID); cleanupErr != nil {
		logger.Error("清理未发布 Runner 容器失败: %v", cleanupErr)
		return
	}
	if previousContainerID == "" || previousContainerID == containerID {
		return
	}
	restoreCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := docker.NewDockerClient()
	if err != nil || cli == nil {
		if err != nil {
			logger.Error("恢复旧 Runner 容器失败: %v", err)
		}
		return
	}
	defer cli.Close()
	inspect, err := cli.ContainerInspect(restoreCtx, previousContainerID)
	if err != nil || (inspect.State != nil && inspect.State.Running) {
		return
	}
	if err := cli.ContainerStart(restoreCtx, previousContainerID, container.StartOptions{}); err != nil {
		logger.Error("恢复旧 Runner 容器失败: %v", err)
		return
	}
	logger.Info("未发布 Runner 已清理，旧 Runner 容器已恢复")
}
func resolveRunnerCodeRoot(logger *PipelineLogger, p *model.Pipeline, workspaceDir string) (string, error) {
	sourceDir := strings.TrimSpace(workspaceDir)
	sourceLabel := "工作区目录"
	if sourceDir == "" {
		return "", fmt.Errorf("Runner 工作区目录为空")
	}
	artifactPath := strings.TrimSpace(p.ArtifactPath)
	if artifactPath != "" {
		artifactSrc := filepath.Join(sourceDir, artifactPath)
		info, err := os.Stat(artifactSrc)
		if err == nil {
			if info.IsDir() {
				sourceDir = artifactSrc
				sourceLabel = fmt.Sprintf("产物目录(%s)", artifactPath)
			} else {
				sourceDir = filepath.Dir(artifactSrc)
				sourceLabel = fmt.Sprintf("产物所在目录(%s)", artifactPath)
			}
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("检查 Runner 产物目录失败: %w", err)
		}
	}
	codeRoot := detectRunnerCodeRoot(sourceDir)
	logger.Info("Runner: 直接使用%s作为运行目录: %s", sourceLabel, sourceDir)
	if codeRoot != sourceDir {
		logger.Info("Runner: 检测到单一子目录，自动切换代码根目录到 %s", codeRoot)
	}
	return codeRoot, nil
}

func validateRunnerModeSource(codeRoot string, runnerCfg map[string]interface{}) error {
	if err := ValidateRunnerPersistentPaths(runnerCfg); err != nil {
		return err
	}
	mode := strings.ToLower(strings.TrimSpace(asString(runnerCfg["mode"])))
	if mode == "" {
		mode = "build_run"
	}
	if mode != "build_run" {
		return nil
	}
	if strings.TrimSpace(asString(runnerCfg["buildCommand"])) != "" {
		return nil
	}
	if _, err := os.Stat(filepath.Join(codeRoot, ".output/server/index.mjs")); err == nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(codeRoot, "package.json")); err == nil {
		return nil
	}
	return fmt.Errorf("Runner 当前为 build_run 模式，但运行目录 %s 中既没有 package.json，也没有 .output/server/index.mjs，且未提供自定义 buildCommand；请改为 run 模式，或填写构建命令", codeRoot)
}
func ValidateRunnerPersistentPaths(runnerCfg map[string]interface{}) error {
	if runnerCfg == nil {
		return nil
	}
	paths := normalizeRunnerPersistentPaths(runnerCfg["persistentPaths"])
	for _, item := range paths {
		if isForbiddenRunnerPersistentPath(item) {
			return fmt.Errorf("Runner 持久化目录不支持 `%s`；`node_modules` 属于依赖目录，会导致 npm/pnpm 权限和脏缓存问题，请删除后重试", item)
		}
	}
	return nil
}
func normalizeRunnerPersistentPaths(raw interface{}) []string {
	switch v := raw.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if trimmed := strings.TrimSpace(asString(item)); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return nil
	}
}
func isForbiddenRunnerPersistentPath(raw string) bool {
	candidate := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if candidate == "" {
		return false
	}
	candidate = strings.TrimPrefix(path.Clean("/"+candidate), "/")
	return candidate == "node_modules" || strings.HasPrefix(candidate, "node_modules/")
}
func detectRunnerCodeRoot(releaseDir string) string {
	current := releaseDir
	for i := 0; i < 4; i++ {
		if runnerDirLooksLikeAppRoot(current) {
			return current
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return current
		}
		dirs := make([]string, 0, 1)
		for _, entry := range entries {
			name := entry.Name()
			if name == ".DS_Store" || name == "__MACOSX" {
				continue
			}
			if !entry.IsDir() {
				return current
			}
			dirs = append(dirs, filepath.Join(current, name))
		}
		if len(dirs) != 1 {
			return current
		}
		current = dirs[0]
	}
	return current
}
func runnerDirLooksLikeAppRoot(dir string) bool {
	markers := []string{"package.json", ".output", ".next", "dist", "Dockerfile", "index.html", "server.js", "docker-compose.yml"}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}
