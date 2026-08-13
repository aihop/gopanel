package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func (s *FlowRunApplicationService) Advance(runID uint) {
	owner := flowWorkerOwner(runID)
	for {
		now := time.Now()
		claimed, err := s.repo.ClaimRun(runID, owner, now, now.Add(30*time.Second))
		if err != nil || !claimed {
			return
		}
		run, err := s.repo.GetRunInternal(runID)
		if err != nil || (run.Status != flowRunQueued && run.Status != flowRunRunning) {
			return
		}
		if run.CurrentStage == "publishing" && run.PipelineRecordID > 0 {
			s.resumeFlowPublish(run)
			return
		}
		if run.CurrentStage == "deploying" && run.PipelineRecordID > 0 && run.ReleaseID > 0 {
			s.resumeFlowDeploy(run)
			return
		}
		if run.PipelineRecordID == 0 {
			if !s.startFlowBuild(run) {
				return
			}
			continue
		}
		record, err := s.recordRepo.Get(run.PipelineRecordID)
		if err != nil {
			s.failStage(run, "building", "pipeline_record_unavailable", err.Error())
			return
		}
		if (record.Status == "success" || record.Status == "failed") && IsPipelineExecutionActive(record.ID) {
			time.Sleep(s.pollInterval)
			continue
		}
		switch record.Status {
		case "success":
			s.finishFlowBuild(run, record)
			return
		case "failed":
			s.failStage(run, "building", "pipeline_failed", record.ErrorMessage)
			return
		default:
			if !IsPipelineExecutionActive(record.ID) && record.UpdatedAt.Before(time.Now().Add(-2*time.Minute)) {
				message := "pipeline execution was interrupted before completion"
				_ = s.recordRepo.UpdateStatus(record.ID, "failed", message)
				s.failStage(run, "building", "pipeline_interrupted", message)
				return
			}
			time.Sleep(s.pollInterval)
		}
	}
}

func (s *FlowRunApplicationService) startFlowBuild(run *model.FlowRun) bool {
	attempt, err := s.repo.CurrentStageAttempt(run.ID, "building")
	if err != nil {
		s.failStage(run, "building", "flow_attempt_unavailable", err.Error())
		return false
	}
	now := time.Now()
	_ = s.repo.UpdateRun(run.ID, map[string]any{
		"status": flowRunRunning, "current_stage": "building", "started_at": now,
		"failure_code": "", "error_summary": "",
	})
	stage := &model.FlowStageRun{
		FlowRunID: run.ID, Stage: "building", Attempt: attempt, Status: "running",
		IdempotencyKey: fmt.Sprintf("flow:%d:build:%d", run.ID, attempt),
		ResourceType:   "pipeline_record", Summary: "pipeline build started", StartedAt: &now,
	}
	_ = s.repo.UpsertStage(stage)
	recordID, err := s.runPipeline(run.PipelineID, run.Version, run.SourceCommit, PipelineRunSource{
		Type: "flow_run", ID: run.ID, IdempotencyKey: stage.IdempotencyKey,
		LogSummary: fmt.Sprintf("来源类型: %s | 来源摘要: %s", run.SourceType, run.SourceDigest),
	})
	if err != nil {
		s.failStage(run, "building", "pipeline_start_failed", err.Error())
		return false
	}
	run.PipelineRecordID = recordID
	stage.ResourceID = recordID
	_ = s.repo.UpsertStage(stage)
	if err := s.repo.UpdateRun(run.ID, map[string]any{"pipeline_record_id": recordID}); err != nil {
		s.failStage(run, "building", "pipeline_record_link_failed", err.Error())
		return false
	}
	return true
}

func (s *FlowRunApplicationService) resumeFlowPublish(run *model.FlowRun) {
	record, err := s.recordRepo.Get(run.PipelineRecordID)
	if err != nil || record.Status != "success" {
		detail := "pipeline record is not available for publication"
		if err != nil {
			detail = err.Error()
		}
		s.failStage(run, "publishing", "pipeline_record_unavailable", detail)
		return
	}
	s.publishFlowRelease(run, record)
}

func (s *FlowRunApplicationService) finishFlowBuild(run *model.FlowRun, record *model.PipelineRecord) {
	logger := GetPipelineLogger(record.ID)
	logger.Info("Pipeline 构建完成，正在校验 Flow 正式版本身份...")
	if record.Version != run.Version || !pipelineCommitEqual(record.ExpectedCommit, run.SourceCommit) {
		s.failStage(run, "building", "pipeline_identity_mismatch", "pipeline record does not match the locked flow version and commit")
		return
	}
	if isFlowCodeSourceType(run.SourceType) && strings.TrimSpace(record.SourceDigest) != strings.TrimSpace(run.SourceDigest) {
		s.failStage(run, "building", "pipeline_source_mismatch", "pipeline record does not match the locked Code delivery source")
		return
	}
	logger.Info("Flow 正式版本身份校验通过")
	now := time.Now()
	buildAttempt, err := s.repo.CurrentStageAttempt(run.ID, "building")
	if err != nil {
		s.failStage(run, "building", "flow_attempt_unavailable", err.Error())
		return
	}
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "building", Attempt: buildAttempt, Status: "success",
		IdempotencyKey: fmt.Sprintf("flow:%d:build:%d", run.ID, buildAttempt),
		ResourceType:   "pipeline_record", ResourceID: record.ID,
		Summary: "pipeline build completed", CompletedAt: &now,
	})
	s.publishFlowRelease(run, record)
}

