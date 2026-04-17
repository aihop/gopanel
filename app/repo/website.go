package repo

import (
	"context"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewWebsite() *WebsiteRepo {
	return &WebsiteRepo{
		db: global.DB,
	}
}

type WebsiteRepo struct {
	db *gorm.DB
}

func (r *WebsiteRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.Website{}, &model.WebsiteDomain{})
}

func (w *WebsiteRepo) WithAppInstallId(appInstallID uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("app_install_id = ?", appInstallID)
	}
}

func (w *WebsiteRepo) WithIDs(ids []uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id in (?)", ids)
	}
}

func (w *WebsiteRepo) WithID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func (w *WebsiteRepo) WithDomain(domain string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("primary_domain = ? OR alias = ?", domain, domain)
	}
}

func (w *WebsiteRepo) WithDomainLike(domain string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("primary_domain like ?", "%"+domain+"%")
	}
}

func (w *WebsiteRepo) WithAlias(alias string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("alias = ?", alias)
	}
}

func (w *WebsiteRepo) WithDefaultServer() DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("default_server = 1")
	}
}

func (w *WebsiteRepo) Page(page, size int, opts ...DBOption) (int64, []model.Website, error) {
	var websites []model.Website
	db := getDb(opts...).Model(&model.Website{})
	count := int64(0)
	db = db.Count(&count)
	err := db.Debug().Limit(size).Offset(size * (page - 1)).Find(&websites).Error
	return count, websites, err
}

func (r *WebsiteRepo) List(ctx *gormx.Contextx) (res []*model.Website, err error) {
	err = r.db.Model(&model.Website{}).Preload("Domains").Find(&res).Error
	return
}

func (w *WebsiteRepo) ListBy(opts ...DBOption) ([]model.Website, error) {
	var websites []model.Website
	err := getDb(opts...).Model(&model.Website{}).Preload("Domains").Find(&websites).Error
	return websites, err
}

func (w *WebsiteRepo) GetFirst(opts ...DBOption) (model.Website, error) {
	var website model.Website
	db := getDb(opts...).Model(&model.Website{})
	if err := db.Preload("Domains").First(&website).Error; err != nil {
		return website, err
	}
	return website, nil
}

func (w *WebsiteRepo) GetBy(opts ...DBOption) ([]model.Website, error) {
	var websites []model.Website
	db := getDb(opts...).Model(&model.Website{})
	if err := db.Find(&websites).Error; err != nil {
		return websites, err
	}
	return websites, nil
}

func (w *WebsiteRepo) Create(ctx context.Context, app *model.Website) error {
	return getTx(ctx).Omit(clause.Associations).Create(app).Error
}

func (w *WebsiteRepo) Save(ctx context.Context, app *model.Website) error {
	return getTx(ctx).Omit(clause.Associations).Save(app).Error
}

func (w *WebsiteRepo) SaveWithoutCtx(website *model.Website) error {
	return global.DB.Save(website).Error
}

func (w *WebsiteRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.Website{}).Error
}

func (w *WebsiteRepo) DeleteAll(ctx context.Context) error {
	return getTx(ctx).Where("1 = 1 ").Delete(&model.Website{}).Error
}

func (r *WebsiteRepo) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	err = r.db.Model(&model.Website{}).Scopes(gormx.Wheres(where)).Count(&res).Error
	return
}

func (w *WebsiteRepo) WithPipelineID(pipelineID uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("pipeline_id = ?", pipelineID)
	}
}
