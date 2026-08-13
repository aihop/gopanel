package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"gorm.io/gorm"
)

func (s *PipelineService) RunPipeline(pipelineID uint, version, expectedCommit string) (uint, error) {
	return s.RunPipelineForSource(pipelineID, version, expectedCommit, PipelineRunSource{})
}

type PipelineRunSource struct {
	Type           string
	ID             uint
	IdempotencyKey string
	LogSummary     string
}

func (s *PipelineService) RunPipelineForSource(pipelineID uint, version, expectedCommit string, source PipelineRunSource) (uint, error) {
	pipelineMutationMu.Lock()
	defer pipelineMutationMu.Unlock()

	if source.IdempotencyKey != "" {
		existing, err := s.recordRepo.GetByIdempotencyKey(source.IdempotencyKey)
		if err == nil {
			return existing.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	pipeline, err := s.repo.Get(pipelineID)
	if err != nil {
		return 0, err
	}
	expectedCommit, err = normalizePipelineExpectedCommit(expectedCommit)
	if err != nil {
		return 0, err
	}
	if expectedCommit != "" && strings.TrimSpace(pipeline.RepoUrl) == "" && pipelineSourceType(pipeline) != "code" {
		return 0, buserr.New(constant.ErrPipelineExpectedCommitRepo)
	}
	record := &model.PipelineRecord{
		PipelineID: pipeline.ID, Status: "pending", Version: version, ExpectedCommit: expectedCommit,
		SourceType: source.Type, SourceID: source.ID, IdempotencyKey: source.IdempotencyKey,
	}
	err = s.recordRepo.Create(record)
	if err != nil {
		return 0, err
	}
	if pipeline.Version != version {
		pipeline.Version = version
		_ = s.repo.Update(pipeline)
	}
	logger := GetPipelineLogger(record.ID)
	if source.Type == "flow_run" {
		logger.Info("====== Flow #%d 正式交付开始 ======", source.ID)
		logger.Info("正式版本: %s", version)
		if strings.TrimSpace(source.LogSummary) != "" {
			logger.Info("%s", strings.TrimSpace(source.LogSummary))
		}
		logger.Info("流水线执行记录已创建: #%d", record.ID)
	}
	go s.executePipeline(pipeline, record)
	return record.ID, nil
}

func (s *PipelineService) executePipeline(p *model.Pipeline, record *model.PipelineRecord) {
	recordID := record.ID
	ctx, cancel := context.WithCancel(context.Background())
	pipelineCancels.Store(recordID, cancel)
	defer func() {
		cancel()
		pipelineCancels.Delete(recordID)
	}()

	executionLock := pipelineExecutionLock(p.ID)
	executionLock.Lock()
	defer executionLock.Unlock()

	logger := GetPipelineLogger(recordID)
	defer func() {
		if record.SourceType != "flow_run" {
			logger.Info("EOF")
			RemovePipelineLogger(recordID)
		}
	}()
	logger.Info("====== Pipeline #%d 执行开始 ======", recordID)
	logger.Info("应用: %s | 分支: %s", p.Name, p.Branch)
	if record.ExpectedCommit != "" {
		logger.Info("锁定构建提交: %s", record.ExpectedCommit)
	}
	workspaceDir := pipelineWorkspaceDir(p)
	releaseDir := pipelineReleaseDir(p)
	logger.Info("工作区目录: %s", workspaceDir)
	logger.Info("发布目录: %s", releaseDir)
	if pipelineSourceType(p) == "code" {
		s.recordRepo.UpdateStatus(recordID, "preparing", "")
		commitHash, sourceDigest, err := s.prepareCodePipelineSource(ctx, logger, p, record, workspaceDir)
		if err != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Code snapshot failed: %v", err))
			logger.Error("Code 项目快照失败: %v", err)
			return
		}
		_ = s.recordRepo.UpdateCodeSource(recordID, p.CodeProjectID, commitHash, sourceDigest)
		record.CodeProjectID = p.CodeProjectID
		record.CommitHash = commitHash
		record.SourceDigest = sourceDigest
	} else if p.RepoUrl != "" {
		if err := ensurePipelineClonePrerequisites(p.RepoUrl); err != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", err.Error())
			logger.Error("%v", err)
			return
		}
		s.recordRepo.UpdateStatus(recordID, "cloning", "")
		// 上次成功构建的 commit 作为更新说明的起点；取不到就退化成最近若干条
		sinceCommit := ""
		if last, err := s.recordRepo.LatestSuccessCommitHash(p.ID, recordID); err == nil {
			sinceCommit = last
		}
		commitHash, changelog, err := s.stepClone(ctx, logger, p, workspaceDir, sinceCommit, record.ExpectedCommit)
		if err != nil {
			if ctx.Err() != nil {
				s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
				logger.Error("流水线已手动取消")
			} else {
				s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Clone failed: %v", err))
			}
			return
		}
		if strings.TrimSpace(commitHash) != "" {
			_ = s.recordRepo.UpdateCommitHash(recordID, commitHash)
			record.CommitHash = commitHash
		}
		if strings.TrimSpace(changelog) != "" {
			_ = s.recordRepo.UpdateChangelog(recordID, changelog)
			record.Changelog = changelog
		}
	} else {
		logger.Info("未配置 RepoUrl，采用纯脚本模式，跳过自动拉取...")
		_ = os.MkdirAll(workspaceDir, 0755)
	}
	if pipelineSourceType(p) != "code" {
		if err := verifyPipelineExpectedCommit(ctx, workspaceDir, record.ExpectedCommit); err != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", err.Error())
			logger.Error("构建提交校验失败: %v", err)
			return
		}
	}
	if p.BuildImage == "host" || p.BuildImage == "" {
		files, _ := os.ReadDir(workspaceDir)
		var fileNames []string
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
		logger.Info("工作区目录检查 (%s): [%s]", workspaceDir, strings.Join(fileNames, ", "))
	}
	if pipelineShouldPrepareReleaseBeforeBuild(p) {
		if err := preparePipelineReleaseDir(logger, workspaceDir, releaseDir); err != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Prepare release failed: %v", err))
			logger.Error("准备发布目录失败: %v", err)
			return
		}
	} else {
		if err := resetPipelineReleaseSyncMarker(releaseDir); err != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Prepare release sync failed: %v", err))
			logger.Error("准备 Release 固化状态失败: %v", err)
			return
		}
		logger.Info("宿主机构建将在完成后一次性固化 Release，跳过构建前源码复制")
	}
	s.recordRepo.UpdateStatus(recordID, "building", "")
	logger.Info("开始构建版本...，版本号: %s", record.Version)
	// 这里要加判断，如果是 build_run，才执行构建命令

	_, err := s.stepBuild(ctx, logger, p, workspaceDir, releaseDir, record.Version)

	if err != nil {
		if ctx.Err() != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
			logger.Error("流水线已手动取消")
		} else {
			s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Build failed: %v", err))
		}
		return
	}
	if pipelineSourceType(p) != "code" {
		if err := verifyPipelineExpectedCommit(ctx, workspaceDir, record.ExpectedCommit); err != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", err.Error())
			logger.Error("构建后提交校验失败: %v", err)
			return
		}
	}
	artifactDir := releaseDir
	archivePath := ""
	runnerEnabled := strings.EqualFold(strings.TrimSpace(p.RunnerMode), "runner")
	if !runnerEnabled {
		archivePath, err = s.stepArchive(ctx, logger, p, artifactDir, recordID)
		if err != nil {
			if ctx.Err() != nil {
				s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
				logger.Error("流水线已手动取消")
				return
			}
			logger.Error("归档失败，但不影响发布: %v", err)
		} else {
			s.recordRepo.UpdateArchive(recordID, archivePath)
			record.ArchiveFile = archivePath
		}
	}
	s.recordRepo.UpdateStatus(recordID, "deploying", "准备执行 Runner 步骤...")
	if runnerEnabled {
		runnerHostPort, runnerContainerID, _, err := s.stepRunner(ctx, logger, p, releaseDir)
		if err != nil {
			if ctx.Err() != nil {
				s.recordRepo.UpdateStatus(recordID, "failed", "用户手动终止")
				logger.Error("流水线已手动取消")
			} else {
				s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Runner failed: %v", err))
				logger.Error("Runner 步骤失败: %v", err)
			}
			return
		}
		if runnerContainerID != "" {
			runnerReleaseDir, snapshotErr := snapshotPipelineRunnerArtifact(ctx, logger, p, recordID, runnerContainerID)
			if snapshotErr != nil {
				s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Runner artifact failed: %v", snapshotErr))
				logger.Error("Runner 构建结果固化失败: %v", snapshotErr)
				s.cleanupUnpublishedRunner(ctx, logger, p.ID, runnerContainerID)
				return
			}
			artifactDir = runnerReleaseDir
			record.RunnerReleaseDir = runnerReleaseDir
			record.RunnerContainerID = runnerContainerID
			record.RunnerHostPort = runnerHostPort
			logger.Info("Runner 容器已启动：containerId=%s, hostPort=%d", runnerContainerID, runnerHostPort)
			archivePath, err = s.stepArchive(ctx, logger, p, artifactDir, recordID)
			if err != nil {
				s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Runner archive failed: %v", err))
				logger.Error("Runner 正式构建结果归档失败: %v", err)
				s.cleanupUnpublishedRunner(ctx, logger, p.ID, runnerContainerID)
				return
			}
			if archivePath != "" {
				if updateErr := s.recordRepo.UpdateArchive(recordID, archivePath); updateErr != nil {
					s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Save Runner archive failed: %v", updateErr))
					logger.Error("保存 Runner 正式制品路径失败: %v", updateErr)
					s.cleanupUnpublishedRunner(ctx, logger, p.ID, runnerContainerID)
					return
				}
				record.ArchiveFile = archivePath
			}
		}
	} else {
		logger.Info("未启用 Runner 步骤，跳过...")
	}
	switch strings.TrimSpace(p.ActionType) {
	case "build_image", "build":
		imageArtifact, err := s.stepBuildImage(ctx, logger, p, artifactDir, recordID)
		if err != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("镜像构建失败: %v", err))
			logger.Error("镜像构建失败: %v", err)
			s.cleanupUnpublishedRunner(ctx, logger, p.ID, record.RunnerContainerID)
			return
		}
		if err := s.recordRepo.UpdateImageArtifact(recordID, imageArtifact.Tag, imageArtifact.ID, imageArtifact.Digest, imageArtifact.ImmutableRef); err != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("保存镜像制品身份失败: %v", err))
			logger.Error("保存镜像制品身份失败: %v", err)
			s.cleanupUnpublishedRunner(ctx, logger, p.ID, record.RunnerContainerID)
			return
		}
		record.ImageTag = imageArtifact.Tag
		record.ImageID = imageArtifact.ID
		record.ImageDigest = imageArtifact.Digest
		record.ImageRef = imageArtifact.ImmutableRef
		if !s.persistRunnerResult(ctx, logger, p, record) {
			return
		}
		s.recordRepo.UpdateStatus(recordID, "success", fmt.Sprintf("镜像构建成功: %s", imageArtifact.ImmutableRef))
		logger.Info("镜像构建成功: %s", imageArtifact.ImmutableRef)

	default:
		if !s.persistRunnerResult(ctx, logger, p, record) {
			return
		}
		s.recordRepo.UpdateStatus(recordID, "success", "构建成功（未配置后续操作）")
		logger.Info("流水线构建成功，网站发布请从容器列表选择端口绑定")
	}
	logger.Info("====== Pipeline #%d 执行成功！======", recordID)
}

