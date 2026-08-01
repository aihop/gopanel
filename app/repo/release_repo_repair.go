package repo

import (
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	fileutil "github.com/aihop/gopanel/utils/files"
	"gorm.io/gorm"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
	if err := r.db.Model(&model.Release{}).Select("pipeline_record_id, COUNT(*) AS count").Where("pipeline_record_id > 0").Group("pipeline_record_id").Having("COUNT(*) > 1").Find(&groups).Error; err != nil {
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
		if err := tx.Model(&model.Release{}).Where("id = ?", keeper.ID).Updates(map[string]interface{}{"version": merged.Version, "commit_hash": merged.CommitHash, "changelog": merged.Changelog, "source_type": merged.SourceType, "image_tag": merged.ImageTag, "archive_file": merged.ArchiveFile, "release_dir": merged.ReleaseDir, "artifact_meta": merged.ArtifactMeta, "status": merged.Status, "remark": merged.Remark}).Error; err != nil {
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
	if strings.TrimSpace(dst.Changelog) == "" && strings.TrimSpace(src.Changelog) != "" {
		dst.Changelog = src.Changelog
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