func (s *FlowRunApplicationService) publishFlowRelease(run *model.FlowRun, record *model.PipelineRecord) {
	logger := GetPipelineLogger(record.ID)
	now := time.Now()
	_ = s.repo.UpdateRun(run.ID, map[string]any{"current_stage": "publishing"})
	logger.Info("正在发布不可变 Release...")
	publishAttempt, err := s.repo.StageAttemptForExecution(run.ID, "publishing")
	if err != nil {
		s.failStage(run, "publishing", "flow_attempt_unavailable", err.Error())
		return
	}
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "publishing", Attempt: publishAttempt, Status: "running",
		IdempotencyKey: fmt.Sprintf("flow:%d:publish:%d", run.ID, publishAttempt),
		ResourceType:   "pipeline_record", ResourceID: record.ID,
		Summary: "release publication started", StartedAt: &now,
	})
	release, err := s.publishRecord(record.ID)
	if err != nil {
		s.failStage(run, "publishing", "release_publish_failed", err.Error())
		return
	}
	logger.Info("Release #%d 发布完成，制品摘要: %s", release.ID, release.ArtifactDigest)
	completedAt := time.Now()
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "publishing", Attempt: publishAttempt, Status: "success",
		IdempotencyKey: fmt.Sprintf("flow:%d:publish:%d", run.ID, publishAttempt),
		ResourceType:   "release", ResourceID: release.ID,
		Summary: "release published", CompletedAt: &completedAt,
	})
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "release_ready", Attempt: 1, Status: "success",
		IdempotencyKey: fmt.Sprintf("flow:%d:release-ready:1", run.ID),
		ResourceType:   "release", ResourceID: release.ID,
		Summary: "immutable release is ready", StartedAt: &completedAt, CompletedAt: &completedAt,
	})
	run.ReleaseID = release.ID
	if err := s.repo.UpdateRun(run.ID, map[string]any{"release_id": release.ID}); err != nil {
		s.failStage(run, "publishing", "release_link_failed", err.Error())
		return
	}
	s.deployFlowRelease(run, record)
}

func (s *FlowRunApplicationService) resumeFlowDeploy(run *model.FlowRun) {
	record, err := s.recordRepo.Get(run.PipelineRecordID)
	if err != nil || record.Status != "success" {
		detail := "pipeline record is not available for deployment"
		if err != nil {
			detail = err.Error()
		}
		s.failStage(run, "deploying", "pipeline_record_unavailable", detail)
		return
	}
	s.deployFlowRelease(run, record)
}

