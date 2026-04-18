package repo

import (
	"context"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type CloudAccountRepo struct{}

func NewCloudAccount() *CloudAccountRepo {
	return &CloudAccountRepo{}
}

func (a *CloudAccountRepo) MigrateTable() error {
	return global.DB.AutoMigrate(&model.CloudAccount{})
}

func (a *CloudAccountRepo) Get(ctx context.Context, accountType string) (model.CloudAccount, error) {
	var account model.CloudAccount
	err := getTx(ctx).Where("account_type = ?", accountType).First(&account).Error
	return account, err
}

func (a *CloudAccountRepo) List(page, limit int) ([]model.CloudAccount, error) {
	var list []model.CloudAccount
	tx := global.DB.Model(&model.CloudAccount{}).Order("id desc")
	if limit != 0 {
		tx = tx.Limit(limit).Offset((page - 1) * limit)
	}
	err := tx.Find(&list).Error
	return list, err
}

func (a *CloudAccountRepo) Count() (int64, error) {
	var count int64
	err := global.DB.Model(&model.CloudAccount{}).Count(&count).Error
	return count, err
}

func (a *CloudAccountRepo) Create(ctx context.Context, account *model.CloudAccount) error {
	return getTx(ctx).Create(account).Error
}

func (a *CloudAccountRepo) Update(ctx context.Context, account *model.CloudAccount) error {
	return getTx(ctx).Save(account).Error
}

func (a *CloudAccountRepo) DeleteByID(ctx context.Context, id uint) error {
	return getTx(ctx).Delete(&model.CloudAccount{}, id).Error
}

func (a *CloudAccountRepo) GetByID(id uint) (model.CloudAccount, error) {
	var account model.CloudAccount
	err := global.DB.Where("id = ?", id).First(&account).Error
	return account, err
}
