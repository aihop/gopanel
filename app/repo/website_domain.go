package repo

import (
	"context"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

type WebsiteDomainRepo struct {
}

func NewWebsiteDomain() *WebsiteDomainRepo {
	return &WebsiteDomainRepo{}
}

func (w *WebsiteDomainRepo) WithPort(port int) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("port = ?", port)
	}
}

func (w WebsiteDomainRepo) GetFirst(opts ...DBOption) (model.WebsiteDomain, error) {
	var domain model.WebsiteDomain
	db := getDb(opts...).Model(&model.WebsiteDomain{})
	if err := db.First(&domain).Error; err != nil {
		return domain, err
	}
	return domain, nil
}

func (w WebsiteDomainRepo) WithDomain(domain string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("domain = ?", domain)
	}
}

func (w WebsiteDomainRepo) WithWebsiteId(websiteId uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("website_id = ?", websiteId)
	}
}

func (w WebsiteDomainRepo) BatchCreate(ctx context.Context, domains []model.WebsiteDomain) error {
	return getTx(ctx).Model(&model.WebsiteDomain{}).Create(&domains).Error
}

func (w WebsiteDomainRepo) GetBy(opts ...DBOption) ([]model.WebsiteDomain, error) {
	var domains []model.WebsiteDomain
	db := getDb(opts...).Model(&model.WebsiteDomain{})
	if err := db.Find(&domains).Error; err != nil {
		return domains, err
	}
	return domains, nil
}

func (w WebsiteDomainRepo) DeleteByWebsiteIdNotIsPrimary(ctx context.Context, websiteId uint) error {
	return getTx(ctx).Where("website_id = ? AND is_primary != 20", websiteId).Delete(&model.WebsiteDomain{}).Error
}
