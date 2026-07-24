package repo

import (
	"context"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

type WebsiteUpstreamRepo struct{}

func NewWebsiteUpstream() *WebsiteUpstreamRepo {
	return &WebsiteUpstreamRepo{}
}

func (r *WebsiteUpstreamRepo) WithWebsiteID(websiteID uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("website_id = ?", websiteID)
	}
}

func (r *WebsiteUpstreamRepo) BatchCreate(ctx context.Context, upstreams []model.WebsiteUpstream) error {
	if len(upstreams) == 0 {
		return nil
	}
	return getTx(ctx).Model(&model.WebsiteUpstream{}).Create(&upstreams).Error
}

func (r *WebsiteUpstreamRepo) ReplaceByWebsiteID(ctx context.Context, websiteID uint, upstreams []model.WebsiteUpstream) error {
	tx := getTx(ctx)
	if err := tx.Where("website_id = ?", websiteID).Delete(&model.WebsiteUpstream{}).Error; err != nil {
		return err
	}
	if len(upstreams) == 0 {
		return nil
	}
	return tx.Model(&model.WebsiteUpstream{}).Create(&upstreams).Error
}

func (r *WebsiteUpstreamRepo) DeleteByWebsiteID(ctx context.Context, websiteID uint) error {
	return getTx(ctx).Where("website_id = ?", websiteID).Delete(&model.WebsiteUpstream{}).Error
}
