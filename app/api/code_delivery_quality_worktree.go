package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

type codeDeliveryQualityWorktree struct {
	SourceDir string
	WorkDir   string
}

func prepareCodeMultiRepositoryQualityRoots(session *model.AIDevSession) ([]codeDeliveryQualityRoot, func() error, error) {
	if session == nil || session.IsolationMode != codeIsolationMultiWorktree {
		return nil, nil, errors.New("多仓库交付会话不可用")
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return nil, nil, errors.New("会话多仓库交付快照不可用")
	}
	rootDir := aiProjectWorktreeRoot(session.UserID)
	if err := os.MkdirAll(rootDir, 0750); err != nil {
		return nil, nil, err
	}
	workspaceDir, err := os.MkdirTemp(rootDir, fmt.Sprintf("quality_%d_", session.ID))
	if err != nil {
		return nil, nil, err
	}
	created := make([]codeDeliveryQualityWorktree, 0, len(repositories))
	cleanup := func() error {
		var cleanupErr error
		for index := len(created) - 1; index >= 0; index-- {
			worktree := created[index]
			if _, err := runCodeGit(worktree.SourceDir, "worktree", "remove", "--force", worktree.WorkDir); err != nil {
				cleanupErr = errors.Join(cleanupErr, err)
			}
		}
		if err := os.RemoveAll(workspaceDir); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
		return cleanupErr
	}
	fail := func(err error) ([]codeDeliveryQualityRoot, func() error, error) {
		return nil, nil, errors.Join(err, cleanup())
	}

	roots := make([]codeDeliveryQualityRoot, 0, len(repositories))
	for _, repository := range repositories {
		if repository.Status == codeDeliveryCompleted {
			continue
		}
		commit := strings.TrimSpace(repository.WorktreeCommit)
		if commit == "" {
			return fail(fmt.Errorf("仓库 %s 的交付快照不可用", repository.LinkName))
		}
		workDir := filepath.Join(workspaceDir, repository.LinkName)
		if filepath.Dir(workDir) != workspaceDir || filepath.Base(workDir) != repository.LinkName {
			return fail(errors.New("质量检查 Worktree 目录无效"))
		}
		if _, err := runCodeGit(repository.SourceDir, "worktree", "add", "--detach", workDir, commit); err != nil {
			return fail(err)
		}
		created = append(created, codeDeliveryQualityWorktree{SourceDir: repository.SourceDir, WorkDir: workDir})
		if err := installCodeManagedPushGuard(workDir); err != nil {
			return fail(err)
		}
		roots = append(roots, codeDeliveryQualityRoot{
			WorkDir: workDir, IdentityDir: repository.WorktreeDir,
			Commit: commit, Label: "仓库 " + repository.LinkName,
		})
	}
	return roots, cleanup, nil
}
