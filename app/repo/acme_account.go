package repo

import (
	"context"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type AcmeAccountRepo struct {
	db *gorm.DB
}

func NewAcmeAccount() *AcmeAccountRepo {
	return &AcmeAccountRepo{db: global.DB}
}

func (r *AcmeAccountRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.AcmeAccount{})
}

func (r *AcmeAccountRepo) List(page, limit int) ([]model.AcmeAccount, error) {
	var list []model.AcmeAccount
	tx := r.db.Model(&model.AcmeAccount{}).Order("id DESC")
	if limit > 0 {
		tx = tx.Offset((page - 1) * limit).Limit(limit)
	}
	return list, tx.Find(&list).Error
}

func (r *AcmeAccountRepo) Count() (int64, error) {
	var count int64
	return count, r.db.Model(&model.AcmeAccount{}).Count(&count).Error
}

func (r *AcmeAccountRepo) Create(ctx context.Context, account *model.AcmeAccount) error {
	return getTx(ctx).Create(account).Error
}

func (r *AcmeAccountRepo) DeleteByID(ctx context.Context, id uint) error {
	return getTx(ctx).Delete(&model.AcmeAccount{}, id).Error
}

func (r *AcmeAccountRepo) GetByID(id uint) (model.AcmeAccount, error) {
	var account model.AcmeAccount
	return account, r.db.First(&account, id).Error
}
