package service

import (
	"context"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"os"
	"strings"
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
		s.recordRepo.UpdateStatus(recordID, "cloning", "")
		commitHash, err := s.stepClone(ctx, logger, p, workspaceDir)
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
	err := s.stepBuild(ctx, logger, p, workspaceDir, releaseDir, record.Version)
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
	s.recordRepo.UpdateStatus(recordID, "deploying", "同步关联网站的临时运行结果...")
	logger.Info("正在同步所有关联此流水线的网站临时运行结果...")
	finalImage := detectBuiltImageRef(p, record.Version, logger.GetLogs())
	if finalImage != "" {
		logger.Info("检测到本次真实构建镜像: %s", finalImage)
		_ = s.recordRepo.UpdateImageTag(recordID, finalImage)
		record.ImageTag = finalImage
	}
	summary, err := NewWebsite().DeployFromPipeline(ctx, p.ID, recordID, record.Version, archivePath, finalImage)
	if err != nil {
		s.recordRepo.UpdateStatus(recordID, "failed", err.Error())
		logger.Error("同步网站临时运行结果失败: %v", err)
		return
	}
	if summary != nil && summary.Matched == 0 {
		s.recordRepo.UpdateStatus(recordID, "success", "构建成功")
		logger.Info("流水线构建成功")
		return
	}
	msg := ""
	if summary != nil {
		msg = fmt.Sprintf("已完成 %d/%d 个网站临时结果同步", summary.Success, summary.Matched)
		logger.Info("%s", msg)
	}
	s.recordRepo.UpdateStatus(recordID, "success", msg)
	logger.Info("====== Pipeline #%d 执行成功！======", recordID)
}
