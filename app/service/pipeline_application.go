package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"gorm.io/gorm"
)

type PipelineApplicationService struct {
	db           *gorm.DB
	pipelineRepo *repo.PipelineRepo
	recordRepo   *repo.PipelineRecordRepo
	releaseRepo  *repo.ReleaseRepo
	executor     *PipelineService
}

func NewPipelineApplication(db *gorm.DB) *PipelineApplicationService {
	return &PipelineApplicationService{db: db, pipelineRepo: repo.NewPipeline(db), recordRepo: repo.NewPipelineRecord(db), releaseRepo: repo.NewRelease(db), executor: NewPipelineService(db)}
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
	req.ActionType = normalizePipelineActionType(req.ActionType)
	sourceType, codeProjectID, err := s.normalizePipelineSource(req.SourceType, req.CodeProjectID)
	if err != nil {
		return err
	}
	pipelineKey := normalizePipelineKey(req.PipelineKey)
	if err := s.validatePipelineKey(pipelineKey, 0, ""); err != nil {
		return err
	}
	if req.RunnerMode == "runner" {
		if err := ValidateRunnerPersistentPaths(req.RunnerConfig); err != nil {
			return err
		}
	}
	pipeline := &model.Pipeline{Name: req.Name, Description: req.Description, SourceType: sourceType, CodeProjectID: codeProjectID, RepoUrl: req.RepoUrl, Branch: req.Branch, Version: req.Version, AuthType: req.AuthType, AuthData: req.AuthData, ActionType: req.ActionType, BuildImage: req.BuildImage, BuildScript: req.BuildScript, ArtifactPath: req.ArtifactPath, PipelineKey: pipelineKey, RunnerMode: req.RunnerMode}
	clearUnusedPipelineSource(pipeline)
	if req.RunnerMode == "runner" && len(req.RunnerConfig) > 0 {
		if b, err := json.Marshal(req.RunnerConfig); err == nil {
			pipeline.RunnerConfig = string(b)
		}
	}

	if len(req.ActionParams) > 0 {
		if b, err := json.Marshal(req.ActionParams); err == nil {
			pipeline.ActionParams = string(b)
		}
	}

	return s.pipelineRepo.Create(pipeline)
}
func (s *PipelineApplicationService) Update(req request.PipelineUpdate) error {
	req.ActionType = normalizePipelineActionType(req.ActionType)
	pipeline, err := s.pipelineRepo.Get(req.ID)
	if err != nil {
		return err
	}
	sourceType, codeProjectID, err := s.normalizePipelineSource(req.SourceType, req.CodeProjectID)
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
	pipeline.SourceType = sourceType
	pipeline.CodeProjectID = codeProjectID
	pipeline.RepoUrl = req.RepoUrl
	pipeline.Branch = req.Branch
	pipeline.Version = req.Version
	pipeline.AuthType = req.AuthType
	pipeline.AuthData = req.AuthData
	clearUnusedPipelineSource(pipeline)
	pipeline.ActionType = req.ActionType

	if len(req.ActionParams) > 0 {
		if b, err := json.Marshal(req.ActionParams); err == nil {
			pipeline.ActionParams = string(b)
		}
	}

	pipeline.BuildImage = req.BuildImage
	pipeline.BuildScript = req.BuildScript
	pipeline.ArtifactPath = req.ArtifactPath
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

func (s *PipelineApplicationService) normalizePipelineSource(sourceType string, codeProjectID uint) (string, uint, error) {
	sourceType = strings.ToLower(strings.TrimSpace(sourceType))
	if sourceType == "" {
		sourceType = "git"
	}
	if sourceType != "git" && sourceType != "code" {
		return "", 0, buserr.New(constant.ErrPipelineSourceInvalid)
	}
	if sourceType == "git" {
		return sourceType, 0, nil
	}
	if codeProjectID == 0 {
		return "", 0, buserr.New(constant.ErrPipelineCodeProjectRequired)
	}
	var count int64
	if err := s.db.Model(&model.AIProject{}).Where("id = ?", codeProjectID).Count(&count).Error; err != nil {
		return "", 0, err
	}
	if count == 0 {
		return "", 0, buserr.New(constant.ErrPipelineCodeProjectNotFound)
	}
	return sourceType, codeProjectID, nil
}

func clearUnusedPipelineSource(pipeline *model.Pipeline) {
	if pipeline == nil || pipeline.SourceType != "code" {
		return
	}
	pipeline.RepoUrl = ""
	pipeline.AuthType = "none"
	pipeline.AuthData = ""
}

func normalizePipelineActionType(value string) string {
	switch value {
	case "build", "build_image":
		return "build_image"
	default:
		return "none"
	}
}
func (s *PipelineApplicationService) Delete(id uint) error {
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
