package repo

import (
	"errors"
	"fmt"
	"time"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

func (r *FlowRepo) CreateRun(item *model.FlowRun, stage *model.FlowStageRun) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		stage.FlowRunID = item.ID
		if stage.IdempotencyKey == "" {
			stage.IdempotencyKey = fmt.Sprintf("flow:%d:%s:%d", item.ID, stage.Stage, stage.Attempt)
		}
		return tx.Create(stage).Error
	})
}

func (r *FlowRepo) DeleteRun(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("flow_run_id = ?", id).Delete(&model.FlowStageRun{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.FlowRun{}, id).Error
	})
}

func (r *FlowRepo) GetRun(id uint, userID uint, includeAll bool) (*model.FlowRun, error) {
	query := r.db.Model(&model.FlowRun{}).
		Joins("JOIN flows ON flows.id = flow_runs.flow_id")
	if !includeAll {
		query = query.Where("flows.created_by = ?", userID)
	}
	var item model.FlowRun
	err := query.Preload("Stages", func(db *gorm.DB) *gorm.DB {
		return db.Order("id asc")
	}).Where("flow_runs.id = ?", id).First(&item).Error
	return &item, err
}

func (r *FlowRepo) GetRunInternal(id uint) (*model.FlowRun, error) {
	var item model.FlowRun
	err := r.db.First(&item, id).Error
	return &item, err
}

func (r *FlowRepo) PageRuns(flowID, userID uint, includeAll bool, page, limit int) (int64, []model.FlowRun, error) {
	query := r.db.Model(&model.FlowRun{}).
		Joins("JOIN flows ON flows.id = flow_runs.flow_id")
	if flowID > 0 {
		query = query.Where("flow_runs.flow_id = ?", flowID)
	}
	if !includeAll {
		query = query.Where("flows.created_by = ?", userID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var items []model.FlowRun
	err := query.Select("flow_runs.*").Order("flow_runs.id desc").
		Offset((page - 1) * limit).Limit(limit).Find(&items).Error
	return total, items, err
}

func (r *FlowRepo) VersionExists(flowID uint, version string) (bool, error) {
	var count int64
	err := r.db.Model(&model.FlowRun{}).Where("flow_id = ? AND version = ?", flowID, version).Count(&count).Error
	return count > 0, err
}

func (r *FlowRepo) UpdateRun(id uint, values map[string]any) error {
	return r.db.Model(&model.FlowRun{}).Where("id = ?", id).Updates(values).Error
}

func (r *FlowRepo) ResumeFailedRun(id uint, values map[string]any, stage *model.FlowStageRun) (bool, error) {
	resumed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.FlowRun{}).Where("id = ? AND status = ?", id, "failed").Updates(values)
		if result.Error != nil || result.RowsAffected != 1 {
			return result.Error
		}
		if err := tx.Create(stage).Error; err != nil {
			return err
		}
		resumed = true
		return nil
	})
	return resumed, err
}

func (r *FlowRepo) NextStageAttempt(flowRunID uint, stage string) (int, error) {
	var maxAttempt int
	err := r.db.Model(&model.FlowStageRun{}).
		Where("flow_run_id = ? AND stage = ?", flowRunID, stage).
		Select("COALESCE(MAX(attempt), 0)").Scan(&maxAttempt).Error
	return maxAttempt + 1, err
}

func (r *FlowRepo) CurrentStageAttempt(flowRunID uint, stage string) (int, error) {
	var latest model.FlowStageRun
	err := r.db.Where("flow_run_id = ? AND stage = ?", flowRunID, stage).
		Order("attempt desc").First(&latest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return latest.Attempt, nil
}

func (r *FlowRepo) StageAttemptForExecution(flowRunID uint, stage string) (int, error) {
	var latest model.FlowStageRun
	err := r.db.Where("flow_run_id = ? AND stage = ?", flowRunID, stage).
		Order("attempt desc").First(&latest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	if latest.Status == "pending" || latest.Status == "running" {
		return latest.Attempt, nil
	}
	return latest.Attempt + 1, nil
}

func (r *FlowRepo) UpsertStage(stage *model.FlowStageRun) error {
	var existing model.FlowStageRun
	err := r.db.Where("flow_run_id = ? AND stage = ? AND attempt = ?", stage.FlowRunID, stage.Stage, stage.Attempt).First(&existing).Error
	if err == nil {
		stage.ID = existing.ID
		return r.db.Model(&existing).Updates(stage).Error
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.Create(stage).Error
}

func (r *FlowRepo) ClaimRun(id uint, owner string, now, expiresAt time.Time) (bool, error) {
	result := r.db.Model(&model.FlowRun{}).
		Where("id = ? AND status IN ? AND (lease_expires_at IS NULL OR lease_expires_at < ? OR lease_owner = ?)",
			id, []string{"queued", "running"}, now, owner).
		Updates(map[string]any{"lease_owner": owner, "lease_expires_at": expiresAt})
	return result.RowsAffected == 1, result.Error
}

func (r *FlowRepo) RenewRunLease(id uint, owner string, expiresAt time.Time) (bool, error) {
	result := r.db.Model(&model.FlowRun{}).
		Where("id = ? AND status IN ? AND lease_owner = ?", id, []string{"queued", "running"}, owner).
		Update("lease_expires_at", expiresAt)
	return result.RowsAffected == 1, result.Error
}

func (r *FlowRepo) ActiveRunIDs() ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.FlowRun{}).Where("status IN ?", []string{"queued", "running"}).Pluck("id", &ids).Error
	return ids, err
}
