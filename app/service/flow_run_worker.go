package service

import (
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
		if run.PipelineRecordID == 0 {
			if !s.startFlowBuild(run) {
				return
			}
			continue
		}
		record, err := s.recordRepo.Get(run.PipelineRecordID)
		if err != nil {
			s.failRun(run, "pipeline_record_unavailable", err.Error())
			return
		}
		switch record.Status {
		case "success":
			s.finishFlowBuild(run, record)
			return
		case "failed":
			s.failStage(run, "building", "pipeline_failed", record.ErrorMessage)
			return
		default:
			if !IsPipelineLoggerActive(record.ID) && record.UpdatedAt.Before(time.Now().Add(-2*time.Minute)) {
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
	now := time.Now()
	_ = s.repo.UpdateRun(run.ID, map[string]any{
		"status": flowRunRunning, "current_stage": "building", "started_at": now,
		"failure_code": "", "error_summary": "",
	})
	stage := &model.FlowStageRun{
		FlowRunID: run.ID, Stage: "building", Attempt: 1, Status: "running",
		IdempotencyKey: fmt.Sprintf("flow:%d:build:1", run.ID),
		ResourceType:   "pipeline_record", Summary: "pipeline build started", StartedAt: &now,
	}
	_ = s.repo.UpsertStage(stage)
	recordID, err := s.runPipeline(run.PipelineID, run.Version, run.SourceCommit, PipelineRunSource{
		Type: "flow_run", ID: run.ID, IdempotencyKey: stage.IdempotencyKey,
	})
	if err != nil {
		s.failStage(run, "building", "pipeline_start_failed", err.Error())
		return false
	}
	stage.ResourceID = recordID
	_ = s.repo.UpsertStage(stage)
	if err := s.repo.UpdateRun(run.ID, map[string]any{"pipeline_record_id": recordID}); err != nil {
		s.failStage(run, "building", "pipeline_record_link_failed", err.Error())
		return false
	}
	return true
}

func (s *FlowRunApplicationService) finishFlowBuild(run *model.FlowRun, record *model.PipelineRecord) {
	if record.Version != run.Version || !pipelineCommitEqual(record.ExpectedCommit, run.SourceCommit) {
		s.failStage(run, "building", "pipeline_identity_mismatch", "pipeline record does not match the locked flow version and commit")
		return
	}
	if run.SourceType == "code_delivery" && strings.TrimSpace(record.SourceDigest) != strings.TrimSpace(run.SourceDigest) {
		s.failStage(run, "building", "pipeline_source_mismatch", "pipeline record does not match the locked Code delivery source")
		return
	}
	now := time.Now()
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "building", Attempt: 1, Status: "success",
		IdempotencyKey: fmt.Sprintf("flow:%d:build:1", run.ID),
		ResourceType:   "pipeline_record", ResourceID: record.ID,
		Summary: "pipeline build completed", CompletedAt: &now,
	})
	_ = s.repo.UpdateRun(run.ID, map[string]any{"current_stage": "publishing"})
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "publishing", Attempt: 1, Status: "running",
		IdempotencyKey: fmt.Sprintf("flow:%d:publish:1", run.ID),
		ResourceType:   "pipeline_record", ResourceID: record.ID,
		Summary: "release publication started", StartedAt: &now,
	})
	release, err := s.publishRecord(record.ID)
	if err != nil {
		s.failStage(run, "publishing", "release_publish_failed", err.Error())
		return
	}
	completedAt := time.Now()
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "publishing", Attempt: 1, Status: "success",
		IdempotencyKey: fmt.Sprintf("flow:%d:publish:1", run.ID),
		ResourceType:   "release", ResourceID: release.ID,
		Summary: "release published", CompletedAt: &completedAt,
	})
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "release_ready", Attempt: 1, Status: "success",
		IdempotencyKey: fmt.Sprintf("flow:%d:release-ready:1", run.ID),
		ResourceType:   "release", ResourceID: release.ID,
		Summary: "immutable release is ready", StartedAt: &completedAt, CompletedAt: &completedAt,
	})
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: "waiting_deployment", Attempt: 1, Status: "pending",
		IdempotencyKey: fmt.Sprintf("flow:%d:deployment:1", run.ID),
		ResourceType:   "release", ResourceID: release.ID,
		Summary: "waiting for deployment orchestration",
	})
	_ = s.repo.UpdateRun(run.ID, map[string]any{
		"release_id": release.ID, "current_stage": "waiting_deployment",
		"status": flowRunWaitingDeployment, "lease_owner": "", "lease_expires_at": nil,
	})
}

func (s *FlowRunApplicationService) failStage(run *model.FlowRun, stage, code, detail string) {
	now := time.Now()
	detail = strings.TrimSpace(detail)
	_ = s.repo.UpsertStage(&model.FlowStageRun{
		FlowRunID: run.ID, Stage: stage, Attempt: 1, Status: "failed",
		IdempotencyKey: fmt.Sprintf("flow:%d:%s:1", run.ID, stage),
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
