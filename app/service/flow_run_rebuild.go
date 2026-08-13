package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"gorm.io/gorm"
)

func (s *FlowRunApplicationService) Rebuild(id, userID uint, includeAll bool) (*model.FlowRun, error) {
	source, err := s.repo.GetRunInternal(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, buserr.New(constant.ErrFlowNotFound)
	}
	if err != nil {
		return nil, err
	}
	flow, err := s.repo.Get(source.FlowID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, buserr.New(constant.ErrFlowNotFound)
	}
	if err != nil {
		return nil, err
	}
	if !includeAll && flow.CreatedBy != userID {
		return nil, buserr.New(constant.ErrFlowForbidden)
	}
	if !flow.Enabled {
		return nil, buserr.New(constant.ErrFlowDisabled)
	}
	if !flowRunCanRebuild(source.Status) {
		return nil, buserr.New(constant.ErrFlowRunRebuildUnsupported)
	}
	version, err := s.nextRebuildVersion(source.FlowID, source.Version)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	rebuilt := &model.FlowRun{
		FlowID: source.FlowID, ProjectID: source.ProjectID, PipelineID: source.PipelineID,
		Version: version, SourceRepository: source.SourceRepository, SourceType: source.SourceType,
		SourceBranch: source.SourceBranch, SourceCommit: source.SourceCommit, SourceDigest: source.SourceDigest,
		SourceManifest: source.SourceManifest, SessionID: source.SessionID, TaskID: source.TaskID,
		CodeDeliveryJobID: source.CodeDeliveryJobID, CurrentStage: "created", Status: flowRunQueued,
		RequestedBy: userID,
	}
	stage := &model.FlowStageRun{
		Stage: "created", Attempt: 1, Status: "success",
		Summary: fmt.Sprintf("rebuilt from flow run #%d", source.ID), StartedAt: &now, CompletedAt: &now,
	}
	if err := s.repo.CreateRun(rebuilt, stage); err != nil {
		if isFlowRunVersionDuplicate(err) {
			return nil, buserr.New(constant.ErrFlowVersionExists)
		}
		return nil, err
	}
	if err := s.retainRebuiltCodeSource(rebuilt); err != nil {
		_ = s.repo.DeleteRun(rebuilt.ID)
		return nil, err
	}
	if s.autoStart {
		go s.Advance(rebuilt.ID)
	}
	return rebuilt, nil
}

func (s *FlowRunApplicationService) nextRebuildVersion(flowID uint, sourceVersion string) (string, error) {
	version := nextFlowPatchVersion(sourceVersion)
	for attempt := 0; attempt < 100; attempt++ {
		exists, err := s.repo.VersionExists(flowID, version)
		if err != nil {
			return "", err
		}
		if !exists {
			return version, nil
		}
		version = nextFlowPatchVersion(version)
	}
	return "", buserr.New(constant.ErrFlowVersionExists)
}

func (s *FlowRunApplicationService) retainRebuiltCodeSource(run *model.FlowRun) error {
	if run == nil || !isFlowCodeSourceType(run.SourceType) {
		return nil
	}
	var manifest flowSourceManifest
	if json.Unmarshal([]byte(run.SourceManifest), &manifest) != nil || len(manifest.Repositories) == 0 {
		return buserr.New(constant.ErrFlowCodeSourceInvalid)
	}
	var project model.AIProject
	if err := s.db.First(&project, run.ProjectID).Error; err != nil {
		return buserr.New(constant.ErrFlowCodeSourceInvalid)
	}
	resolved, err := resolvePipelineCodeSourceManifest(&project, manifest)
	if err != nil {
		return buserr.New(constant.ErrFlowCodeSourceInvalid)
	}
	if err := retainFlowSourceCommits(run.ID, resolved); err != nil {
		return buserr.New(constant.ErrFlowCodeSourceInvalid)
	}
	return nil
}

func flowRunCanRebuild(status string) bool {
	status = strings.TrimSpace(status)
	return status == flowRunSuccess || status == flowRunWaitingDeployment
}
