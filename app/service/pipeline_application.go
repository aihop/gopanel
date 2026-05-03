package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	fileutil "github.com/aihop/gopanel/utils/files"
	"gorm.io/gorm"
)

type PipelineApplicationService struct {
	pipelineRepo *repo.PipelineRepo
	recordRepo   *repo.PipelineRecordRepo
	releaseRepo  *repo.ReleaseRepo
	appDeployRepo *repo.AppDeployRepo
	websiteRepo  *repo.WebsiteRepo
	executor     *PipelineService
}

func NewPipelineApplication(db *gorm.DB) *PipelineApplicationService {
	return &PipelineApplicationService{
		pipelineRepo:  repo.NewPipeline(db),
		recordRepo:    repo.NewPipelineRecord(db),
		releaseRepo:   repo.NewRelease(db),
		appDeployRepo: repo.NewAppDeploy(db),
		websiteRepo:   repo.NewWebsite(),
		executor:      NewPipelineService(db),
	}
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

	pipeline := &model.Pipeline{
		Name:         req.Name,
		Description:  req.Description,
		RepoUrl:      req.RepoUrl,
		Branch:       req.Branch,
		Version:      req.Version,
		AuthType:     req.AuthType,
		AuthData:     req.AuthData,
		BuildImage:   req.BuildImage,
		BuildScript:  req.BuildScript,
		OutputImage:  req.OutputImage,
		ArtifactPath: req.ArtifactPath,
		ExposePort:   req.ExposePort,
		PipelineKey:  pipelineKey,
		RunnerMode:   req.RunnerMode,
	}
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

func (s *PipelineApplicationService) Run(pipelineID uint, version string) (uint, error) {
	return s.executor.RunPipeline(pipelineID, version)
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
	if err := s.fillActiveDeployCountsForRecords(list); err != nil {
		return 0, nil, err
	}
	return total, list, nil
}

func (s *PipelineApplicationService) ReleasePage(pipelineID uint, page, limit int) (int64, []model.Release, error) {
	total, list, err := s.releaseRepo.PageByPipeline(pipelineID, page, limit)
	if err != nil {
		return 0, nil, err
	}
	if err := s.fillActiveDeployCountsForReleases(list); err != nil {
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
	deployCount, err := s.appDeployRepo.CountByPipelineRecordID(recordID)
	if err != nil {
		return err
	}
	if deployCount > 0 {
		return fmt.Errorf("该执行记录已有关联部署记录，不允许删除")
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

	artifactMeta, err := json.Marshal(map[string]interface{}{
		"artifactPath":      strings.TrimSpace(pipeline.ArtifactPath),
		"buildImage":        strings.TrimSpace(pipeline.BuildImage),
		"outputImage":       strings.TrimSpace(pipeline.OutputImage),
		"pipelineKey":       strings.TrimSpace(pipeline.PipelineKey),
		"runnerMode":        strings.TrimSpace(pipeline.RunnerMode),
		"runnerHostPort":    record.RunnerHostPort,
		"runnerContainerId": strings.TrimSpace(record.RunnerContainerID),
	})
	if err != nil {
		return nil, err
	}

	item := &model.Release{
		PipelineID:       pipeline.ID,
		PipelineRecordID: record.ID,
		Version:          strings.TrimSpace(record.Version),
		CommitHash:       strings.TrimSpace(record.CommitHash),
		SourceType:       sourceType,
		ImageTag:         imageTag,
		ArchiveFile:      archiveFile,
		ReleaseDir:       releaseDir,
		ArtifactMeta:     string(artifactMeta),
		Status:           "ready",
	}
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

func normalizePipelineKey(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func (s *PipelineApplicationService) validatePipelineKey(pipelineKey string, excludeID uint, currentPipelineKey string) error {
	if pipelineKey == "" {
		return errors.New("流水线标识不能为空")
	}
	exists, err := s.pipelineRepo.ExistsPipelineKey(pipelineKey, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("流水线标识 `%s` 已存在，请换一个", pipelineKey)
	}

	pipelineDir := filepath.Join(global.CONF.System.BaseDir, "pipelines", pipelineKey)
	if _, err := os.Stat(pipelineDir); err == nil && strings.TrimSpace(currentPipelineKey) != pipelineKey {
		return fmt.Errorf("流水线目录 `%s` 已存在，流水线标识重复了，请换其他的", pipelineDir)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	appDir := filepath.Join(global.CONF.System.BaseDir, "apps", pipelineKey)
	if _, err := os.Stat(appDir); err == nil && strings.TrimSpace(currentPipelineKey) != pipelineKey {
		return fmt.Errorf("安装目录 `%s` 已存在，流水线标识重复了，请换其他的", appDir)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *PipelineApplicationService) fillReleasedFlags(records []model.PipelineRecord) error {
	if len(records) == 0 {
		return nil
	}
	recordIDs := make([]uint, 0, len(records))
	for _, item := range records {
		if item.ID > 0 {
			recordIDs = append(recordIDs, item.ID)
		}
	}
	releasedMap, err := s.releaseRepo.ExistsByPipelineRecordIDs(recordIDs)
	if err != nil {
		return err
	}
	for i := range records {
		records[i].Released = releasedMap[records[i].ID]
	}
	return nil
}

func (s *PipelineApplicationService) fillActiveDeployCountsForRecords(records []model.PipelineRecord) error {
	if len(records) == 0 {
		return nil
	}
	recordIDs := make([]uint, 0, len(records))
	for _, item := range records {
		if item.ID > 0 {
			recordIDs = append(recordIDs, item.ID)
		}
	}
	countMap, err := s.appDeployRepo.ActiveWebsiteCountByPipelineRecordIDs(recordIDs)
	if err != nil {
		return err
	}
	nameMap, err := s.appDeployRepo.ActiveWebsiteNamesByPipelineRecordIDs(recordIDs)
	if err != nil {
		return err
	}
	for i := range records {
		records[i].ActiveWebsiteCount = countMap[records[i].ID]
		records[i].ActiveWebsiteNames = nameMap[records[i].ID]
	}
	return nil
}

func (s *PipelineApplicationService) fillActiveDeployCountsForReleases(releases []model.Release) error {
	if len(releases) == 0 {
		return nil
	}
	releaseIDs := make([]uint, 0, len(releases))
	for _, item := range releases {
		if item.ID > 0 {
			releaseIDs = append(releaseIDs, item.ID)
		}
	}
	countMap, err := s.appDeployRepo.ActiveWebsiteCountByReleaseIDs(releaseIDs)
	if err != nil {
		return err
	}
	nameMap, err := s.appDeployRepo.ActiveWebsiteNamesByReleaseIDs(releaseIDs)
	if err != nil {
		return err
	}
	for i := range releases {
		releases[i].ActiveWebsiteCount = countMap[releases[i].ID]
		releases[i].ActiveWebsiteNames = nameMap[releases[i].ID]
	}
	return nil
}

func snapshotPipelineReleaseDir(pipeline *model.Pipeline, record *model.PipelineRecord, src string) (string, error) {
	src = strings.TrimSpace(src)
	if pipeline == nil || record == nil {
		return "", fmt.Errorf("发布版本失败：缺少流水线或构建记录信息")
	}
	if src == "" {
		return "", fmt.Errorf("发布版本失败：缺少可固化的发布目录")
	}

	info, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("读取发布目录失败: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("发布目录不存在: %s", src)
	}

	snapshotDir := filepath.Join(pipelineArchiveDir(pipeline), fmt.Sprintf("release-record-%d", record.ID))
	if err := os.RemoveAll(snapshotDir); err != nil {
		return "", fmt.Errorf("清理历史版本快照失败: %w", err)
	}
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return "", fmt.Errorf("创建版本快照目录失败: %w", err)
	}
	if err := fileutil.CopyDirContents(src, snapshotDir); err != nil {
		return "", fmt.Errorf("固化版本目录失败: %w", err)
	}
	return snapshotDir, nil
}

func isReleasePipelineRecordDuplicate(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}
