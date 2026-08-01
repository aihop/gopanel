package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

func (s *PipelineService) RunPipeline(pipelineID uint, version string) (uint, error) {
	pipeline, err := s.repo.Get(pipelineID)
	if err != nil {
		return 0, err
	}
	record := &model.PipelineRecord{PipelineID: pipeline.ID, Status: "pending", Version: version}
	err = s.recordRepo.Create(record)
	if err != nil {
		return 0, err
	}
	if pipeline.Version != version {
		pipeline.Version = version
		_ = s.repo.Update(pipeline)
	}
	go s.executePipeline(pipeline, record)
	return record.ID, nil
}

func (s *PipelineService) executePipeline(p *model.Pipeline, record *model.PipelineRecord) {
	recordID := record.ID
	logger := GetPipelineLogger(recordID)
	ctx, cancel := context.WithCancel(context.Background())
	pipelineCancels.Store(recordID, cancel)
	defer func() {
		pipelineCancels.Delete(recordID)
		logger.Info("EOF")
		RemovePipelineLogger(recordID)
	}()
	logger.Info("====== Pipeline #%d 执行开始 ======", recordID)
	logger.Info("应用: %s | 分支: %s", p.Name, p.Branch)
	workspaceDir := pipelineWorkspaceDir(p)
	releaseDir := pipelineReleaseDir(p)
	logger.Info("工作区目录: %s", workspaceDir)
	logger.Info("发布目录: %s", releaseDir)
	if p.RepoUrl != "" {
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
		commitHash, changelog, err := s.stepClone(ctx, logger, p, workspaceDir, sinceCommit)
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
	if p.BuildImage == "host" || p.BuildImage == "" {
		files, _ := os.ReadDir(workspaceDir)
		var fileNames []string
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
		logger.Info("工作区目录检查 (%s): [%s]", workspaceDir, strings.Join(fileNames, ", "))
	}
	if err := preparePipelineReleaseDir(logger, workspaceDir, releaseDir); err != nil {
		s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("Prepare release failed: %v", err))
		logger.Error("准备发布目录失败: %v", err)
		return
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
	archivePath, err := s.stepArchive(ctx, logger, p, releaseDir, recordID)
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
	s.recordRepo.UpdateStatus(recordID, "deploying", "准备执行 Runner 步骤...")
	if strings.EqualFold(strings.TrimSpace(p.RunnerMode), "runner") {
		runnerHostPort, runnerContainerID, runnerReleaseDir, err := s.stepRunner(ctx, logger, p, releaseDir)
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
			_ = s.recordRepo.UpdateRunnerResult(recordID, runnerReleaseDir, runnerContainerID, runnerHostPort)
			record.RunnerReleaseDir = runnerReleaseDir
			record.RunnerContainerID = runnerContainerID
			record.RunnerHostPort = runnerHostPort
			logger.Info("Runner 容器已启动：containerId=%s, hostPort=%d", runnerContainerID, runnerHostPort)
		}
	} else {
		logger.Info("未启用 Runner 步骤，跳过...")
	}
	switch strings.TrimSpace(p.ActionType) {
	case "build_image", "build":
		imageRef, err := s.stepBuildImage(ctx, logger, p, releaseDir, recordID)
		if err != nil {
			s.recordRepo.UpdateStatus(recordID, "failed", fmt.Sprintf("镜像构建失败: %v", err))
			logger.Error("镜像构建失败: %v", err)
			return
		}
		_ = s.recordRepo.UpdateImageTag(recordID, imageRef)
		s.recordRepo.UpdateStatus(recordID, "success", fmt.Sprintf("镜像构建成功: %s", imageRef))
		logger.Info("镜像构建成功: %s", imageRef)

	default:
		s.recordRepo.UpdateStatus(recordID, "success", "构建成功（未配置后续操作）")
		logger.Info("流水线构建成功，网站发布请从容器列表选择端口绑定")
	}
	logger.Info("====== Pipeline #%d 执行成功！======", recordID)
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
