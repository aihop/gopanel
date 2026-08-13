package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

func (s *PipelineApplicationService) Run(pipelineID uint, version, expectedCommit string) (uint, error) {
	return s.executor.RunPipeline(pipelineID, version, expectedCommit)
}
func (s *PipelineApplicationService) Stop(recordID uint) {
	StopPipeline(recordID)
}
func (s *PipelineApplicationService) RecordPage(ctx context.Context, pipelineID uint, page, limit int) (int64, []model.PipelineRecord, error) {
	total, list, err := s.recordRepo.PageByPipeline(pipelineID, page, limit)
	if err != nil {
		return 0, nil, err
	}
	FillPipelineRecordRuntimeMeta(ctx, list)
	if err := s.fillReleasedFlags(list); err != nil {
		return 0, nil, err
	}
	return total, list, nil
}
func (s *PipelineApplicationService) ReleasePage(pipelineID uint, page, limit int) (int64, []model.Release, error) {
	total, list, err := s.releaseRepo.PageByPipeline(pipelineID, page, limit)
	if err != nil {
		return 0, nil, err
	}
	return total, list, nil
}
func (s *PipelineApplicationService) ReleaseGet(id uint) (*model.Release, error) {
	return s.releaseRepo.Get(id)
}
func (s *PipelineApplicationService) DeleteRecord(recordID uint) error {
	record, err := s.recordRepo.Get(recordID)
	if err != nil {
		return err
	}
	if record.Status == "pending" || record.Status == "cloning" || record.Status == "building" || record.Status == "deploying" {
		return fmt.Errorf("执行中的记录不允许删除")
	}
	releaseCount, err := s.releaseRepo.CountByPipelineRecordID(recordID)
	if err != nil {
		return err
	}
	if releaseCount > 0 {
		return fmt.Errorf("该执行记录已生成正式版本，不允许删除")
	}
	return s.recordRepo.Delete(recordID)
}
func (s *PipelineApplicationService) PublishRecord(recordID uint) (*model.Release, error) {
	record, err := s.recordRepo.Get(recordID)
	if err != nil {
		return nil, err
	}
	if record.Status != "success" {
		return nil, fmt.Errorf("仅构建成功的记录可发布为正式版本")
	}
	existing, err := s.releaseRepo.GetByPipelineRecordID(recordID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	pipeline, err := s.pipelineRepo.Get(record.PipelineID)
	if err != nil {
		return nil, err
	}
	sourceType := "release_dir"
	imageTag := strings.TrimSpace(record.ImageTag)
	archiveFile := strings.TrimSpace(record.ArchiveFile)
	releaseDir := strings.TrimSpace(record.RunnerReleaseDir)
	if releaseDir == "" {
		releaseDir = pipelineReleaseDir(pipeline)
	}
	switch {
	case imageTag != "":
		sourceType = "image"
	case archiveFile != "":
		sourceType = "archive"
	default:
		releaseDir, err = snapshotPipelineReleaseDir(pipeline, record, releaseDir)
		if err != nil {
			return nil, err
		}
	}
	artifactMeta, err := json.Marshal(map[string]interface{}{"artifactPath": strings.TrimSpace(pipeline.ArtifactPath), "buildImage": strings.TrimSpace(pipeline.BuildImage), "pipelineKey": strings.TrimSpace(pipeline.PipelineKey), "runnerMode": strings.TrimSpace(pipeline.RunnerMode), "runnerHostPort": record.RunnerHostPort, "runnerContainerId": strings.TrimSpace(record.RunnerContainerID)})
	if err != nil {
		return nil, err
	}
	item := &model.Release{PipelineID: pipeline.ID, PipelineRecordID: record.ID, Version: strings.TrimSpace(record.Version), CommitHash: strings.TrimSpace(record.CommitHash), Changelog: strings.TrimSpace(record.Changelog), SourceType: sourceType, ImageTag: imageTag, ArchiveFile: archiveFile, ReleaseDir: releaseDir, ArtifactMeta: string(artifactMeta), Status: "ready"}
	if err := s.releaseRepo.Create(item); err != nil {
		if isReleasePipelineRecordDuplicate(err) {
			existing, findErr := s.releaseRepo.GetByPipelineRecordID(recordID)
			if findErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}
	return item, nil
}
