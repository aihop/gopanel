package service

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/utils/docker"
)

func snapshotPipelineRunnerArtifact(ctx context.Context, logger *PipelineLogger, pipeline *model.Pipeline, recordID uint, containerID string) (string, error) {
	containerID = strings.TrimSpace(containerID)
	if pipeline == nil || recordID == 0 || containerID == "" {
		return "", errors.New("Runner 构建结果缺少流水线、执行记录或容器信息")
	}
	cli, err := docker.NewDockerClient()
	if err != nil {
		return "", fmt.Errorf("初始化容器客户端失败: %w", err)
	}
	if cli == nil {
		return "", errors.New("当前容器运行时不可用")
	}
	defer cli.Close()

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("读取 Runner 容器信息失败: %w", err)
	}
	workingDir := ""
	if inspect.Config != nil {
		workingDir = strings.TrimSpace(inspect.Config.WorkingDir)
	}
	if workingDir == "" {
		return "", errors.New("Runner 容器工作目录为空，无法固化构建结果")
	}
	reader, stat, err := cli.CopyFromContainer(ctx, containerID, workingDir)
	if err != nil {
		return "", fmt.Errorf("从 Runner 容器导出构建结果失败: %w", err)
	}
	defer reader.Close()

	archiveDir := pipelineArchiveDir(pipeline)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", err
	}
	temporaryDir, err := os.MkdirTemp(archiveDir, fmt.Sprintf(".runner-record-%d-", recordID))
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temporaryDir)
	if logger != nil {
		logger.Info("正在从 Runner 容器固化构建结果: %s", workingDir)
	}
	excludedPaths := runnerArtifactExcludedPaths(pipeline, workingDir)
	if err := extractRunnerArtifactTar(ctx, reader, temporaryDir, stat.Name, excludedPaths); err != nil {
		return "", fmt.Errorf("解包 Runner 构建结果失败: %w", err)
	}
	if err := validatePipelineRunnerArtifact(pipeline, temporaryDir); err != nil {
		return "", err
	}
	resultDir := filepath.Join(archiveDir, fmt.Sprintf("runner-record-%d", recordID))
	if err := os.RemoveAll(resultDir); err != nil {
		return "", err
	}
	if err := os.Rename(temporaryDir, resultDir); err != nil {
		return "", err
	}
	if logger != nil {
		logger.Info("Runner 构建结果已固化: %s", resultDir)
	}
	return resultDir, nil
}

func validatePipelineRunnerArtifact(pipeline *model.Pipeline, artifactDir string) error {
	if pipeline == nil {
		return errors.New("Runner 制品校验缺少流水线信息")
	}
	raw := map[string]interface{}{}
	if strings.TrimSpace(pipeline.RunnerConfig) != "" {
		if err := json.Unmarshal([]byte(pipeline.RunnerConfig), &raw); err != nil {
			return fmt.Errorf("Runner 配置无效，无法校验正式制品: %w", err)
		}
	}
	startCommand := strings.TrimSpace(parseRunnerConfig(raw).StartCommand)
	requiredPath := runnerStartCommandArtifactPath(startCommand)
	if requiredPath == "" {
		return nil
	}
	target, err := safeRunnerArtifactTarget(artifactDir, filepath.FromSlash(requiredPath))
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Runner 构建未生成启动入口 %s，禁止发布正式制品", requiredPath)
		}
		return fmt.Errorf("检查 Runner 启动入口 %s 失败: %w", requiredPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("Runner 启动入口 %s 不是文件，禁止发布正式制品", requiredPath)
	}
	return nil
}

func runnerStartCommandArtifactPath(command string) string {
	for _, field := range strings.Fields(command) {
		candidate := strings.Trim(field, "'\"")
		if candidate == "" || strings.HasPrefix(candidate, "-") || strings.ContainsAny(candidate, "$|&;<>(){}*?") {
			continue
		}
		explicitRelative := strings.HasPrefix(candidate, "./")
		candidate = strings.TrimPrefix(candidate, "./")
		candidate = path.Clean(candidate)
		if candidate == "." || path.IsAbs(candidate) || candidate == ".." || strings.HasPrefix(candidate, "../") {
			continue
		}
		extension := strings.ToLower(path.Ext(candidate))
		switch extension {
		case ".js", ".mjs", ".cjs", ".ts", ".py", ".php", ".rb", ".jar", ".html":
			return candidate
		}
		if explicitRelative || candidate == "artisan" {
			return candidate
		}
	}
	return ""
}

