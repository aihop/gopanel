package service

import (
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	fileutil "github.com/aihop/gopanel/utils/files"
	"os"
	"path/filepath"
	"strings"
)

func normalizePipelineKey(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
func (s *PipelineApplicationService) validatePipelineKey(pipelineKey string, excludeID uint, currentPipelineKey string) error {
	if pipelineKey == "" {
		return errors.New("流水线标识不能为空")
	}
	exists, err := s.pipelineRepo.ExistsPipelineKey(pipelineKey, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("流水线标识 `%s` 已存在，请换一个", pipelineKey)
	}
	pipelineDir := filepath.Join(global.CONF.System.BaseDir, "pipelines", pipelineKey)
	if _, err := os.Stat(pipelineDir); err == nil && strings.TrimSpace(currentPipelineKey) != pipelineKey {
		return fmt.Errorf("流水线目录 `%s` 已存在，流水线标识重复了，请换其他的", pipelineDir)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	appDir := filepath.Join(global.CONF.System.BaseDir, "apps", pipelineKey)
	if _, err := os.Stat(appDir); err == nil && strings.TrimSpace(currentPipelineKey) != pipelineKey {
		return fmt.Errorf("安装目录 `%s` 已存在，流水线标识重复了，请换其他的", appDir)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
func (s *PipelineApplicationService) fillReleasedFlags(records []model.PipelineRecord) error {
	if len(records) == 0 {
		return nil
	}
	recordIDs := make([]uint, 0, len(records))
	for _, item := range records {
		if item.ID > 0 {
			recordIDs = append(recordIDs, item.ID)
		}
	}
	releasedMap, err := s.releaseRepo.ExistsByPipelineRecordIDs(recordIDs)
	if err != nil {
		return err
	}
	for i := range records {
		records[i].Released = releasedMap[records[i].ID]
	}
	return nil
}
func snapshotPipelineReleaseDir(pipeline *model.Pipeline, record *model.PipelineRecord, src string) (string, error) {
	src = strings.TrimSpace(src)
	if pipeline == nil || record == nil {
		return "", fmt.Errorf("发布版本失败：缺少流水线或构建记录信息")
	}
	if src == "" {
		return "", fmt.Errorf("发布版本失败：缺少可固化的发布目录")
	}
	info, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("读取发布目录失败: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("发布目录不存在: %s", src)
	}
	snapshotDir := filepath.Join(pipelineArchiveDir(pipeline), fmt.Sprintf("release-record-%d", record.ID))
	if err := os.RemoveAll(snapshotDir); err != nil {
		return "", fmt.Errorf("清理历史版本快照失败: %w", err)
	}
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return "", fmt.Errorf("创建版本快照目录失败: %w", err)
	}
	if err := fileutil.CopyDirContents(src, snapshotDir); err != nil {
		return "", fmt.Errorf("固化版本目录失败: %w", err)
	}
	return snapshotDir, nil
}
func isReleasePipelineRecordDuplicate(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}
