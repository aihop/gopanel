package repo

import "github.com/aihop/gopanel/app/model"

func (r *ReleaseRepo) MigrateTable() error {
	return r.db.AutoMigrate(&model.Release{})
}
func (r *ReleaseRepo) Create(item *model.Release) error {
	return r.db.Create(item).Error
}
func (r *ReleaseRepo) Get(id uint) (*model.Release, error) {
	var item model.Release
	err := r.db.First(&item, id).Error
	return &item, err
}
func (r *ReleaseRepo) GetByPipelineRecordID(recordID uint) (*model.Release, error) {
	var item model.Release
	err := r.db.Where("pipeline_record_id = ?", recordID).Order("id desc").First(&item).Error
	return &item, err
}
func (r *ReleaseRepo) ExistsByPipelineRecordIDs(recordIDs []uint) (map[uint]bool, error) {
	result := make(map[uint]bool)
	if len(recordIDs) == 0 {
		return result, nil
	}
	type pair struct{ PipelineRecordID uint }
	var rows []pair
	if err := r.db.Model(&model.Release{}).Select("DISTINCT pipeline_record_id").Where("pipeline_record_id IN ?", recordIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.PipelineRecordID > 0 {
			result[row.PipelineRecordID] = true
		}
	}
	return result, nil
}
func (r *ReleaseRepo) CountByPipelineRecordID(recordID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Release{}).Where("pipeline_record_id = ?", recordID).Count(&count).Error
	return count, err
}
func (r *ReleaseRepo) CountByPipelineID(pipelineID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Release{}).Where("pipeline_id = ?", pipelineID).Count(&count).Error
	return count, err
}
func (r *ReleaseRepo) PageByPipeline(pipelineId uint, page, limit int) (int64, []model.Release, error) {
	var total int64
	var list []model.Release
	query := r.db.Model(&model.Release{}).Where("pipeline_id = ?", pipelineId)
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	err := query.Order("id desc").Offset((page - 1) * limit).Limit(limit).Find(&list).Error
	return total, list, err
}
func (r *ReleaseRepo) LatestByPipelineID(pipelineId uint) (*model.Release, error) {
	var item model.Release
	err := r.db.Model(&model.Release{}).Where("pipeline_id = ?", pipelineId).Order("id desc").First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}