func runnerArtifactExcludedPaths(pipeline *model.Pipeline, workingDir string) map[string]struct{} {
	excluded := map[string]struct{}{}
	if pipeline == nil || strings.TrimSpace(pipeline.RunnerConfig) == "" {
		return excluded
	}
	raw := map[string]interface{}{}
	if json.Unmarshal([]byte(pipeline.RunnerConfig), &raw) != nil {
		return excluded
	}
	runner := parseRunnerConfig(raw)
	for _, persistentPath := range runner.PersistentPaths {
		relative, _, ok := normalizeRunnerPersistentPath(workingDir, persistentPath)
		if ok {
			excluded[filepath.ToSlash(filepath.Clean(relative))] = struct{}{}
		}
	}
	return excluded
}

func extractRunnerArtifactTar(ctx context.Context, reader io.Reader, destination, rootName string, excludedPaths map[string]struct{}) error {
	archive := tar.NewReader(reader)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		relative, err := runnerArtifactRelativePath(header.Name, rootName)
		if err != nil {
			return err
		}
		if relative == "." || shouldSkipRunnerArtifactEntry(relative, excludedPaths) {
			continue
		}
		target, err := safeRunnerArtifactTarget(destination, relative)
		if err != nil {
			return err
		}
		mode := os.FileMode(header.Mode) & os.ModePerm
		switch header.Typeflag {
		case tar.TypeXGlobalHeader, tar.TypeXHeader:
			continue
		case tar.TypeDir:
			if err := os.MkdirAll(target, mode); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(file, archive)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink:
			if path.IsAbs(header.Linkname) {
				return fmt.Errorf("Runner 构建结果包含绝对符号链接: %s", header.Name)
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(target), filepath.FromSlash(header.Linkname)))
			if !codeSnapshotPathWithin(destination, resolved) {
				return fmt.Errorf("Runner 构建结果包含越界符号链接: %s", header.Name)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		case tar.TypeLink:
			linkRelative, err := runnerArtifactRelativePath(header.Linkname, rootName)
			if err != nil || shouldSkipRunnerArtifactEntry(linkRelative, excludedPaths) {
				return fmt.Errorf("Runner 构建结果包含非法硬链接: %s", header.Name)
			}
			linkTarget, err := safeRunnerArtifactTarget(destination, linkRelative)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Link(linkTarget, target); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Runner 构建结果包含不支持的文件类型: %s", header.Name)
		}
	}
}

func runnerArtifactRelativePath(name, rootName string) (string, error) {
	cleanName := path.Clean(strings.TrimPrefix(strings.TrimSpace(name), "./"))
	cleanRoot := path.Clean(strings.TrimPrefix(strings.TrimSpace(rootName), "./"))
	if cleanName == "." || cleanName == cleanRoot {
		return ".", nil
	}
	if cleanRoot != "." && strings.HasPrefix(cleanName, cleanRoot+"/") {
		cleanName = strings.TrimPrefix(cleanName, cleanRoot+"/")
	}
	if cleanName == "" || cleanName == "." || path.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, "../") {
		return "", fmt.Errorf("Runner 构建结果包含非法路径: %s", name)
	}
	return filepath.FromSlash(cleanName), nil
}

func safeRunnerArtifactTarget(root, relative string) (string, error) {
	target := filepath.Join(root, filepath.Clean(relative))
	if !codeSnapshotPathWithin(root, target) {
		return "", fmt.Errorf("Runner 构建结果路径超出固化目录: %s", relative)
	}
	return target, nil
}

func shouldSkipRunnerArtifactEntry(relative string, excludedPaths map[string]struct{}) bool {
	clean := filepath.ToSlash(filepath.Clean(relative))
	for _, part := range strings.Split(clean, "/") {
		if part == "node_modules" && clean != "node_modules" && !strings.HasPrefix(clean, "node_modules/") {
			continue
		}
		if _, ok := archiveExcludedNames[part]; ok {
			return true
		}
	}
	for excluded := range excludedPaths {
		if clean == excluded || strings.HasPrefix(clean, excluded+"/") {
			return true
		}
	}
	return false
}
