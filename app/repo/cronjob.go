package repo

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"gorm.io/gorm"
)

func NewCronjob() *CronjobRepo {
	return &CronjobRepo{
		db: global.DB,
	}
}

type CronjobRepo struct {
	db *gorm.DB
}

func (r *CronjobRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.Cronjob{}, &model.JobRecords{})
}

func (r *CronjobRepo) Create(item *model.Cronjob) error {
	return r.db.Create(item).Error
}

func (r *CronjobRepo) Update(item *model.Cronjob) error {
	return r.db.Save(item).Error
}

func (r *CronjobRepo) UpdateEntryID(id uint, entryID int) error {
	return r.db.Model(&model.Cronjob{}).Where("id = ?", id).Update("entry_id", entryID).Error
}

func (r *CronjobRepo) Get(id uint) (res *model.Cronjob, err error) {
	err = r.db.Where("id = ?", id).First(&res).Error
	return
}

func (r *CronjobRepo) Delete(id uint) error {
	return r.db.Delete(&model.Cronjob{}, id).Error
}

func (r *CronjobRepo) List(ctx *gormx.Contextx) (res []*model.Cronjob, err error) {
	db := r.db.Model(&model.Cronjob{}).Scopes(gormx.Wheres(&gormx.Wherex{
		Wheres:     ctx.Wheres,
		Conditions: ctx.Conditions,
		Joins:      ctx.Joins,
		Select:     ctx.Select,
	}))
	if ctx.Order != "" {
		db = db.Order(ctx.Order)
	} else {
		db = db.Order("id desc")
	}
	if ctx.Limit > 0 {
		db = db.Offset((ctx.Page - 1) * ctx.Limit).Limit(ctx.Limit)
	}
	err = db.Find(&res).Error
	return
}

func (r *CronjobRepo) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	err = r.db.Model(&model.Cronjob{}).Scopes(gormx.Wheres(where)).Count(&res).Error
	return
}

func (r *CronjobRepo) ListByType(jobType string) (res []*model.Cronjob, err error) {
	err = r.db.Where("type = ?", jobType).Find(&res).Error
	return
}

func (r *CronjobRepo) ListEnabled() (res []*model.Cronjob, err error) {
	err = r.db.Where("status = ?", constant.StatusEnable).Find(&res).Error
	return
}

func (r *CronjobRepo) CreateRecord(record *model.JobRecords) error {
	return r.db.Create(record).Error
}

func (r *CronjobRepo) UpdateRecord(id uint, vars map[string]interface{}) error {
	return r.db.Model(&model.JobRecords{}).Where("id = ?", id).Updates(vars).Error
}

func (r *CronjobRepo) ListRecords(cronjobID uint, limit int) (res []*model.JobRecords, err error) {
	db := r.db.Where("cronjob_id = ?", cronjobID).Order("start_time desc")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err = db.Find(&res).Error
	return
}

func (r *CronjobRepo) DeleteRecords(cronjobID uint) error {
	return r.db.Where("cronjob_id = ?", cronjobID).Delete(&model.JobRecords{}).Error
}

// ListLatestRecords 批量取每个 cronjobID 最近一次的执行记录，用于列表页展示，避免逐行 N+1 查询
func (r *CronjobRepo) ListLatestRecords(cronjobIDs []uint) (map[uint]*model.JobRecords, error) {
	result := make(map[uint]*model.JobRecords, len(cronjobIDs))
	if len(cronjobIDs) == 0 {
		return result, nil
	}
	var records []*model.JobRecords
	if err := r.db.Where("cronjob_id in ?", cronjobIDs).Order("start_time desc").Find(&records).Error; err != nil {
		return nil, err
	}
	for _, record := range records {
		if _, ok := result[record.CronjobID]; !ok {
			result[record.CronjobID] = record
		}
	}
	return result, nil
}
