package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

func prepareCodeMultiRepositoryQualityRoots(session *model.AIDevSession) ([]codeDeliveryQualityRoot, func() error, error) {
	if session == nil || session.IsolationMode != codeIsolationMultiWorktree {
		return nil, nil, errors.New("多仓库交付会话不可用")
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return nil, nil, errors.New("会话多仓库交付快照不可用")
	}
	if err := restoreCodeMultiRepositoryIntegrationWorktrees(session, repositories); err != nil {
		return nil, nil, err
	}
	repositories, err = loadCodeSessionRepositories(session.ID)
	if err != nil {
		return nil, nil, err
	}
	roots := make([]codeDeliveryQualityRoot, 0, len(repositories))
	for _, repository := range repositories {
		commit := strings.TrimSpace(effectiveCodeRepositoryCommit(&repository))
		workDir := strings.TrimSpace(repository.IntegrationWorkDir)
		if commit == "" || workDir == "" {
			return nil, nil, fmt.Errorf("仓库 %s 的最终集成提交不可用", repository.LinkName)
		}
		roots = append(roots, codeDeliveryQualityRoot{
			WorkDir: workDir, IdentityDir: repository.WorktreeDir,
			Commit: commit, Label: "仓库 " + repository.LinkName,
		})
	}
	return roots, func() error { return nil }, nil
}

func codeMultiRepositoryHasQualityChecks(session *model.AIDevSession) (bool, error) {
	enabled, err := codeDeliveryQualityGateEnabled(session)
	if err != nil || !enabled {
		return false, err
	}
	roots, cleanup, err := prepareCodeMultiRepositoryQualityRoots(session)
	if err != nil {
		return false, err
	}
	hasChecks := len(detectCodeDeliveryQualityChecks(roots)) > 0
	return hasChecks, cleanup()
}
