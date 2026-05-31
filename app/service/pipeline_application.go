package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"gorm.io/gorm"
)

type PipelineApplicationService struct {
	pipelineRepo  *repo.PipelineRepo
	recordRepo    *repo.PipelineRecordRepo
	releaseRepo   *repo.ReleaseRepo
	appDeployRepo *repo.AppDeployRepo
	websiteRepo   *repo.WebsiteRepo
	executor      *PipelineService
}

func NewPipelineApplication(db *gorm.DB) *PipelineApplicationService {
	return &PipelineApplicationService{pipelineRepo: repo.NewPipeline(db), recordRepo: repo.NewPipelineRecord(db), releaseRepo: repo.NewRelease(db), appDeployRepo: repo.NewAppDeploy(db), websiteRepo: repo.NewWebsite(), executor: NewPipelineService(db)}
}
func (s *PipelineApplicationService) Page(ctx context.Context, page, limit int) (int64, []model.Pipeline, error) {
	total, list, err := s.pipelineRepo.Page(page, limit)
	if err != nil {
		return 0, nil, err
	}
	FillPipelineRuntimeMeta(ctx, list)
	return total, list, nil
}
func (s *PipelineApplicationService) Create(req request.PipelineCreate) error {
	pipelineKey := normalizePipelineKey(req.PipelineKey)
	if err := s.validatePipelineKey(pipelineKey, 0, ""); err != nil {
		return err
	}
	if req.RunnerMode == "runner" {
		if err := ValidateRunnerPersistentPaths(req.RunnerConfig); err != nil {
			return err
		}
	}
	pipeline := &model.Pipeline{Name: req.Name, Description: req.Description, RepoUrl: req.RepoUrl, Branch: req.Branch, Version: req.Version, AuthType: req.AuthType, AuthData: req.AuthData, ActionType: req.ActionType, ActionParams: req.ActionParams, AutoDeploy: req.AutoDeploy, BuildImage: req.BuildImage, BuildScript: req.BuildScript, OutputImage: req.OutputImage, ArtifactPath: req.ArtifactPath, ExposePort: req.ExposePort, PipelineKey: pipelineKey, RunnerMode: req.RunnerMode}
	if req.RunnerMode == "runner" && len(req.RunnerConfig) > 0 {
		if b, err := json.Marshal(req.RunnerConfig); err == nil {
			pipeline.RunnerConfig = string(b)
		}
	}
	return s.pipelineRepo.Create(pipeline)
}
func (s *PipelineApplicationService) Update(req request.PipelineUpdate) error {
	pipeline, err := s.pipelineRepo.Get(req.ID)
	if err != nil {
		return err
	}
	pipelineKey := normalizePipelineKey(req.PipelineKey)
	if err := s.validatePipelineKey(pipelineKey, pipeline.ID, pipeline.PipelineKey); err != nil {
		return err
	}
	if req.RunnerMode == "runner" {
		if err := ValidateRunnerPersistentPaths(req.RunnerConfig); err != nil {
			return err
		}
	}
	pipeline.Name = req.Name
	pipeline.Description = req.Description
	pipeline.RepoUrl = req.RepoUrl
	pipeline.Branch = req.Branch
	pipeline.Version = req.Version
	pipeline.AuthType = req.AuthType
	pipeline.AuthData = req.AuthData
	pipeline.AutoDeploy = req.AutoDeploy
	pipeline.ActionType = req.ActionType
	pipeline.ActionParams = req.ActionParams
	pipeline.BuildImage = req.BuildImage
	pipeline.BuildScript = req.BuildScript
	pipeline.OutputImage = req.OutputImage
	pipeline.ArtifactPath = req.ArtifactPath
	pipeline.ExposePort = req.ExposePort
	pipeline.PipelineKey = pipelineKey
	pipeline.RunnerMode = req.RunnerMode
	if req.RunnerMode != "runner" {
		pipeline.RunnerConfig = ""
	} else if req.RunnerConfig != nil {
		if len(req.RunnerConfig) == 0 {
			pipeline.RunnerConfig = ""
		} else if b, err := json.Marshal(req.RunnerConfig); err == nil {
			pipeline.RunnerConfig = string(b)
		}
	}
	return s.pipelineRepo.Update(pipeline)
}
func (s *PipelineApplicationService) Delete(id uint) error {
	websiteCount, err := s.websiteRepo.CountByPipelineID(id)
	if err != nil {
		return err
	}
	if websiteCount > 0 {
		return fmt.Errorf("该流水线已被网站绑定，不允许删除")
	}
	recordCount, err := s.recordRepo.CountByPipelineID(id)
	if err != nil {
		return err
	}
	if recordCount > 0 {
		return fmt.Errorf("该流水线已存在执行记录，不允许删除")
	}
	releaseCount, err := s.releaseRepo.CountByPipelineID(id)
	if err != nil {
		return err
	}
	if releaseCount > 0 {
		return fmt.Errorf("该流水线已存在正式版本，不允许删除")
	}
	return s.pipelineRepo.Delete(id)
}
