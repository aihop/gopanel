package repo

import (
	"context"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewSSLPushRule() *SSLPushRuleRepo {
	return &SSLPushRuleRepo{
		db: global.DB,
	}
}

type SSLPushRuleRepo struct {
	db *gorm.DB
}

func (w *SSLPushRuleRepo) MigrateTable() error {
	return w.db.AutoMigrate(&model.SSLPushRule{})
}

func (w *SSLPushRuleRepo) WithID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func (w *SSLPushRuleRepo) WithSSLID(sslId uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("ssl_id = ?", sslId)
	}
}

func (w *SSLPushRuleRepo) GetFirst(opts ...DBOption) (model.SSLPushRule, error) {
	var rule model.SSLPushRule
	db := getDb(opts...).Model(&model.SSLPushRule{})
	if err := db.First(&rule).Error; err != nil {
		return rule, err
	}
	return rule, nil
}

func (w *SSLPushRuleRepo) GetBy(opts ...DBOption) ([]model.SSLPushRule, error) {
	var rules []model.SSLPushRule
	db := getDb(opts...).Model(&model.SSLPushRule{})
	if err := db.Find(&rules).Error; err != nil {
		return rules, err
	}
	return rules, nil
}

func (w *SSLPushRuleRepo) Create(ctx context.Context, rule *model.SSLPushRule) error {
	return getTx(ctx).Omit(clause.Associations).Create(rule).Error
}

func (w *SSLPushRuleRepo) Save(ctx context.Context, rule *model.SSLPushRule) error {
	return getTx(ctx).Omit(clause.Associations).Save(rule).Error
}

func (w *SSLPushRuleRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.SSLPushRule{}).Error
}

func (r *SSLPushRuleRepo) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	err = r.db.Model(&model.SSLPushRule{}).Scopes(gormx.Wheres(where)).Count(&res).Error
	return
}

func (r *SSLPushRuleRepo) Search(ctx *gormx.Contextx) (res []*model.SSLPushRule, err error) {
	db := r.db.Model(&model.SSLPushRule{}).Scopes(gormx.Wheres(&gormx.Wherex{
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
	err = db.Find(&res).Error
	return
}
