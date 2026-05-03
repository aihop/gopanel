package repo

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	fileutil "github.com/aihop/gopanel/utils/files"
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

func (r *PipelineRecordRepo) UpdateImageTag(id uint, imageTag string) error {
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Update("image_tag", imageTag).Error
}

func (r *PipelineRecordRepo) UpdateCommitHash(id uint, commitHash string) error {
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Update("commit_hash", commitHash).Error
}

func (r *PipelineRecordRepo) UpdateRunnerResult(id uint, releaseDir, containerID string, hostPort int) error {
	return r.db.Model(&model.PipelineRecord{}).Where("id = ?", id).Updates(map[string]interface{}{
		"runner_release_dir":  releaseDir,
		"runner_container_id": containerID,
		"runner_host_port":    hostPort,
	}).Error
}

func (r *PipelineRecordRepo) LatestRunnerContainerID(pipelineId uint) (string, error) {
	var rec model.PipelineRecord
	err := r.db.Model(&model.PipelineRecord{}).
		Where("pipeline_id = ? AND runner_container_id <> ''", pipelineId).
		Order("id desc").
		First(&rec).Error
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

	subQuery := r.db.Model(&model.PipelineRecord{}).
		Select("MAX(id) AS id").
		Where("pipeline_id IN ? AND runner_container_id <> ''", pipelineIDs).
		Group("pipeline_id")

	var records []model.PipelineRecord
	if err := r.db.Model(&model.PipelineRecord{}).
		Where("id IN (?)", subQuery).
		Find(&records).Error; err != nil {
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
	err := r.db.Model(&model.PipelineRecord{}).
		Where("pipeline_id = ?", pipelineId).
		Order("id desc").
		First(&rec).Error
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

type ReleaseRepo struct {
	db *gorm.DB
}

func NewRelease(db *gorm.DB) *ReleaseRepo {
	return &ReleaseRepo{db: db}
}

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

	type pair struct {
		PipelineRecordID uint
	}
	var rows []pair
	if err := r.db.Model(&model.Release{}).
		Select("DISTINCT pipeline_record_id").
		Where("pipeline_record_id IN ?", recordIDs).
		Find(&rows).Error; err != nil {
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
	err := r.db.Model(&model.Release{}).
		Where("pipeline_id = ?", pipelineId).
		Order("id desc").
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ReleaseRepo) EnsureUniquePipelineRecordIndex() error {
	if err := r.dedupeByPipelineRecordID(); err != nil {
		return err
	}
	if r.db.Migrator().HasIndex(&model.Release{}, "uniq_release_pipeline_record") {
		return nil
	}
	return r.db.Migrator().CreateIndex(&model.Release{}, "uniq_release_pipeline_record")
}

func (r *ReleaseRepo) dedupeByPipelineRecordID() error {
	type duplicateGroup struct {
		PipelineRecordID uint
		Count            int64
	}

	var groups []duplicateGroup
	if err := r.db.Model(&model.Release{}).
		Select("pipeline_record_id, COUNT(*) AS count").
		Where("pipeline_record_id > 0").
		Group("pipeline_record_id").
		Having("COUNT(*) > 1").
		Find(&groups).Error; err != nil {
		return err
	}
	if len(groups) == 0 {
		return nil
	}

	for _, group := range groups {
		if err := r.mergeDuplicatePipelineRecordReleases(group.PipelineRecordID); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReleaseRepo) mergeDuplicatePipelineRecordReleases(recordID uint) error {
	var releases []model.Release
	if err := r.db.Where("pipeline_record_id = ?", recordID).Order("id desc").Find(&releases).Error; err != nil {
		return err
	}
	if len(releases) <= 1 {
		return nil
	}

	keeper := releases[0]
	duplicates := releases[1:]
	merged := keeper
	for _, item := range duplicates {
		mergeReleaseFields(&merged, &item)
	}

	duplicateIDs := make([]uint, 0, len(duplicates))
	for _, item := range duplicates {
		duplicateIDs = append(duplicateIDs, item.ID)
	}
	sort.Slice(duplicateIDs, func(i, j int) bool {
		return duplicateIDs[i] < duplicateIDs[j]
	})

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Release{}).Where("id = ?", keeper.ID).Updates(map[string]interface{}{
			"version":       merged.Version,
			"commit_hash":   merged.CommitHash,
			"source_type":   merged.SourceType,
			"image_tag":     merged.ImageTag,
			"archive_file":  merged.ArchiveFile,
			"release_dir":   merged.ReleaseDir,
			"artifact_meta": merged.ArtifactMeta,
			"status":        merged.Status,
			"remark":        merged.Remark,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AppDeploy{}).
			Where("release_id IN ?", duplicateIDs).
			Update("release_id", keeper.ID).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Release{}, duplicateIDs).Error
	})
}