func (s *FlowRunApplicationService) deployFlowRelease(run *model.FlowRun, record *model.PipelineRecord) {
	logger := GetPipelineLogger(record.ID)
	var flow model.Flow
	if err := s.db.Preload("Environments", "enabled = ?", true).First(&flow, run.FlowID).Error; err != nil {
		s.failStage(run, "deploying", "flow_environment_unavailable", err.Error())
		return
	}
	automatic := make([]model.FlowEnvironment, 0, len(flow.Environments))
	for _, environment := range flow.Environments {
		if environment.AutoDeploy && !environment.ApprovalRequired {
			automatic = append(automatic, environment)
		}
	}
	if len(automatic) == 0 {
		_ = s.repo.UpsertStage(&model.FlowStageRun{
			FlowRunID: run.ID, Stage: "waiting_deployment", Attempt: 1, Status: "pending",
			IdempotencyKey: fmt.Sprintf("flow:%d:deployment:1", run.ID), ResourceType: "release", ResourceID: run.ReleaseID,
			Summary: "waiting for deployment approval or manual orchestration",
		})
		_ = s.repo.UpdateRun(run.ID, map[string]any{
			"release_id": run.ReleaseID, "current_stage": "waiting_deployment", "status": flowRunWaitingDeployment,
			"lease_owner": "", "lease_expires_at": nil,
		})
		logger.Info("Flow #%d 未配置无需审批的自动部署环境，保留 Release 并等待后续部署", run.ID)
		finishFlowRunLogger(record.ID)
		return
	}
	if record.RunnerHostPort <= 0 || strings.TrimSpace(record.RunnerContainerID) == "" {
		s.failStage(run, "deploying", "runner_target_unavailable", "pipeline did not produce a ready Runner container target")
		return
	}
	attempt, err := s.repo.StageAttemptForExecution(run.ID, "deploying")
	if err != nil {
		s.failStage(run, "deploying", "flow_attempt_unavailable", err.Error())
		return
	}
	startedAt := time.Now()
	_ = s.repo.UpdateRun(run.ID, map[string]any{"release_id": run.ReleaseID, "current_stage": "deploying", "status": flowRunRunning})
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "deploying", Attempt: attempt, Status: "running",
		IdempotencyKey: fmt.Sprintf("flow:%d:deploy:%d", run.ID, attempt), ResourceType: "release", ResourceID: run.ReleaseID,
		Summary: "automatic website deployment started", StartedAt: &startedAt,
	})
	for _, environment := range automatic {
		logger.Info("正在自动部署 Flow 环境: %s -> website #%d, port=%d", environment.Name, environment.WebsiteID, record.RunnerHostPort)
		if err := s.deployRunner(context.Background(), environment, record, run.Version); err != nil {
			s.failStage(run, "deploying", "website_deploy_failed", fmt.Sprintf("%s: %v", environment.Name, err))
			return
		}
		logger.Info("Flow 环境自动部署完成: %s -> website #%d", environment.Name, environment.WebsiteID)
	}
	completedAt := time.Now()
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "deploying", Attempt: attempt, Status: "success",
		IdempotencyKey: fmt.Sprintf("flow:%d:deploy:%d", run.ID, attempt), ResourceType: "release", ResourceID: run.ReleaseID,
		Summary: "automatic website deployment completed", CompletedAt: &completedAt,
	})
	_ = s.repo.UpdateRun(run.ID, map[string]any{
		"current_stage": "deployed", "status": flowRunSuccess, "completed_at": completedAt,
		"lease_owner": "", "lease_expires_at": nil,
	})
	logger.Info("Flow #%d 已完成构建、发布与自动部署", run.ID)
	finishFlowRunLogger(record.ID)
}

func deployFlowRunnerEnvironment(ctx context.Context, environment model.FlowEnvironment, record *model.PipelineRecord, version string) error {
	if record == nil {
		return fmt.Errorf("pipeline record is nil")
	}
	target := containerWebsiteTarget{
		ContainerID: strings.TrimSpace(record.RunnerContainerID), WebsiteID: environment.WebsiteID,
		HostPort: record.RunnerHostPort, Scheme: "http", Address: fmt.Sprintf("127.0.0.1:%d", record.RunnerHostPort),
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := checkContainerWebsiteEndpoint(checkCtx, target.Scheme, target.Address); err != nil {
		return err
	}
	var website model.Website
	if err := global.DB.Select("container_id").First(&website, environment.WebsiteID).Error; err != nil {
		return err
	}
	previousContainerID := strings.TrimSpace(website.ContainerID)
	if err := bindContainerTargetToWebsite(ctx, target, strings.TrimSpace(version)); err != nil {
		return err
	}
	cleanupPreviousWebsiteContainer(previousContainerID, target.ContainerID)
	return nil
}

func (s *FlowRunApplicationService) failStage(run *model.FlowRun, stage, code, detail string) {
	now := time.Now()
	detail = strings.TrimSpace(detail)
	attempt, err := s.repo.CurrentStageAttempt(run.ID, stage)
	if err != nil {
		attempt = 1
	}
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: stage, Attempt: attempt, Status: "failed",
		IdempotencyKey: fmt.Sprintf("flow:%d:%s:%d", run.ID, stage, attempt),
		ErrorCode:      code, ErrorDetail: detail, CompletedAt: &now,
	})
	s.failRun(run, code, detail)
}

func (s *FlowRunApplicationService) failRun(run *model.FlowRun, code, detail string) {
	now := time.Now()
	_ = s.repo.UpdateRun(run.ID, map[string]any{
		"current_stage": "failed", "status": flowRunFailed, "failure_code": code,
		"error_summary": strings.TrimSpace(detail), "completed_at": now,
		"lease_owner": "", "lease_expires_at": nil,
	})
	if run.PipelineRecordID > 0 {
		logger := GetPipelineLogger(run.PipelineRecordID)
		logger.Error("Flow #%d 执行失败 [%s]: %s", run.ID, code, strings.TrimSpace(detail))
		finishFlowRunLogger(run.PipelineRecordID)
	}
}

func finishFlowRunLogger(recordID uint) {
	if recordID == 0 {
		return
	}
	logger := GetPipelineLogger(recordID)
	logger.Info("====== Flow 执行结束 ======")
	RemovePipelineLogger(recordID)
}

func ReconcileFlowRuns() {
	if global.DB == nil {
		return
	}
	service := NewFlowRunApplication(global.DB)
	ids, err := service.repo.ActiveRunIDs()
	if err != nil {
		global.LOG.Errorf("[Flow] 查询待恢复交付失败: %v", err)
		return
	}
	for _, id := range ids {
		go service.Advance(id)
	}
}
