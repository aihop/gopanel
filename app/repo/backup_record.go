package repo

import (
	"context"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"gorm.io/gorm"
)

func NewBackupRecord() *BackupRecordRepo {
	return &BackupRecordRepo{
		db: global.DB,
	}
}

type BackupRecordRepo struct {
	db *gorm.DB
}

func (r *BackupRecordRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.BackupRecord{})
}

func (r *BackupRecordRepo) List(ctx *gormx.Contextx) (res []*model.BackupRecord, err error) {
	err = r.db.Model(model.BackupRecord{}).Scopes(gormx.Context(ctx)).Find(&res).Error
	return
}

func (r *BackupRecordRepo) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	err = r.db.Model(&model.BackupRecord{}).Scopes(gormx.Wheres(where)).Count(&res).Error
	return
}

func (u *BackupRecordRepo) WithByCronID(cronjobID uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("cronjob_id = ?", cronjobID)
	}
}

func (u *BackupRecordRepo) WithByDetailName(detailName string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		if len(detailName) == 0 {
			return g
		}
		return g.Where("detail_name = ?", detailName)
	}
}

func (u *BackupRecordRepo) WithByType(backupType string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		if len(backupType) == 0 {
			return g
		}
		return g.Where("type = ?", backupType)
	}
}

func (u *BackupRecordRepo) ListRecord(opts ...DBOption) ([]model.BackupRecord, error) {
	var users []model.BackupRecord
	db := global.DB.Model(&model.BackupRecord{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&users).Error
	return users, err
}

func (u *BackupRecordRepo) Create(record *model.BackupRecord) error {
	return global.DB.Create(record).Error
}

func (u *BackupRecordRepo) Delete(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.BackupRecord{}).Error
}