func (r *ReleaseRepo) FixSharedReleaseDirs() error {
	var releases []model.Release
	if err := r.db.Where("source_type = ?", "release_dir").Find(&releases).Error; err != nil {
		return err
	}
	if len(releases) == 0 {
		return nil
	}

	pipelineIDs := make([]uint, 0, len(releases))
	seenPipeline := make(map[uint]struct{})
	for _, item := range releases {
		if item.PipelineID == 0 {
			continue
		}
		if _, ok := seenPipeline[item.PipelineID]; ok {
			continue
		}
		seenPipeline[item.PipelineID] = struct{}{}
		pipelineIDs = append(pipelineIDs, item.PipelineID)
	}

	var pipelines []model.Pipeline
	if len(pipelineIDs) > 0 {
		if err := r.db.Where("id IN ?", pipelineIDs).Find(&pipelines).Error; err != nil {
			return err
		}
	}
	pipelineMap := make(map[uint]model.Pipeline, len(pipelines))
	for _, item := range pipelines {
		pipelineMap[item.ID] = item
	}

	for _, item := range releases {
		pipeline, ok := pipelineMap[item.PipelineID]
		if !ok {
			continue
		}
		if err := r.fixReleaseDirSnapshot(&pipeline, &item); err != nil {
			log.Printf("Release snapshot repair warning: release=%d pipeline=%d err=%v", item.ID, item.PipelineID, err)
		}
	}
	return nil
}

func (r *ReleaseRepo) fixReleaseDirSnapshot(pipeline *model.Pipeline, item *model.Release) error {
	if pipeline == nil || item == nil {
		return nil
	}

	currentDir := strings.TrimSpace(item.ReleaseDir)
	sharedDir := pipelineSharedReleaseDir(pipeline)
	snapshotDir := pipelineReleaseSnapshotDir(pipeline, item)

	if currentDir != "" && sameCleanPath(currentDir, snapshotDir) {
		return nil
	}
	if currentDir != "" && !sameCleanPath(currentDir, sharedDir) {
		return nil
	}

	srcDir := currentDir
	if srcDir == "" {
		srcDir = sharedDir
	}
	if srcDir == "" {
		return fmt.Errorf("empty release dir")
	}
	info, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source dir not found: %s", srcDir)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source dir is not a directory: %s", srcDir)
	}

	if err := os.RemoveAll(snapshotDir); err != nil {
		return err
	}
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return err
	}
	if err := fileutil.CopyDirContents(srcDir, snapshotDir); err != nil {
		return err
	}
	return r.db.Model(&model.Release{}).Where("id = ?", item.ID).Update("release_dir", snapshotDir).Error
}

func pipelineBaseDirForRelease(p *model.Pipeline) string {
	key := strings.TrimSpace(p.PipelineKey)
	if key == "" {
		key = fmt.Sprintf("%d", p.ID)
	}
	return filepath.Join(global.CONF.System.BaseDir, "pipelines", key)
}

func pipelineSharedReleaseDir(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDirForRelease(p), "release")
}

func pipelineArchiveDirForRelease(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDirForRelease(p), "archive")
}

func pipelineReleaseSnapshotDir(p *model.Pipeline, item *model.Release) string {
	name := fmt.Sprintf("release-%d", item.ID)
	if item.PipelineRecordID > 0 {
		name = fmt.Sprintf("release-record-%d", item.PipelineRecordID)
	}
	return filepath.Join(pipelineArchiveDirForRelease(p), name)
}

func sameCleanPath(a, b string) bool {
	if strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func mergeReleaseFields(dst, src *model.Release) {
	if dst == nil || src == nil {
		return
	}
	if strings.TrimSpace(dst.Version) == "" && strings.TrimSpace(src.Version) != "" {
		dst.Version = src.Version
	}
	if strings.TrimSpace(dst.CommitHash) == "" && strings.TrimSpace(src.CommitHash) != "" {
		dst.CommitHash = src.CommitHash
	}
	if strings.TrimSpace(dst.SourceType) == "" && strings.TrimSpace(src.SourceType) != "" {
		dst.SourceType = src.SourceType
	}
	if strings.TrimSpace(dst.ImageTag) == "" && strings.TrimSpace(src.ImageTag) != "" {
		dst.ImageTag = src.ImageTag
	}
	if strings.TrimSpace(dst.ArchiveFile) == "" && strings.TrimSpace(src.ArchiveFile) != "" {
		dst.ArchiveFile = src.ArchiveFile
	}
	if strings.TrimSpace(dst.ReleaseDir) == "" && strings.TrimSpace(src.ReleaseDir) != "" {
		dst.ReleaseDir = src.ReleaseDir
	}
	if strings.TrimSpace(dst.ArtifactMeta) == "" && strings.TrimSpace(src.ArtifactMeta) != "" {
		dst.ArtifactMeta = src.ArtifactMeta
	}
	if strings.TrimSpace(dst.Status) == "" && strings.TrimSpace(src.Status) != "" {
		dst.Status = src.Status
	}
	if strings.TrimSpace(dst.Remark) == "" && strings.TrimSpace(src.Remark) != "" {
		dst.Remark = src.Remark
	}
}
