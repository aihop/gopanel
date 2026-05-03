package repo

import "github.com/aihop/gopanel/app/model"

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
func (r *PipelineRepo) Page(page, limit int) (int64, []model.Pipeline, error) {
	var total int64
	var list []model.Pipeline
	err := r.db.Model(&model.Pipeline{}).Count(&total).Error
	if err != nil {
		return 0, nil, err
	}
	err = r.db.Order("id desc").Offset((page - 1) * limit).Limit(limit).Find(&list).Error
	return total, list, err
}
func (r *PipelineRepo) ExistsPipelineKey(pipelineKey string, excludeID uint) (bool, error) {
	if pipelineKey == "" {
		return false, nil
	}
	query := r.db.Model(&model.Pipeline{}).Where("pipeline_key = ?", pipelineKey)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
