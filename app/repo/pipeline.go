package repo

import (
	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

type PipelineRepo struct {
	db *gorm.DB
}

func NewPipeline(db *gorm.DB) *PipelineRepo {
	return &PipelineRepo{db: db}
}

// 迁移
func (r *PipelineRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.Pipeline{})
}

func (r *PipelineRepo) Create(pipeline *model.Pipeline) error {
	return r.db.Create(pipeline).Error
}

func (r *PipelineRepo) Update(pipeline *model.Pipeline) error {
	return r.db.Save(pipeline).Error
}

func (r *PipelineRepo) Delete(id uint) error {
	return r.db.Delete(&model.Pipeline{}, id).Error
}

func (r *PipelineRepo) Get(id uint) (*model.Pipeline, error) {
	var pipeline model.Pipeline
	err := r.db.First(&pipeline, id).Error
	return &pipeline, err
}

func (r *PipelineRepo) Page(page, pageSize int) (int64, []model.Pipeline, error) {
	var total int64
	var list []model.Pipeline
	err := r.db.Model(&model.Pipeline{}).Count(&total).Error
	if err != nil {
		return 0, nil, err
	}
	err = r.db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return total, list, err
}

type PipelineRecordRepo struct {
	db *gorm.DB
}

func NewPipelineRecord(db *gorm.DB) *PipelineRecordRepo {
	return &PipelineRecordRepo{db: db}
}

// 迁移
func (r *PipelineRecordRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.PipelineRecord{})
}

func (r *PipelineRecordRepo) Create(record *model.PipelineRecord) error {
	return r.db.Create(record).Error
}

func (r *PipelineRecordRepo) Get(id uint) (*model.PipelineRecord, error) {
	var record model.PipelineRecord
	err := r.db.First(&record, id).Error
	return &record, err
}

func (r *PipelineRecordRepo) Delete(id uint) error {
	return r.db.Delete(&model.PipelineRecord{}, id).Error
}

func (r *PipelineRecordRepo) UpdateStatus(id uint, status, errMsg string) error {
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        status,
		"error_message": errMsg,
	}).Error
}

func (r *PipelineRecordRepo) UpdateArchive(id uint, archiveFile string) error {
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Update("archive_file", archiveFile).Error
}

func (r *PipelineRecordRepo) PageByPipeline(pipelineId uint, page, pageSize int) (int64, []model.PipelineRecord, error) {
	var total int64
	var list []model.PipelineRecord
	query := r.db.Model(&model.PipelineRecord{}).Where("pipeline_id = ?", pipelineId)
	err := query.Count(&total).Error
	if err != nil {
		return 0, nil, err
	}
	err = query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return total, list, err
}
