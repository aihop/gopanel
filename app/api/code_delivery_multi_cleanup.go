package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func cleanupCodeMultiRepositoryIntegrationWorktrees(
	session *model.AIDevSession,
	repositories []model.AIDevSessionRepository,
) error {
	if session == nil || session.ID == 0 || len(repositories) == 0 {
		return nil
	}
	root := codeMultiDeliveryRootDir(session)
	if filepath.Clean(root) != filepath.Join(
		aiProjectWorktreeRoot(session.UserID), fmt.Sprintf("delivery_%d_multi", session.ID),
	) {
		return errors.New("多仓库集成交付目录不在 GoPanel 管理目录中")
	}
	ordered, err := codeDeliveryRepositoriesInOrder(repositories, false)
	if err != nil {
		return err
	}
	for index := range ordered {
		repository := &ordered[index]
		workDir := strings.TrimSpace(repository.IntegrationWorkDir)
		if workDir == "" {
			continue
		}
		if !isPathInside(workDir, root) {
			return fmt.Errorf("仓库 %s 的集成交付 Worktree 路径越界", repository.LinkName)
		}
		if _, err := os.Lstat(workDir); errors.Is(err, os.ErrNotExist) {
			if isCodeGitWorktree(repository.SourceDir) {
				if _, pruneErr := runCodeGit(repository.SourceDir, "worktree", "prune", "--expire", "now"); pruneErr != nil {
					return pruneErr
				}
			}
			continue
		} else if err != nil {
			return err
		}
		if !isCodeGitWorktree(repository.SourceDir) {
			if err := os.RemoveAll(workDir); err != nil {
				return err
			}
			continue
		}
		if exists, err := validateExistingCodeIntegrationWorktree(repository, workDir); err != nil {
			return err
		} else if !exists {
			continue
		}
		if _, err := runCodeGit(repository.SourceDir, "worktree", "remove", "--force", workDir); err != nil {
			return fmt.Errorf("清理仓库 %s 的集成交付 Worktree 失败：%w", repository.LinkName, err)
		}
	}
	if info, err := os.Lstat(root); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("多仓库集成交付根目录无效")
		}
		if err := os.RemoveAll(root); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return global.DB.Model(&model.AIDevSessionRepository{}).
		Where("session_id = ?", session.ID).Update("integration_work_dir", "").Error
}
