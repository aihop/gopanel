package repo

import (
	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

type FlowRepo struct {
	db *gorm.DB
}

func NewFlow(db *gorm.DB) *FlowRepo {
	return &FlowRepo{db: db}
}

func (r *FlowRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.Flow{}, &model.FlowEnvironment{}, &model.FlowRun{}, &model.FlowStageRun{})
}

func (r *FlowRepo) Page(userID uint, includeAll bool, page, limit int) (int64, []model.Flow, error) {
	query := r.db.Model(&model.Flow{})
	if !includeAll {
		query = query.Where("created_by = ?", userID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var items []model.Flow
	err := query.Preload("Environments", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort asc, id asc")
	}).Order("id desc").Offset((page - 1) * limit).Limit(limit).Find(&items).Error
	return total, items, err
}

func (r *FlowRepo) CreateWithEnvironments(item *model.Flow) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(item).Error
	})
}

func (r *FlowRepo) Get(id uint) (*model.Flow, error) {
	var item model.Flow
	err := r.db.Preload("Environments", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort asc, id asc")
	}).First(&item, id).Error
	return &item, err
}
