package repo

import (
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

type AppDeployRepo struct {
	db *gorm.DB
}

func NewAppDeploy(db *gorm.DB) *AppDeployRepo {
	return &AppDeployRepo{db: db}
}

func (r *AppDeployRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.AppDeploy{})
}

func (r *AppDeployRepo) SyncFromLegacy() error {
	var legacyList []model.LegacyWebsiteDeploy
	if err := r.db.Order("id asc").Find(&legacyList).Error; err != nil {
		return err
	}
	for _, legacy := range legacyList {
		var count int64
		if err := r.db.Model(&model.AppDeploy{}).Where("id = ?", legacy.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		item := model.AppDeploy{
			BaseModel:        legacy.BaseModel,
			WebsiteID:        legacy.WebsiteID,
			ReleaseID:        legacy.ReleaseID,
			PipelineRecordID: legacy.PipelineRecordID,
			Version:          legacy.Version,
			SourceType:       legacy.SourceType,
			SourceUrl:        legacy.SourceUrl,
			ArchiveFile:      legacy.ArchiveFile,
			ReleaseDir:       legacy.ReleaseDir,
			RuntimeDir:       legacy.RuntimeDir,
			ImageTag:         legacy.ImageTag,
			Status:           legacy.Status,
			LogText:          legacy.LogText,
			ContainerID:      legacy.ContainerID,
			Port:             legacy.Port,
			IsActive:         legacy.IsActive,
			DockerCompose:    legacy.DockerCompose,
			Env:              legacy.Env,
			AppInstallID:     legacy.AppInstallID,
		}
		if err := r.db.Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *AppDeployRepo) CountByPipelineRecordID(recordID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.AppDeploy{}).Where("pipeline_record_id = ?", recordID).Count(&count).Error
	return count, err
}

func (r *AppDeployRepo) ActiveWebsiteCountByPipelineRecordIDs(recordIDs []uint) (map[uint]int, error) {
	result := make(map[uint]int)
	if len(recordIDs) == 0 {
		return result, nil
	}

	type row struct {
		PipelineRecordID uint
		Count            int
	}
	var rows []row
	if err := r.db.Model(&model.AppDeploy{}).
		Select("pipeline_record_id, COUNT(*) AS count").
		Where("is_active = ? AND pipeline_record_id IN ?", true, recordIDs).
		Group("pipeline_record_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		if item.PipelineRecordID > 0 {
			result[item.PipelineRecordID] = item.Count
		}
	}
	return result, nil
}

func (r *AppDeployRepo) ActiveWebsiteCountByReleaseIDs(releaseIDs []uint) (map[uint]int, error) {
	result := make(map[uint]int)
	if len(releaseIDs) == 0 {
		return result, nil
	}

	type row struct {
		ReleaseID uint
		Count     int
	}
	var rows []row
	if err := r.db.Model(&model.AppDeploy{}).
		Select("release_id, COUNT(*) AS count").
		Where("is_active = ? AND release_id IN ?", true, releaseIDs).
		Group("release_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		if item.ReleaseID > 0 {
			result[item.ReleaseID] = item.Count
		}
	}
	return result, nil
}

func (r *AppDeployRepo) ActiveWebsiteNamesByPipelineRecordIDs(recordIDs []uint) (map[uint][]string, error) {
	return r.activeWebsiteNamesByTarget("pipeline_record_id", recordIDs)
}

func (r *AppDeployRepo) ActiveWebsiteNamesByReleaseIDs(releaseIDs []uint) (map[uint][]string, error) {
	return r.activeWebsiteNamesByTarget("release_id", releaseIDs)
}

func (r *AppDeployRepo) activeWebsiteNamesByTarget(column string, ids []uint) (map[uint][]string, error) {
	result := make(map[uint][]string)
	if len(ids) == 0 {
		return result, nil
	}

	type row struct {
		TargetID      uint
		PrimaryDomain string
		Alias         string
	}
	var rows []row
	query := fmt.Sprintf("app_deploys.%s AS target_id, website.primary_domain, website.alias", column)
	if err := r.db.Table("app_deploys").
		Select(query).
		Joins("LEFT JOIN website ON website.id = app_deploys.website_id").
		Where(fmt.Sprintf("app_deploys.is_active = ? AND app_deploys.%s IN ?", column), true, ids).
		Order("website.primary_domain asc, website.alias asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		if item.TargetID == 0 {
			continue
		}
		name := item.PrimaryDomain
		if name == "" {
			name = item.Alias
		}
		if name == "" {
			continue
		}
		result[item.TargetID] = append(result[item.TargetID], name)
	}
	return result, nil
}

func (r *AppDeployRepo) ActiveReleaseByWebsiteIDs(websiteIDs []uint) (map[uint]model.AppDeploy, error) {
	result := make(map[uint]model.AppDeploy)
	if len(websiteIDs) == 0 {
		return result, nil
	}
	var rows []model.AppDeploy
	if err := r.db.
		Where("website_id IN ? AND is_active = ? AND release_id > 0", websiteIDs, true).
		Order("id desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		if item.WebsiteID == 0 {
			continue
		}
		if _, exists := result[item.WebsiteID]; exists {
			continue
		}
		result[item.WebsiteID] = item
	}
	return result, nil
}

func (r *AppDeployRepo) LatestPipelineSyncByWebsiteIDs(websiteIDs []uint) (map[uint]model.AppDeploy, error) {
	result := make(map[uint]model.AppDeploy)
	if len(websiteIDs) == 0 {
		return result, nil
	}
	var rows []model.AppDeploy
	if err := r.db.
		Where("website_id IN ? AND source_type IN ?", websiteIDs, []string{"pipeline_sync", "pipeline"}).
		Order("id desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		if item.WebsiteID == 0 {
			continue
		}
		if _, exists := result[item.WebsiteID]; exists {
			continue
		}
		result[item.WebsiteID] = item
	}
	return result, nil
}
