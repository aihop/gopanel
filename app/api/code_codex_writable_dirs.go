package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
)

func codexWritableDirsForSession(session *model.AIDevSession) ([]string, error) {
	if session == nil || session.ProjectID == 0 || session.SourceWorkDir != "" || session.WorktreeBranch != "" {
		return nil, nil
	}
	project, err := repo.NewAIGroupRepo().GetGroupByID(session.ProjectID)
	if err != nil {
		return nil, err
	}
	return resolveCodexWritableDirs(project.SourceDirs)
}

func resolveCodexWritableDirs(sourceDirs []string) ([]string, error) {
	seen := make(map[string]struct{}, len(sourceDirs))
	resolvedDirs := make([]string, 0, len(sourceDirs))
	for _, sourceDir := range sourceDirs {
		sourceDir = strings.TrimSpace(sourceDir)
		if sourceDir == "" || !filepath.IsAbs(sourceDir) {
			return nil, errors.New("项目源目录必须是有效的绝对目录")
		}
		resolvedDir, err := filepath.EvalSymlinks(filepath.Clean(sourceDir))
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(resolvedDir)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, errors.New("项目源路径不是目录")
		}
		if _, exists := seen[resolvedDir]; exists {
			continue
		}
		seen[resolvedDir] = struct{}{}
		resolvedDirs = append(resolvedDirs, resolvedDir)
	}
	return resolvedDirs, nil
}

func addCodexWritableDirArgs(args, writableDirs []string) []string {
	if len(writableDirs) == 0 {
		return args
	}
	insertionIndex := len(args)
	for index, arg := range args {
		if arg == "exec" || arg == "resume" {
			insertionIndex = index
			break
		}
	}
	extraArgs := make([]string, 0, len(writableDirs)*2)
	for _, writableDir := range writableDirs {
		extraArgs = append(extraArgs, "--add-dir", writableDir)
	}
	result := make([]string, 0, len(args)+len(extraArgs))
	result = append(result, args[:insertionIndex]...)
	result = append(result, extraArgs...)
	result = append(result, args[insertionIndex:]...)
	return result
}
