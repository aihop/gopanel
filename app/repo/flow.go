package repo

import (
	"errors"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

var ErrFlowHasHistory = errors.New("flow has run history")

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

func (r *FlowRepo) UpdateWithEnvironments(item *model.Flow, environments []model.FlowEnvironment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Flow{}).Where("id = ?", item.ID).Updates(map[string]any{
			"name":                           item.Name,
			"pipeline_id":                    item.PipelineID,
			"auto_start_after_code_delivery": item.AutoStartAfterCodeDelivery,
		}).Error; err != nil {
			return err
		}
		var existing []model.FlowEnvironment
		if err := tx.Where("flow_id = ?", item.ID).Find(&existing).Error; err != nil {
			return err
		}
		existingByName := make(map[string]model.FlowEnvironment, len(existing))
		for _, environment := range existing {
			existingByName[environment.Name] = environment
		}
		names := make([]string, 0, len(environments))
		for index := range environments {
			environment := &environments[index]
			names = append(names, environment.Name)
			if current, ok := existingByName[environment.Name]; ok {
				environment.ID = current.ID
				environment.FlowID = item.ID
				if err := tx.Model(&model.FlowEnvironment{}).Where("id = ?", current.ID).Updates(map[string]any{
					"website_id":        environment.WebsiteID,
					"auto_deploy":       environment.AutoDeploy,
					"approval_required": environment.ApprovalRequired,
					"sort":              environment.Sort,
					"enabled":           true,
				}).Error; err != nil {
					return err
				}
				continue
			}
			environment.FlowID = item.ID
			if err := tx.Create(environment).Error; err != nil {
				return err
			}
		}
		return tx.Where("flow_id = ? AND name NOT IN ?", item.ID, names).Delete(&model.FlowEnvironment{}).Error
	})
}

func (r *FlowRepo) DeleteConfiguration(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var runCount int64
		if err := tx.Model(&model.FlowRun{}).Where("flow_id = ?", id).Count(&runCount).Error; err != nil {
			return err
		}
		if runCount > 0 {
			return ErrFlowHasHistory
		}
		if err := tx.Where("flow_id = ?", id).Delete(&model.FlowEnvironment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Flow{}, id).Error
	})
}

func (r *FlowRepo) Get(id uint) (*model.Flow, error) {
	var item model.Flow
	err := r.db.Preload("Environments", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort asc, id asc")
	}).First(&item, id).Error
	return &item, err
}
