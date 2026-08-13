package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/container"
	clienttypes "github.com/docker/docker/client"
	"gorm.io/gorm"
)

type PipelineForceDeleteResult struct {
	PipelineID      uint     `json:"pipelineId"`
	RecordCount     int64    `json:"recordCount"`
	ReleaseCount    int64    `json:"releaseCount"`
	CleanupWarnings []string `json:"cleanupWarnings"`
}

func (s *PipelineApplicationService) ForceDelete(id uint, confirmName string) (*PipelineForceDeleteResult, error) {
	pipelineMutationMu.Lock()
	locked := true
	defer func() {
		if locked {
			pipelineMutationMu.Unlock()
		}
	}()

	pipeline, err := s.pipelineRepo.Get(id)
	if err != nil {
		return nil, err
	}
	if confirmName != pipeline.Name {
		return nil, buserr.New(constant.ErrPipelineForceDeleteName)
	}
	if err := s.ensurePipelineForceDeleteAllowed(id); err != nil {
		return nil, err
	}

	recordCount, err := s.recordRepo.CountByPipelineID(id)
	if err != nil {
		return nil, err
	}
	releaseCount, err := s.releaseRepo.CountByPipelineID(id)
	if err != nil {
		return nil, err
	}
	containerIDs, err := s.recordRepo.RunnerContainerIDsByPipelineID(id)
	if err != nil {
		return nil, err
	}
	recordIDs, err := s.recordRepo.IDsByPipelineID(id)
	if err != nil {
		return nil, err
	}
	if err := s.ensurePipelineContainersUnused(containerIDs); err != nil {
		return nil, err
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("pipeline_id = ?", id).Delete(&model.Release{}).Error; err != nil {
			return err
		}
		if err := tx.Where("pipeline_id = ?", id).Delete(&model.PipelineRecord{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Pipeline{}, id).Error
	})
	if err != nil {
		return nil, err
	}
	pipelineMutationMu.Unlock()
	locked = false

	result := &PipelineForceDeleteResult{PipelineID: id, RecordCount: recordCount, ReleaseCount: releaseCount}
	if err := removePipelineRunnerContainers(containerIDs); err != nil {
		global.LOG.Errorf("Failed to clean Runner containers after deleting pipeline %d: %v", id, err)
		result.CleanupWarnings = append(result.CleanupWarnings, "runner_cleanup_failed")
	}
	if err := removePipelineOwnedDirectories(pipeline); err != nil {
		global.LOG.Errorf("Failed to clean owned directories after deleting pipeline %d: %v", id, err)
		result.CleanupWarnings = append(result.CleanupWarnings, "directory_cleanup_failed")
	}
	if err := removePipelineLogFiles(recordIDs); err != nil {
		global.LOG.Errorf("Failed to clean logs after deleting pipeline %d: %v", id, err)
		if !containsString(result.CleanupWarnings, "directory_cleanup_failed") {
			result.CleanupWarnings = append(result.CleanupWarnings, "directory_cleanup_failed")
		}
	}
	return result, nil
}

func (s *PipelineApplicationService) ensurePipelineForceDeleteAllowed(id uint) error {
	var runningCount int64
	if err := s.db.Model(&model.PipelineRecord{}).
		Where("pipeline_id = ? AND status IN ?", id, []string{"pending", "preparing", "cloning", "building", "deploying"}).
		Count(&runningCount).Error; err != nil {
		return err
	}
	if runningCount > 0 {
		return buserr.New(constant.ErrPipelineForceDeleteRunning)
	}
	var flowCount int64
	if err := s.db.Model(&model.Flow{}).Where("pipeline_id = ?", id).Count(&flowCount).Error; err != nil {
		return err
	}
	if flowCount > 0 {
		return buserr.New(constant.ErrPipelineForceDeleteFlow)
	}
	var flowRunCount int64
	if err := s.db.Model(&model.FlowRun{}).Where("pipeline_id = ?", id).Count(&flowRunCount).Error; err != nil {
		return err
	}
	if flowRunCount > 0 {
		return buserr.New(constant.ErrPipelineForceDeleteHistory)
	}
	return nil
}

func (s *PipelineApplicationService) ensurePipelineContainersUnused(containerIDs []string) error {
	if len(containerIDs) == 0 {
		return nil
	}
	var websites []model.Website
	if err := s.db.Select("id, primary_domain, alias, container_id").Where("container_id IN ?", containerIDs).Find(&websites).Error; err != nil {
		return err
	}
	if len(websites) == 0 {
		return nil
	}
	names := make([]string, 0, len(websites))
	for _, website := range websites {
		name := strings.TrimSpace(website.PrimaryDomain)
		if name == "" {
			name = strings.TrimSpace(website.Alias)
		}
		names = append(names, name)
	}
	return buserr.WithMap(constant.ErrPipelineForceDeleteWebsite, map[string]interface{}{"websites": strings.Join(names, "、")})
}

func removePipelineRunnerContainers(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	cli, err := docker.NewDockerClient()
	if err != nil {
		return fmt.Errorf("流水线已删除，但连接容器运行时失败: %w", err)
	}
	if cli == nil {
		return fmt.Errorf("流水线已删除，但容器运行时不可用")
	}
	defer cli.Close()
	for _, rawID := range ids {
		containerID := strings.TrimSpace(rawID)
		if containerID == "" {
			continue
		}
		if err := cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true}); err != nil && !clienttypes.IsErrNotFound(err) {
			return fmt.Errorf("流水线已删除，但清理 Runner 容器 %s 失败: %w", containerID, err)
		}
	}
	return nil
}

func removePipelineOwnedDirectories(pipeline *model.Pipeline) error {
	pipelinesDir := filepath.Join(global.CONF.System.BaseDir, "pipelines")
	directories := [][2]string{{pipelinesDir, pipelineBaseDir(pipeline)}}
	if key := strings.TrimSpace(pipeline.PipelineKey); key != "" {
		directories = append(directories, [2]string{filepath.Join(global.CONF.System.BaseDir, "apps"), filepath.Join(global.CONF.System.BaseDir, "apps", key)})
	}
	for _, paths := range directories {
		directory, err := safePipelineOwnedDirectory(paths[0], paths[1])
		if err != nil {
			return err
		}
		if directory == filepath.Join(pipelinesDir, "logs") {
			return fmt.Errorf("流水线已删除，但拒绝清理保留日志目录 %s", directory)
		}
		if err := os.RemoveAll(directory); err != nil {
			return fmt.Errorf("流水线记录已删除，但清理目录 %s 失败: %w", directory, err)
		}
	}
	return nil
}

func removePipelineLogFiles(recordIDs []uint) error {
	for _, recordID := range recordIDs {
		if err := os.Remove(getLogFilePath(recordID)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func safePipelineOwnedDirectory(baseDir, targetDir string) (string, error) {
	base, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Clean(targetDir))
	if err != nil {
		return "", err
	}
	if target == base || !strings.HasPrefix(target, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("流水线已删除，但拒绝清理越界目录 %s", target)
	}
	return target, nil
}
