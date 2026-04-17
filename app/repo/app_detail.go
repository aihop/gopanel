package repo

import (
	"context"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

func NewAppDetail() *AppDetailRepo {
	return &AppDetailRepo{
		DB: global.DB,
	}
}

type AppDetailRepo struct {
	DB *gorm.DB
}

func (r *AppDetailRepo) MigrateTable() error {
	return r.DB.AutoMigrate(&model.AppDetail{})
}

func (a *AppDetailRepo) CtxUpdate(ctx context.Context, detail model.AppDetail) error {
	return getTx(ctx).Save(&detail).Error
}

func (a AppDetailRepo) GetFirst(opts ...DBOption) (model.AppDetail, error) {
	var detail model.AppDetail
	err := getDb(opts...).Model(&model.AppDetail{}).First(&detail).Error
	return detail, err
}

func (a AppDetailRepo) GetBy(opts ...DBOption) ([]model.AppDetail, error) {
	var details []model.AppDetail
	err := getDb(opts...).Model(&model.AppDetail{}).Find(&details).Error
	return details, err
}

func (a AppDetailRepo) WithAppId(appId uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("app_id = ?", appId)
	}
}

func (a AppDetailRepo) Update(ctx context.Context, detail model.AppDetail) error {
	return getTx(ctx).Save(&detail).Error
}
