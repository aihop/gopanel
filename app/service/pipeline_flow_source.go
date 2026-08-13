package service

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

const flowBuildFactFileName = "gopanel_flow.json"

const codeSourceMaterializeTimeout = 15 * time.Minute

const codeSourceRepositoryTimeout = 5 * time.Minute

type flowBuildFact struct {
	SchemaVersion int                          `json:"schemaVersion"`
	FlowRunID     uint                         `json:"flowRunId"`
	Version       string                       `json:"version"`
	SourceType    string                       `json:"sourceType"`
	SourceDigest  string                       `json:"sourceDigest"`
	DeliveryJobID uint                         `json:"deliveryJobId"`
	SessionID     uint                         `json:"sessionId"`
	TaskID        uint                         `json:"taskId"`
	TaskTitle     string                       `json:"taskTitle"`
	Repositories  []model.FlowSourceRepository `json:"repositories"`
}

func (s *PipelineService) prepareCodePipelineSource(
	ctx context.Context,
	logger *PipelineLogger,
	pipeline *model.Pipeline,
	record *model.PipelineRecord,
	workspace string,
) (string, string, error) {
	if record != nil && record.SourceType == "flow_run" && record.SourceID > 0 {
		return s.prepareLockedCodeSource(ctx, logger, pipeline, record, workspace)
	}
	return s.prepareCodeProjectSnapshot(ctx, logger, pipeline, workspace, record.ExpectedCommit)
}

func (s *PipelineService) prepareLockedCodeSource(
	ctx context.Context,
	logger *PipelineLogger,
	pipeline *model.Pipeline,
	record *model.PipelineRecord,
	workspace string,
) (string, string, error) {
	var run model.FlowRun
	if err := s.db.First(&run, record.SourceID).Error; err != nil {
		return "", "", fmt.Errorf("读取 Flow 正式版本失败: %w", err)
	}
	if run.PipelineID != pipeline.ID || run.ProjectID != pipeline.CodeProjectID || !isFlowCodeSourceType(run.SourceType) {
		return "", "", errors.New("Flow 正式版本与流水线代码来源不一致")
	}
	var manifest flowSourceManifest
	if err := json.Unmarshal([]byte(run.SourceManifest), &manifest); err != nil || len(manifest.Repositories) == 0 {
		return "", "", errors.New("Flow 正式版本的代码来源清单无效")
	}
	if manifest.SchemaVersion != flowSourceManifestLegacySchemaVersion && manifest.SchemaVersion != flowSourceManifestSchemaVersion {
		return "", "", errors.New("Flow 正式版本的代码来源清单版本不受支持")
	}
	digest, err := flowSourceManifestDigest(manifest)
	if err != nil || digest != run.SourceDigest {
		return "", "", errors.New("Flow 正式版本的代码来源摘要不匹配")
	}
	if record.ExpectedCommit != "" && !pipelineCommitEqual(record.ExpectedCommit, flowManifestPrimaryCommit(manifest)) {
		return "", "", errors.New("Flow 正式版本的兼容提交身份不匹配")
	}
	var project model.AIProject
	if err := s.db.First(&project, pipeline.CodeProjectID).Error; err != nil {
		return "", "", fmt.Errorf("读取 Code 项目失败: %w", err)
	}
	resolvedManifest, err := resolvePipelineCodeSourceManifest(&project, manifest)
	if err != nil {
		return "", "", err
	}
	materializeCtx, cancel := context.WithTimeout(ctx, codeSourceMaterializeTimeout)
	defer cancel()
	logger.Info("Pipeline 正在从 Code 准备锁定源码: version=%s, source=%s", run.Version, run.SourceType)
	if err := resetCodePipelineWorkspace(workspace); err != nil {
		return "", "", err
	}
	workspaceReady := false
	defer func() {
		if !workspaceReady {
			_ = os.RemoveAll(workspace)
		}
	}()
	if err := materializeCodeSourceManifest(materializeCtx, logger, resolvedManifest, workspace); err != nil {
		if errors.Is(materializeCtx.Err(), context.DeadlineExceeded) {
			return "", "", fmt.Errorf("准备 Code 锁定源码超时: %w", materializeCtx.Err())
		}
		return "", "", err
	}
	fact := flowBuildFact{
		SchemaVersion: flowSourceManifestSchemaVersion, FlowRunID: run.ID, Version: run.Version,
		SourceType: run.SourceType, SourceDigest: run.SourceDigest, DeliveryJobID: run.CodeDeliveryJobID,
		SessionID: run.SessionID, TaskID: run.TaskID, TaskTitle: manifest.TaskTitle,
		Repositories: flowPublicSourceRepositories(manifest),
	}
	if err := writeFlowBuildFact(workspace, fact); err != nil {
		return "", "", err
	}
	workspaceReady = true
	logger.Info("Code 锁定源码已就绪: repositories=%d, digest=%s", len(manifest.Repositories), digest)
	return flowManifestPrimaryCommit(manifest), digest, nil
}

func resetCodePipelineWorkspace(workspace string) error {
	workspace = filepath.Clean(strings.TrimSpace(workspace))
	if workspace == "." || workspace == string(filepath.Separator) || filepath.Base(workspace) != "workspace" {
		return errors.New("Code 流水线工作区路径非法")
	}
	if err := os.RemoveAll(workspace); err != nil {
		return fmt.Errorf("重置 Code 流水线工作区失败: %w", err)
	}
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return fmt.Errorf("创建 Code 流水线工作区失败: %w", err)
	}
	return nil
}

