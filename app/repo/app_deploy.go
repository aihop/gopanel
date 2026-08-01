package repo

import (
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
	if err := r.db.AutoMigrate(&model.AppDeploy{}); err != nil {
		return err
	}
	if err := r.db.Where("source_type IN ?", []string{"pipeline", "pipeline_sync", "release"}).Delete(&model.AppDeploy{}).Error; err != nil {
		return err
	}
	for _, column := range []string{"release_id", "pipeline_record_id"} {
		if r.db.Migrator().HasColumn(&model.AppDeploy{}, column) {
			if err := r.db.Migrator().DropColumn(&model.AppDeploy{}, column); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *AppDeployRepo) SyncFromLegacy() error {
	if !r.db.Migrator().HasTable(&model.LegacyWebsiteDeploy{}) {
		return nil
	}
	var legacyList []model.LegacyWebsiteDeploy
	if err := r.db.Order("id asc").Find(&legacyList).Error; err != nil {
		return err
	}
	for _, legacy := range legacyList {
		switch legacy.SourceType {
		case "pipeline", "pipeline_sync", "release":
			continue
		}
		var count int64
		if err := r.db.Model(&model.AppDeploy{}).Where("id = ?", legacy.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		item := model.AppDeploy{
			BaseModel:     legacy.BaseModel,
			WebsiteID:     legacy.WebsiteID,
			Version:       legacy.Version,
			SourceType:    legacy.SourceType,
			SourceUrl:     legacy.SourceUrl,
			ArchiveFile:   legacy.ArchiveFile,
			ReleaseDir:    legacy.ReleaseDir,
			RuntimeDir:    legacy.RuntimeDir,
			ImageTag:      legacy.ImageTag,
			Status:        legacy.Status,
			LogText:       legacy.LogText,
			ContainerID:   legacy.ContainerID,
			Port:          legacy.Port,
			IsActive:      legacy.IsActive,
			DockerCompose: legacy.DockerCompose,
			Env:           legacy.Env,
			AppInstallID:  legacy.AppInstallID,
		}
		if err := r.db.Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *AppDeployRepo) ReplaceActiveContainerBinding(tx *gorm.DB, deploy *model.AppDeploy) error {
	if err := tx.Model(&model.AppDeploy{}).
		Where("website_id = ? AND is_active = ?", deploy.WebsiteID, true).
		Update("is_active", false).Error; err != nil {
		return err
	}
	return tx.Create(deploy).Error
}
