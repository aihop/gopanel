package repo

import (
	"context"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewSSL() *SSLRepo {
	return &SSLRepo{
		db: global.DB,
	}
}

type SSLRepo struct {
	db *gorm.DB
}

func (w *SSLRepo) MigrateTable() error {
	return w.db.AutoMigrate(&model.SSL{}, &model.SSLPushRule{})
}

func (w *SSLRepo) WithID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func (w *SSLRepo) WithAppInstallId(appInstallID uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("app_install_id = ?", appInstallID)
	}
}

func (w *SSLRepo) WithIDs(ids []uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id in (?)", ids)
	}
}

func (w *SSLRepo) WithDomain(domain string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("primary_domain = ?", domain)
	}
}

func (w *SSLRepo) WithDomainLike(domain string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("primary_domain like ?", "%"+domain+"%")
	}
}

func (w *SSLRepo) WithAlias(alias string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("alias = ?", alias)
	}
}

func (w *SSLRepo) WithSSLSSLID(sslId uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("WebsiteSSL_ssl_id = ?", sslId)
	}
}

func (w *SSLRepo) WithGroupID(groupId uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("WebsiteSSL_group_id = ?", groupId)
	}
}

func (w *SSLRepo) WithDefaultServer() DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("default_server = 1")
	}
}

func (w *SSLRepo) Page(page, size int, opts ...DBOption) (int64, []model.SSL, error) {
	var SSLs []model.SSL
	db := getDb(opts...).Model(&model.SSL{})
	count := int64(0)
	db = db.Count(&count)
	err := db.Debug().Limit(size).Offset(size * (page - 1)).Find(&SSLs).Error
	return count, SSLs, err
}

func (w *SSLRepo) ListBy(opts ...DBOption) ([]model.SSL, error) {
	var SSLs []model.SSL
	err := getDb(opts...).Model(&model.SSL{}).Find(&SSLs).Error
	return SSLs, err
}

func (w *SSLRepo) GetFirst(opts ...DBOption) (model.SSL, error) {
	var SSL model.SSL
	db := getDb(opts...).Model(&model.SSL{})
	if err := db.First(&SSL).Error; err != nil {
		return SSL, err
	}
	return SSL, nil
}

func (w *SSLRepo) GetBy(opts ...DBOption) ([]model.SSL, error) {
	var SSLs []model.SSL
	db := getDb(opts...).Model(&model.SSL{})
	if err := db.Find(&SSLs).Error; err != nil {
		return SSLs, err
	}
	return SSLs, nil
}

func (w *SSLRepo) Create(ctx context.Context, app *model.SSL) error {
	return getTx(ctx).Omit(clause.Associations).Create(app).Error
}

func (w *SSLRepo) Save(ctx context.Context, app *model.SSL) error {
	return getTx(ctx).Omit(clause.Associations).Save(app).Error
}

func (w *SSLRepo) SaveWithoutCtx(SSL *model.SSL) error {
	return global.DB.Save(SSL).Error
}

func (w *SSLRepo) UpdateFields(id uint, fields map[string]interface{}) error {
	return global.DB.Model(&model.SSL{}).Where("id = ?", id).Updates(fields).Error
}

func (w *SSLRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.SSL{}).Error
}

func (w *SSLRepo) DeleteAll(ctx context.Context) error {
	return getTx(ctx).Where("1 = 1 ").Delete(&model.SSL{}).Error
}

func (r *SSLRepo) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	err = r.db.Model(&model.SSL{}).Scopes(gormx.Wheres(where)).Count(&res).Error
	return
}

func (r *SSLRepo) Search(ctx *gormx.Contextx) (res []*model.SSL, err error) {
	db := r.db.Model(&model.SSL{}).Scopes(gormx.Wheres(&gormx.Wherex{
		Wheres:     ctx.Wheres,
		Conditions: ctx.Conditions,
		Joins:      ctx.Joins,
		Select:     ctx.Select,
	}))
	if ctx.Order != "" {
		db = db.Order(ctx.Order)
	}
	if ctx.Limit > 0 {
		db = db.Offset((ctx.Page - 1) * ctx.Limit).Limit(ctx.Limit)
	}
	err = db.Preload("PushRules").Find(&res).Error
	return
}
