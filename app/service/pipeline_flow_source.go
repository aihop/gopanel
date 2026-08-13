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

	"github.com/aihop/gopanel/app/model"
)

const flowBuildFactFileName = "gopanel_flow.json"

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
		return s.prepareFlowCodeDeliverySource(ctx, logger, pipeline, record, workspace)
	}
	return s.prepareCodeProjectSnapshot(ctx, logger, pipeline, workspace, record.ExpectedCommit)
}

func (s *PipelineService) prepareFlowCodeDeliverySource(
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
	digest, err := flowSourceManifestDigest(manifest)
	if err != nil || digest != run.SourceDigest {
		return "", "", errors.New("Flow 正式版本的代码来源摘要不匹配")
	}
	if record.ExpectedCommit != "" && !pipelineCommitEqual(record.ExpectedCommit, flowManifestPrimaryCommit(manifest)) {
		return "", "", errors.New("Flow 正式版本的兼容提交身份不匹配")
	}
	baseDir := filepath.Dir(workspace)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", "", err
	}
	tempDir, err := os.MkdirTemp(baseDir, ".flow-source-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tempDir)
	logger.Info("正在物化 Flow 正式版本: version=%s, source=%s", run.Version, run.SourceType)
	if err := materializeFlowSourceManifest(ctx, manifest, tempDir); err != nil {
		return "", "", err
	}
	fact := flowBuildFact{
		SchemaVersion: flowSourceManifestSchemaVersion, FlowRunID: run.ID, Version: run.Version,
		SourceType: run.SourceType, SourceDigest: run.SourceDigest, DeliveryJobID: run.CodeDeliveryJobID,
		SessionID: run.SessionID, TaskID: run.TaskID, TaskTitle: manifest.TaskTitle,
		Repositories: flowPublicSourceRepositories(manifest),
	}
	if err := writeFlowBuildFact(tempDir, fact); err != nil {
		return "", "", err
	}
	if err := replacePipelineWorkspace(workspace, tempDir); err != nil {
		return "", "", err
	}
	logger.Info("Flow 正式版本源码已就绪: repositories=%d, digest=%s", len(manifest.Repositories), digest)
	return flowManifestPrimaryCommit(manifest), digest, nil
}

func materializeFlowSourceManifest(ctx context.Context, manifest flowSourceManifest, destination string) error {
	repositories := append([]flowSourceManifestRepository(nil), manifest.Repositories...)
	sort.SliceStable(repositories, func(left, right int) bool {
		return flowWorkspacePathDepth(repositories[left].WorkspacePath) < flowWorkspacePathDepth(repositories[right].WorkspacePath)
	})
	for _, repository := range repositories {
		if err := ctx.Err(); err != nil {
			return err
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
		if err := extractFlowGitArchive(ctx, repository.SourceDir, repository.Commit, target); err != nil {
			return fmt.Errorf("物化仓库 %s 失败: %w", repository.Name, err)
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

func extractFlowGitArchive(ctx context.Context, repository, commit, destination string) error {
	command := exec.CommandContext(ctx, "git", "archive", "--format=tar", commit)
	command.Dir = repository
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return err
	}
	extractErr := extractFlowTar(stdout, destination)
	waitErr := command.Wait()
	if extractErr != nil {
		return extractErr
	}
	if waitErr != nil {
		return fmt.Errorf("git archive 失败: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func extractFlowTar(reader io.Reader, destination string) error {
	archive := tar.NewReader(reader)
	for {
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
