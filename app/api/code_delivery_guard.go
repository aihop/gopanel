package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

const codeManagedPrePushHook = `#!/bin/sh
echo "GoPanel: 当前为隔离开发 Worktree，请在 Code 工作区使用统一交付，不要直接 git push。" >&2
exit 1
`

func installCodeManagedPushGuard(worktreeDir string) error {
	worktreeDir = strings.TrimSpace(worktreeDir)
	if worktreeDir == "" {
		return errors.New("会话 Worktree 目录不可用")
	}
	gitDir, err := resolveCodeGitPath(worktreeDir, "--git-dir")
	if err != nil {
		return err
	}
	hooksDir := filepath.Join(gitDir, "gopanel-hooks")
	if err := os.MkdirAll(hooksDir, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-push"), []byte(codeManagedPrePushHook), 0700); err != nil {
		return err
	}
	if _, err := runCodeGit(worktreeDir, "config", "extensions.worktreeConfig", "true"); err != nil {
		return err
	}
	_, err = runCodeGit(worktreeDir, "config", "--worktree", "core.hooksPath", hooksDir)
	return err
}

func ensureCodeManagedPushGuards(session *model.AIDevSession) error {
	if session == nil {
		return nil
	}
	if session.IsolationMode == codeIsolationMultiWorktree {
		repositories, err := loadCodeSessionRepositories(session.ID)
		if err != nil {
			return err
		}
		for _, repository := range repositories {
			if err := installCodeManagedPushGuard(repository.WorktreeDir); err != nil {
				return err
			}
		}
		return nil
	}
	if strings.TrimSpace(session.WorktreeBranch) == "" {
		return nil
	}
	return installCodeManagedPushGuard(session.WorkDir)
}