func resolvePipelineCodeSourceManifest(project *model.AIProject, manifest flowSourceManifest) (flowSourceManifest, error) {
	if project == nil || len(project.SourceDirs) == 0 {
		return flowSourceManifest{}, errors.New("Code 项目没有源目录")
	}
	repositories, err := discoverFlowGitRepositories(project.SourceDirs)
	if err != nil {
		return flowSourceManifest{}, fmt.Errorf("解析 Code 项目仓库失败: %w", err)
	}
	byWorkspacePath := make(map[string]string, len(repositories))
	for _, repository := range repositories {
		_, workspacePath, resolveErr := flowRepositoryWorkspacePath(project.SourceDirs, repository)
		if resolveErr != nil {
			return flowSourceManifest{}, resolveErr
		}
		if _, exists := byWorkspacePath[workspacePath]; exists {
			return flowSourceManifest{}, fmt.Errorf("Code 项目仓库映射重复: %s", workspacePath)
		}
		byWorkspacePath[workspacePath] = repository
	}
	resolved := manifest
	resolved.Repositories = append([]flowSourceManifestRepository(nil), manifest.Repositories...)
	for index := range resolved.Repositories {
		repository := &resolved.Repositories[index]
		sourceDir, exists := byWorkspacePath[repository.WorkspacePath]
		if !exists {
			return flowSourceManifest{}, fmt.Errorf("Code 项目仓库映射已失效: %s", repository.WorkspacePath)
		}
		if !flowGitCommitExists(sourceDir, repository.Commit) {
			return flowSourceManifest{}, fmt.Errorf("Code 项目仓库提交不可用: %s", repository.Name)
		}
		repository.SourceDir = sourceDir
	}
	return resolved, nil
}

func materializeCodeSourceManifest(ctx context.Context, logger *PipelineLogger, manifest flowSourceManifest, destination string) error {
	repositories := append([]flowSourceManifestRepository(nil), manifest.Repositories...)
	sort.SliceStable(repositories, func(left, right int) bool {
		return flowWorkspacePathDepth(repositories[left].WorkspacePath) < flowWorkspacePathDepth(repositories[right].WorkspacePath)
	})
	for index, repository := range repositories {
		if err := ctx.Err(); err != nil {
			return err
		}
		if logger != nil {
			logger.Info("正在准备 Code 仓库 (%d/%d): %s @ %s", index+1, len(repositories), repository.Name, repository.Commit)
		}
		target, err := safeFlowWorkspaceTarget(destination, repository.WorkspacePath)
		if err != nil {
			return err
		}
		if repository.WorkspacePath != "." {
			if err := os.RemoveAll(target); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(target, 0755); err != nil {
			return err
		}
		repositoryCtx, cancel := context.WithTimeout(ctx, codeSourceRepositoryTimeout)
		err = extractFlowGitArchive(repositoryCtx, logger, repository.Name, repository.SourceDir, repository.Commit, target)
		cancel()
		if err != nil {
			if errors.Is(repositoryCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("准备 Code 仓库 %s 超时", repository.Name)
			}
			return fmt.Errorf("准备 Code 仓库 %s 失败: %w", repository.Name, err)
		}
		if logger != nil {
			logger.Info("Code 仓库已准备完成 (%d/%d): %s", index+1, len(repositories), repository.Name)
		}
	}
	return nil
}

func flowWorkspacePathDepth(path string) int {
	path = strings.Trim(filepath.ToSlash(filepath.Clean(path)), "/.")
	if path == "" {
		return 0
	}
	return len(strings.Split(path, "/"))
}

func safeFlowWorkspaceTarget(root, relative string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relative)))
	if clean == "" || clean == "." {
		return filepath.Clean(root), nil
	}
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("Flow 正式版本包含非法仓库路径")
	}
	target := filepath.Join(root, clean)
	if !codeSnapshotPathWithin(root, target) {
		return "", errors.New("Flow 正式版本仓库路径超出构建目录")
	}
	return target, nil
}

func extractFlowGitArchive(ctx context.Context, logger *PipelineLogger, repositoryName, repository, commit, destination string) error {
	archiveFile, err := os.CreateTemp("", "gopanel-code-archive-*.tar")
	if err != nil {
		return err
	}
	archivePath := archiveFile.Name()
	if err := archiveFile.Close(); err != nil {
		_ = os.Remove(archivePath)
		return err
	}
	defer os.Remove(archivePath)

	command := exec.CommandContext(ctx, "git", "archive", "--format=tar", "--output="+archivePath, commit)
	command.Dir = repository
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("git archive 失败: %s", strings.TrimSpace(stderr.String()))
	}
	if logger != nil {
		logger.Info("Code 仓库归档已生成，正在解包: %s", repositoryName)
	}
	archiveReader, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archiveReader.Close()
	return extractFlowTar(ctx, archiveReader, destination)
}

func extractFlowTar(ctx context.Context, reader io.Reader, destination string) error {
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
		target, err := safeFlowWorkspaceTarget(destination, header.Name)
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
			if filepath.IsAbs(header.Linkname) {
				return errors.New("Flow 正式版本包含绝对符号链接")
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(target), header.Linkname))
			if !codeSnapshotPathWithin(destination, resolved) {
				return errors.New("Flow 正式版本包含越界符号链接")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Flow 正式版本包含不支持的文件类型: %s", header.Name)
		}
	}
}

func writeFlowBuildFact(root string, fact flowBuildFact) error {
	content, err := json.MarshalIndent(fact, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(filepath.Join(root, flowBuildFactFileName), content, 0644)
}
