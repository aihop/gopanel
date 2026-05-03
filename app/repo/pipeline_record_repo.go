package repo

import "github.com/aihop/gopanel/app/model"

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
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Updates(map[string]interface{}{"status": status, "error_message": errMsg}).Error
}
func (r *PipelineRecordRepo) UpdateArchive(id uint, archiveFile string) error {
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Update("archive_file", archiveFile).Error
}
func (r *PipelineRecordRepo) UpdateImageTag(id uint, imageTag string) error {
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Update("image_tag", imageTag).Error
}
func (r *PipelineRecordRepo) UpdateCommitHash(id uint, commitHash string) error {
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Update("commit_hash", commitHash).Error
}
func (r *PipelineRecordRepo) UpdateRunnerResult(id uint, releaseDir, containerID string, hostPort int) error {
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Updates(map[string]interface{}{"runner_release_dir": releaseDir, "runner_container_id": containerID, "runner_host_port": hostPort}).Error
}
func (r *PipelineRecordRepo) LatestRunnerContainerID(pipelineId uint) (string, error) {
	var rec model.PipelineRecord
	err := r.db.Model(&model.PipelineRecord{}).Where("pipeline_id = ? AND runner_container_id <> ''", pipelineId).Order("id desc").First(&rec).Error
	if err != nil {
		return "", err
	}
	return rec.RunnerContainerID, nil
}
func (r *PipelineRecordRepo) LatestRunnerContainerIDs(pipelineIDs []uint) (map[uint]string, error) {
	result := make(map[uint]string)
	if len(pipelineIDs) == 0 {
		return result, nil
	}
	subQuery := r.db.Model(&model.PipelineRecord{}).Select("MAX(id) AS id").Where("pipeline_id IN ? AND runner_container_id <> ''", pipelineIDs).Group("pipeline_id")
	var records []model.PipelineRecord
	if err := r.db.Model(&model.PipelineRecord{}).Where("id IN (?)", subQuery).Find(&records).Error; err != nil {
		return nil, err
	}
	for _, rec := range records {
		if rec.PipelineID > 0 && rec.RunnerContainerID != "" {
			result[rec.PipelineID] = rec.RunnerContainerID
		}
	}
	return result, nil
}
func (r *PipelineRecordRepo) LatestByPipelineID(pipelineId uint) (*model.PipelineRecord, error) {
	var rec model.PipelineRecord
	err := r.db.Model(&model.PipelineRecord{}).Where("pipeline_id = ?", pipelineId).Order("id desc").First(&rec).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}
func (r *PipelineRecordRepo) PageByPipeline(pipelineId uint, page, limit int) (int64, []model.PipelineRecord, error) {
	var total int64
	var list []model.PipelineRecord
	query := r.db.Model(&model.PipelineRecord{}).Where("pipeline_id = ?", pipelineId)
	err := query.Count(&total).Error
	if err != nil {
		return 0, nil, err
	}
	err = query.Order("id desc").Offset((page - 1) * limit).Limit(limit).Find(&list).Error
	return total, list, err
}
func (r *PipelineRecordRepo) CountByPipelineID(pipelineID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.PipelineRecord{}).Where("pipeline_id = ?", pipelineID).Count(&count).Error
	return count, err
}
