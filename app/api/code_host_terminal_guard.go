package api

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

var codeHostTerminalLifecycle sync.Mutex
var codeRepositoryOperationPaths = make(map[string]int)

func validateHostTerminalDevelopmentOpen(workDir string) error {
	var count int64
	if err := global.DB.Model(&model.AIDevSession{}).Where("status = ?", codeSessionStatusDelivering).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("当前有 Code 会话正在统一交付，交付完成前不能打开宿主终端")
	}
	workDir = filepath.Clean(workDir)
	for path := range codeRepositoryOperationPaths {
		if workDir == path || isPathInside(workDir, path) || isPathInside(path, workDir) {
			return errors.New("当前项目主仓正在同步或创建隔离会话，请稍后再打开终端")
		}
	}
	return nil
}

func beginCodeRepositoryOperation(project *model.AIProject) bool {
	codeHostTerminalLifecycle.Lock()
	defer codeHostTerminalLifecycle.Unlock()
	if codeProjectHasActiveTerminal(project) {
		return false
	}
	for _, path := range codeProjectOperationPaths(project) {
		path = filepath.Clean(path)
		codeRepositoryOperationPaths[path]++
	}
	return true
}

func endCodeRepositoryOperation(project *model.AIProject) {
	codeHostTerminalLifecycle.Lock()
	defer codeHostTerminalLifecycle.Unlock()
	for _, path := range codeProjectOperationPaths(project) {
		path = filepath.Clean(path)
		if codeRepositoryOperationPaths[path] <= 1 {
			delete(codeRepositoryOperationPaths, path)
		} else {
			codeRepositoryOperationPaths[path]--
		}
	}
}

func codeProjectOperationPaths(project *model.AIProject) []string {
	paths := append([]string(nil), codeProjectSourceDirs(project)...)
	if project != nil && strings.TrimSpace(project.WorkDir) != "" {
		paths = append(paths, project.WorkDir)
	}
	return paths
}

func stopHostTerminalsForCodeDelivery(ctx context.Context, userID uint) error {
	stopContext, cancel := context.WithTimeout(ctx, codeDeliveryQueueTimeout)
	defer cancel()
	return hostTerminals.stopAllForCodeDelivery(stopContext, userID)
}

func (manager *hostTerminalManager) stopAllForCodeDelivery(ctx context.Context, userID uint) error {
	manager.mu.Lock()
	ids := make([]uint, 0, len(manager.sessions))
	for id := range manager.sessions {
		ids = append(ids, id)
	}
	manager.mu.Unlock()
	for _, id := range ids {
		if !manager.stopAndWait(ctx, id) && manager.get(id) != nil {
			return fmt.Errorf("统一交付前无法停止宿主终端会话 %d", id)
		}
		recordHostTerminalAudit(id, userID, "delivery_stop", "success", "", "统一交付前自动停止宿主终端")
	}
	return nil
}
