package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type codeRepositoryDeliveryBaseline struct {
	TargetBranch string
	SourceCommit string
	RemoteCommit string
}

func prepareCodeMultiRepositoryDeliveryWithProgress(
	session *model.AIDevSession,
	report codeDeliveryProgressReporter,
) (codeGitDeliveryResult, error) {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return codeGitDeliveryResult{}, errors.New("会话多仓库 Worktree 元数据不可用")
	}
	if _, err := codeMultiRepositoryWorkspaceDir(session, repositories); err != nil {
		return codeGitDeliveryResult{}, err
	}
	if _, err := codeDeliveryRepositoriesInOrder(repositories, true); err != nil {
		return codeGitDeliveryResult{}, err
	}
	if codeMultiRepositoryNeedsSnapshot(repositories) {
		if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
			return codeGitDeliveryResult{}, err
		}
		repositories, err = loadCodeSessionRepositories(session.ID)
		if err != nil {
			return codeGitDeliveryResult{}, err
		}
	}
	if report != nil {
		report(codeDeliveryStageSyncing, 20)
	}
	baselines := make([]codeRepositoryDeliveryBaseline, len(repositories))
	rebuild := make([]bool, len(repositories))
	for index := range repositories {
		repository := &repositories[index]
		if repository.Status == codeDeliveryCompleted {
			continue
		}
		if err := validateCodeRepositorySnapshotCommit(repository); err != nil {
			return codeGitDeliveryResult{}, err
		}
		baseline, inspectErr := inspectCodeRepositoryDeliveryBaseline(repository)
		if inspectErr != nil {
			return codeGitDeliveryResult{}, inspectErr
		}
		baselines[index] = baseline
		rebuild[index] = !codeRepositoryIntegrationReusable(repository, baseline)
	}
	propagateCodeRepositoryIntegrationRebuild(repositories, rebuild)
	if err := persistCodeRepositoryDeliveryBaselines(repositories, baselines, rebuild); err != nil {
		return codeGitDeliveryResult{}, err
	}
	rebuildByID := make(map[uint]bool, len(repositories))
	for index := range repositories {
		rebuildByID[repositories[index].ID] = rebuild[index]
	}
	working, err := codeDeliveryRepositoriesInOrder(repositories, true)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	neededByID := make(map[uint]bool, len(working))
	bySource := make(map[string]*model.AIDevSessionRepository, len(working))
	for index := range working {
		bySource[filepath.Clean(working[index].SourceDir)] = &working[index]
		needed := rebuildByID[working[index].ID]
		if parent := bySource[filepath.Clean(strings.TrimSpace(working[index].ParentSourceDir))]; parent != nil {
			needed = needed || neededByID[parent.ID]
		}
		neededByID[working[index].ID] = needed
		if !needed {
			continue
		}
		if err := ensureCodeRepositoryIntegrationWorktree(session, &working[index], working); err != nil {
			return codeGitDeliveryResult{}, err
		}
		if working[index].Status == codeDeliveryCompleted {
			if err := materializeCompletedCodeRepositoryIntegration(&working[index]); err != nil {
				return codeGitDeliveryResult{}, err
			}
		}
	}
	working, err = codeDeliveryRepositoriesInOrder(working, false)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	results := make([]codeRepositoryDeliveryResult, 0, len(working))
	hasConflicts := false
	for index := range working {
		repository := &working[index]
		if repository.Status == codeDeliveryCompleted || !rebuildByID[repository.ID] {
			results = append(results, codeStoredRepositoryDeliveryResult(repository))
			continue
		}
		if codeRepositoryHasConflictedDescendant(repository, working) {
			results = append(results, codeStoredRepositoryDeliveryResult(repository))
			continue
		}
		if report != nil {
			report(codeDeliveryStageMerging, 35+(index*20/max(1, len(working))))
		}
		result, mergeErr := prepareCodeRepositoryIntegration(session, repository, working)
		results = append(results, result)
		if mergeErr != nil {
			return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(working), mergeErr), mergeErr
		}
		if result.Status == codeDeliveryJobConflict {
			hasConflicts = true
		}
	}
	if hasConflicts {
		return codeMultiRepositoryConflictResult(results), nil
	}
	return codeGitDeliveryResult{
		Status: codeDeliveryMerged, ResultType: codeMultiRepositoryResultType(results), Repositories: results,
	}, nil
}

func codeRepositoryHasConflictedDescendant(
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) bool {
	for index := range repositories {
		child := &repositories[index]
		if filepath.Clean(child.ParentSourceDir) != filepath.Clean(repository.SourceDir) {
			continue
		}
		if child.Status == codeDeliveryJobConflict || codeRepositoryHasConflictedDescendant(child, repositories) {
			return true
		}
	}
	return false
}

func codeMultiRepositoryNeedsSnapshot(repositories []model.AIDevSessionRepository) bool {
	for _, repository := range repositories {
		switch repository.Status {
		case codeDeliveryPrepared, codeDeliveryMerged, codeDeliveryCompleted:
		default:
			return true
		}
	}
	return false
}

func validateCodeRepositorySnapshotCommit(repository *model.AIDevSessionRepository) error {
	commit := strings.TrimSpace(repository.WorktreeCommit)
	if commit == "" {
		return fmt.Errorf("仓库 %s 的交付快照不可用", repository.LinkName)
	}
	if _, err := runCodeGit(repository.SourceDir, "cat-file", "-e", commit+"^{commit}"); err != nil {
		return fmt.Errorf("仓库 %s 的交付快照提交不存在", repository.LinkName)
	}
	return nil
}