func (s *PipelineService) persistRunnerResult(ctx context.Context, logger *PipelineLogger, pipeline *model.Pipeline, record *model.PipelineRecord) bool {
	if record == nil || strings.TrimSpace(record.RunnerContainerID) == "" {
		return true
	}
	if err := s.recordRepo.UpdateRunnerResult(record.ID, record.RunnerReleaseDir, record.RunnerContainerID, record.RunnerHostPort); err != nil {
		s.recordRepo.UpdateStatus(record.ID, "failed", fmt.Sprintf("Save Runner result failed: %v", err))
		logger.Error("保存 Runner 运行结果失败: %v", err)
		s.cleanupUnpublishedRunner(ctx, logger, pipeline.ID, record.RunnerContainerID)
		return false
	}
	return true
}

func pipelineShouldPrepareReleaseBeforeBuild(pipeline *model.Pipeline) bool {
	return pipeline == nil || pipeline.BuildImage != "host" || strings.TrimSpace(pipeline.BuildScript) == "" || pipeline.RunnerMode == constant.PipelineRunnerModeRunner
}

func ensurePipelineClonePrerequisites(repoURL string) error {
	repoURL = normalizePipelineRepoURL(repoURL)
	if repoURL == "" {
		return nil
	}
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("宿主机未安装 git，无法拉取仓库，请先安装 git")
	}
	if pipelineRepoUsesSSH(repoURL) {
		if _, err := exec.LookPath("ssh"); err != nil {
			return errors.New("当前仓库使用 SSH 地址，但宿主机未安装 ssh，无法拉取仓库，请先安装 openssh-client")
		}
	}
	return nil
}

func pipelineRepoUsesSSH(repoURL string) bool {
	repoURL = normalizePipelineRepoURL(repoURL)
	if repoURL == "" {
		return false
	}
	if strings.HasPrefix(repoURL, "ssh://") || strings.HasPrefix(repoURL, "git@") {
		return true
	}
	if strings.Contains(repoURL, "://") {
		return false
	}
	at := strings.Index(repoURL, "@")
	colon := strings.Index(repoURL, ":")
	slash := strings.Index(repoURL, "/")
	return at > 0 && colon > at && (slash == -1 || colon < slash)
}
