package api

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aihop/gopanel/app/model"
)

var codeHostTerminalLifecycle sync.Mutex
var codeRepositoryOperationPaths = make(map[string]int)
var errCodeProjectActiveTerminal = errors.New("项目主仓正在被项目终端使用，请关闭终端后再操作")

// 交付在独立的集成 Worktree 中进行，不独占源仓工作区，
// 因此交付进行中照常可以打开宿主终端；本地快进失败只会被降级记录。
func validateHostTerminalDevelopmentOpen(workDir string) error {
	workDir = filepath.Clean(workDir)
	for path := range codeRepositoryOperationPaths {
		if workDir == path || isPathInside(workDir, path) || isPathInside(path, workDir) {
			return errors.New("当前项目主仓正在同步或创建隔离会话，请稍后再打开终端")
		}
	}
	return nil
}

func beginCodeRepositoryOperation(project *model.AIProject) error {
	codeHostTerminalLifecycle.Lock()
	defer codeHostTerminalLifecycle.Unlock()
	hasActiveTerminal, err := codeProjectHasActiveTerminal(project)
	if err != nil {
		return err
	}
	if hasActiveTerminal {
		return errCodeProjectActiveTerminal
	}
	for _, path := range codeProjectOperationPaths(project) {
		path = filepath.Clean(path)
		codeRepositoryOperationPaths[path]++
	}
	return nil
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