// inspectCodeRepositoryDeliveryBaseline 只读取交付基线所需的 ref。
// 合并发生在独立的集成 Worktree 中，不触碰源仓工作区，
// 因此源仓是否有未提交变更、当前签出的是哪个分支，都不影响本次交付能否进行。
func inspectCodeRepositoryDeliveryBaseline(repository *model.AIDevSessionRepository) (codeRepositoryDeliveryBaseline, error) {
	targetBranch := strings.TrimSpace(repository.TargetBranch)
	if targetBranch == "" {
		currentBranch, err := runCodeGit(repository.SourceDir, "branch", "--show-current")
		if err != nil {
			return codeRepositoryDeliveryBaseline{}, err
		}
		targetBranch = strings.TrimSpace(currentBranch)
	}
	if targetBranch == "" {
		return codeRepositoryDeliveryBaseline{}, fmt.Errorf("源仓库 %s 没有可用的交付目标分支", repository.LinkName)
	}
	sourceCommit, err := runCodeGit(repository.SourceDir, "rev-parse", "refs/heads/"+targetBranch)
	if err != nil {
		return codeRepositoryDeliveryBaseline{}, fmt.Errorf("源仓库 %s 的交付目标分支 %s 不可用", repository.LinkName, targetBranch)
	}
	baseline := codeRepositoryDeliveryBaseline{
		TargetBranch: targetBranch, SourceCommit: strings.TrimSpace(sourceCommit),
		RemoteCommit: strings.TrimSpace(repository.RemoteCommit),
	}
	return baseline, nil
}

func codeRepositoryIntegrationBaseCommit(sourceCommit string) (string, error) {
	sourceCommit = strings.TrimSpace(sourceCommit)
	if sourceCommit == "" {
		return "", errors.New("源分支提交不可用")
	}
	return sourceCommit, nil
}

func codeRepositoryIntegrationReusable(
	repository *model.AIDevSessionRepository,
	baseline codeRepositoryDeliveryBaseline,
) bool {
	if repository.Status != codeDeliveryMerged || strings.TrimSpace(repository.MergeCommit) == "" ||
		strings.TrimSpace(repository.IntegrationWorkDir) == "" || repository.TargetBranch != baseline.TargetBranch ||
		repository.SourceCommit != baseline.SourceCommit || repository.RemoteCommit != baseline.RemoteCommit {
		return false
	}
	return verifyCodeDeliveryCommit(
		repository.IntegrationWorkDir, repository.MergeCommit, "仓库 "+repository.LinkName+" 的最终集成提交",
	) == nil
}

func propagateCodeRepositoryIntegrationRebuild(repositories []model.AIDevSessionRepository, rebuild []bool) {
	bySource := make(map[string]int, len(repositories))
	for index := range repositories {
		bySource[filepath.Clean(repositories[index].SourceDir)] = index
	}
	for changed := true; changed; {
		changed = false
		for index := range repositories {
			if !rebuild[index] || strings.TrimSpace(repositories[index].ParentSourceDir) == "" {
				continue
			}
			parent, exists := bySource[filepath.Clean(repositories[index].ParentSourceDir)]
			if exists && !rebuild[parent] {
				rebuild[parent], changed = true, true
			}
		}
	}
}

func persistCodeRepositoryDeliveryBaselines(
	repositories []model.AIDevSessionRepository,
	baselines []codeRepositoryDeliveryBaseline,
	rebuild []bool,
) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		for index := range repositories {
			if repositories[index].Status == codeDeliveryCompleted || !rebuild[index] {
				continue
			}
			baseline := baselines[index]
			updates := map[string]any{
				"status": codeDeliveryPrepared, "target_branch": baseline.TargetBranch,
				"source_commit": baseline.SourceCommit, "remote_commit": baseline.RemoteCommit,
				"merge_commit": "", "merged_at": nil, "publish_started_at": nil,
				"source_applied_at": nil, "completed_at": nil, "error_message": "",
				"push_status": codePushPending, "pushed_commit": "", "pushed_at": nil, "push_error": "",
			}
			if err := tx.Model(&repositories[index]).Updates(updates).Error; err != nil {
				return err
			}
			repositories[index].Status = codeDeliveryPrepared
			repositories[index].TargetBranch = baseline.TargetBranch
			repositories[index].SourceCommit = baseline.SourceCommit
			repositories[index].RemoteCommit = baseline.RemoteCommit
			repositories[index].MergeCommit = ""
			repositories[index].MergedAt, repositories[index].PublishStartedAt = nil, nil
			repositories[index].SourceAppliedAt, repositories[index].CompletedAt = nil, nil
			repositories[index].PushStatus, repositories[index].PushedCommit = codePushPending, ""
			repositories[index].PushedAt, repositories[index].PushError = nil, ""
		}
		return nil
	})
}

func codeMultiRepositoryConflictResult(
	results []codeRepositoryDeliveryResult,
) codeGitDeliveryResult {
	conflicts := make([]string, 0)
	for index := range results {
		if results[index].Status == codeDeliveryJobConflict {
			conflicts = append(conflicts, results[index].ConflictFiles...)
		}
	}
	return codeGitDeliveryResult{
		Status: codeDeliveryJobConflict, ConflictFiles: normalizedCodeConflictFiles(conflicts), Repositories: results,
	}
}
