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
		baseline, inspectErr := inspectCodeRepositoryDeliveryBaseline(session, repository)
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
	for index := range working {
		repository := &working[index]
		if repository.Status == codeDeliveryCompleted || !rebuildByID[repository.ID] {
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
			_ = exposeCodeRepositoryIntegrationConflict(repository)
			return codeMultiRepositoryConflictResult(codeStoredRepositoryDeliveryResults(working), result), nil
		}
	}
	return codeGitDeliveryResult{
		Status: codeDeliveryMerged, ResultType: codeMultiRepositoryResultType(results), Repositories: results,
	}, nil
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

func inspectCodeRepositoryDeliveryBaseline(
	session *model.AIDevSession,
	repository *model.AIDevSessionRepository,
) (codeRepositoryDeliveryBaseline, error) {
	status, err := runCodeGit(repository.SourceDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		return codeRepositoryDeliveryBaseline{}, fmt.Errorf("源仓库 %s 存在未提交变更，无法安全交付", repository.LinkName)
	}
	targetBranch := strings.TrimSpace(repository.TargetBranch)
	currentBranch, err := runCodeGit(repository.SourceDir, "branch", "--show-current")
	if err != nil {
		return codeRepositoryDeliveryBaseline{}, err
	}
	if targetBranch == "" {
		targetBranch = strings.TrimSpace(currentBranch)
	}
	if targetBranch == "" || strings.TrimSpace(currentBranch) != targetBranch {
		return codeRepositoryDeliveryBaseline{}, fmt.Errorf("源仓库 %s 当前分支不是交付目标 %s", repository.LinkName, targetBranch)
	}
	sourceCommit, err := runCodeGit(repository.SourceDir, "rev-parse", "refs/heads/"+targetBranch)
	if err != nil {
		return codeRepositoryDeliveryBaseline{}, err
	}
	remoteCommit := ""
	remoteBranch := deliveryRemoteBranch(repository.RemoteBranch, targetBranch)
	if codeDeliveryHasRemote(repository.RemoteName, remoteBranch) {
		if _, err := fetchCodeRepositoryWithCredential(
			repository.SourceDir, repository.RemoteName, codeProjectGitCredentialID(session.ProjectID),
		); err != nil {
			return codeRepositoryDeliveryBaseline{}, fmt.Errorf("仓库 %s 同步失败：%w", repository.LinkName, err)
		}
		remoteCommit, err = runCodeGit(repository.SourceDir, "rev-parse", "refs/remotes/"+repository.RemoteName+"/"+remoteBranch)
		if err != nil {
			return codeRepositoryDeliveryBaseline{}, fmt.Errorf("仓库 %s 的远端目标分支不可用", repository.LinkName)
		}
	}
	baseline := codeRepositoryDeliveryBaseline{
		TargetBranch: targetBranch, SourceCommit: strings.TrimSpace(sourceCommit), RemoteCommit: strings.TrimSpace(remoteCommit),
	}
	if _, err := codeRepositoryIntegrationBaseCommit(repository.SourceDir, baseline.SourceCommit, baseline.RemoteCommit); err != nil {
		return codeRepositoryDeliveryBaseline{}, fmt.Errorf("仓库 %s 无法确定安全集成基线：%w", repository.LinkName, err)
	}
	return baseline, nil
}

func codeRepositoryIntegrationBaseCommit(sourceDir, sourceCommit, remoteCommit string) (string, error) {
	sourceCommit, remoteCommit = strings.TrimSpace(sourceCommit), strings.TrimSpace(remoteCommit)
	if sourceCommit == "" {
		return "", errors.New("源分支提交不可用")
	}
	if remoteCommit == "" || sourceCommit == remoteCommit {
		return sourceCommit, nil
	}
	if _, err := runCodeGit(sourceDir, "merge-base", "--is-ancestor", sourceCommit, remoteCommit); err == nil {
		return remoteCommit, nil
	}
	if _, err := runCodeGit(sourceDir, "merge-base", "--is-ancestor", remoteCommit, sourceCommit); err == nil {
		return sourceCommit, nil
	}
	return "", errors.New("本地目标分支与远端目标分支已分叉")
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

func exposeCodeRepositoryIntegrationConflict(repository *model.AIDevSessionRepository) error {
	status, err := runCodeGit(repository.WorktreeDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		return err
	}
	head, err := runCodeGit(repository.WorktreeDir, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != strings.TrimSpace(repository.WorktreeCommit) {
		return err
	}
	baseCommit, err := codeRepositoryIntegrationBaseCommit(
		repository.SourceDir, repository.SourceCommit, repository.RemoteCommit,
	)
	if err != nil {
		return err
	}
	_, mergeErr := runCodeGit(
		repository.WorktreeDir,
		"-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.local",
		"-c", "commit.gpgsign=false", "merge", "--no-edit", baseCommit,
	)
	if mergeErr != nil && len(codeGitConflictFiles(repository.WorktreeDir)) == 0 {
		_, _ = runCodeGit(repository.WorktreeDir, "merge", "--abort")
	}
	return nil
}

func codeMultiRepositoryConflictResult(
	results []codeRepositoryDeliveryResult,
	conflict codeRepositoryDeliveryResult,
) codeGitDeliveryResult {
	return codeGitDeliveryResult{
		Status:       codeDeliveryJobConflict,
		RepositoryID: conflict.RepositoryID, RepositoryName: conflict.RepositoryName,
		Branch: conflict.Branch, ConflictFiles: conflict.ConflictFiles, Repositories: results,
	}
}
