package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
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
	if pipelineRecordRunning(record.Status) {
		return fmt.Errorf("执行中的记录不允许删除")
	}
	if record.SourceType == "flow_run" {
		return buserr.New(constant.ErrFlowPipelineRecordProtected)
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
	artifact, err := buildPipelineReleaseArtifact(context.Background(), pipeline, record)
	if err != nil {
		return nil, err
	}
	item := artifact.release(pipeline, record)
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
