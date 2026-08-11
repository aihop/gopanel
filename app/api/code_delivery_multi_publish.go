package api

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type codeMultiRepositoryPublishState struct {
	RemoteBranch  string
	RemoteCommit  string
	AlreadyPushed bool
	// Isolated 表示推送目标是本会话独占的交付分支。
	Isolated bool
	// ForceLease 非空时用 --force-with-lease 覆盖自己上次推送的提交，
	// 值为期望的远端当前提交，别人动过分支就会被 Git 拒绝。
	ForceLease string
}

func publishCodeMultiRepositoryDeliveryWithProgress(
	session *model.AIDevSession,
	report codeDeliveryProgressReporter,
) (codeGitDeliveryResult, error) {
	repositories, err := loadCodeDeliverySessionRepositories(session)
	if err != nil || len(repositories) == 0 {
		return codeGitDeliveryResult{}, errors.New("会话多仓库交付记录不可用")
	}
	repositories, err = codeDeliveryRepositoriesInOrder(repositories, false)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if codeMultiRepositoryAllCompleted(repositories) {
		return finishCodeMultiRepositoryDelivery(session, repositories)
	}
	if report != nil {
		report(codeDeliveryStageCleaning, 70)
	}
	for index := range repositories {
		if repositories[index].Status == codeDeliveryCompleted {
			continue
		}
		if _, err := validateCodeMultiRepositorySource(&repositories[index], repositories); err != nil {
			return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), err), err
		}
	}
	for index := range repositories {
		if repositories[index].Status == codeDeliveryCompleted {
			continue
		}
		if err := applyCodeRepositoryLocalSync(&repositories[index], repositories); err != nil {
			return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), err), err
		}
		if repositories[index].PushStatus != codePushPushed &&
			codeDeliveryHasRemote(repositories[index].RemoteName, deliveryRemoteBranch(repositories[index].RemoteBranch, repositories[index].TargetBranch)) {
			if err := persistCodeRepositoryPushResult(&repositories[index], codeRepositoryPushResult{Status: codePushPending}); err != nil {
				return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), err), err
			}
		}
	}
	if err := completeCodeMultiRepositorySources(repositories); err != nil {
		return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), err), err
	}
	return finishCodeMultiRepositoryDelivery(session, repositories)
}

func validateCodeMultiRepositorySource(
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) (bool, error) {
	branch, err := runCodeGit(repository.SourceDir, "branch", "--show-current")
	if err != nil {
		return false, fmt.Errorf("源仓库 %s 的当前分支不可用", repository.LinkName)
	}
	branch = strings.TrimSpace(branch)
	commit, err := runCodeGit(repository.SourceDir, "rev-parse", "refs/heads/"+repository.TargetBranch)
	if err != nil {
		return false, fmt.Errorf("源仓库 %s 的交付目标提交不可用", repository.LinkName)
	}
	commit = strings.TrimSpace(commit)
	if _, err := runCodeGit(repository.SourceDir, "merge-base", "--is-ancestor", repository.MergeCommit, commit); err == nil {
		return true, nil
	}
	if commit != strings.TrimSpace(repository.SourceCommit) {
		return false, fmt.Errorf("源仓库 %s 在交付期间已推进", repository.LinkName)
	}
	if branch == repository.TargetBranch {
		if err := validateCodeMultiRepositorySourceStatus(repository, repositories); err != nil {
			return false, err
		}
	}
	if _, err := runCodeGit(repository.SourceDir, "merge-base", "--is-ancestor", commit, repository.MergeCommit); err != nil {
		return false, fmt.Errorf("仓库 %s 的交付提交无法从目标分支安全快进", repository.LinkName)
	}
	if branch != repository.TargetBranch {
		return false, nil
	}
	return false, nil
}

func validateCodeMultiRepositorySourceStatus(
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) error {
	changedPaths, err := codeMultiRepositoryDeliveryChangedPaths(repository)
	if err != nil {
		return err
	}
	status, err := runCodeGitBytes(
		repository.SourceDir, nil, "status", "--porcelain=v1", "-z", "--ignore-submodules=dirty",
	)
	if err != nil {
		return err
	}
	if len(status) == 0 {
		return nil
	}
	managed := make(map[string]struct{})
	for index := range repositories {
		child := &repositories[index]
		if filepath.Clean(child.ParentSourceDir) != filepath.Clean(repository.SourceDir) || strings.TrimSpace(child.GitlinkPath) == "" {
			continue
		}
		path, pathErr := codeRepositoryGitlinkPath(child)
		if pathErr != nil {
			return pathErr
		}
		managed[path] = struct{}{}
	}
	for _, entry := range bytes.Split(status, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		if len(entry) >= 3 && entry[0] == '?' && entry[1] == '?' {
			path := filepath.ToSlash(string(entry[3:]))
			if codeRepositoryPathOverlaps(path, changedPaths) {
				return fmt.Errorf("源仓库 %s 的未跟踪文件会被交付覆盖：%s", repository.LinkName, path)
			}
			continue
		}
		if len(entry) < 4 || entry[0] != ' ' || entry[1] != 'M' {
			return fmt.Errorf("源仓库 %s 在交付期间出现未提交变更", repository.LinkName)
		}
		path := filepath.ToSlash(string(entry[3:]))
		if _, allowed := managed[path]; !allowed {
			return fmt.Errorf("源仓库 %s 在交付期间出现未提交变更", repository.LinkName)
		}
		if err := validateCodeMultiRepositoryGitlinkTransition(repository, repositories, path); err != nil {
			return err
		}
	}
	return nil
}

func codeMultiRepositoryDeliveryChangedPaths(repository *model.AIDevSessionRepository) ([]string, error) {
	output, err := runCodeGitBytes(
		repository.SourceDir, nil, "diff", "--name-only", "-z",
		repository.SourceCommit, repository.MergeCommit,
	)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, path := range bytes.Split(output, []byte{0}) {
		if len(path) > 0 {
			paths = append(paths, filepath.ToSlash(string(path)))
		}
	}
	return paths, nil
}

func codeRepositoryPathOverlaps(path string, changedPaths []string) bool {
	path = strings.Trim(filepath.ToSlash(path), "/")
	for _, changedPath := range changedPaths {
		changedPath = strings.Trim(filepath.ToSlash(changedPath), "/")
		if path == changedPath || strings.HasPrefix(path, changedPath+"/") || strings.HasPrefix(changedPath, path+"/") {
			return true
		}
	}
	return false
}

func validateCodeMultiRepositoryGitlinkTransition(
	parent *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
	path string,
) error {
	for index := range repositories {
		child := &repositories[index]
		if filepath.Clean(child.ParentSourceDir) != filepath.Clean(parent.SourceDir) ||
			filepath.ToSlash(filepath.Clean(child.GitlinkPath)) != path {
			continue
		}
		entry, err := runCodeGit(parent.SourceDir, "ls-tree", parent.MergeCommit, "--", path)
		if err != nil || !strings.Contains(entry, "160000 commit "+child.MergeCommit+"\t"+path) {
			return fmt.Errorf("源仓库 %s 的最终子仓库指针不可用", parent.LinkName)
		}
		childHead, err := runCodeGit(child.SourceDir, "rev-parse", "HEAD")
		if err != nil {
			return fmt.Errorf("源仓库 %s 的当前提交不可用", child.LinkName)
		}
		childHead = strings.TrimSpace(childHead)
		if childHead == strings.TrimSpace(child.SourceCommit) {
			return nil
		}
		if _, err := runCodeGit(child.SourceDir, "merge-base", "--is-ancestor", child.MergeCommit, childHead); err != nil {
			return fmt.Errorf("源仓库 %s 在交付期间已推进", child.LinkName)
		}
		return nil
	}
	return fmt.Errorf("源仓库 %s 包含未知的子仓库变化", parent.LinkName)
}

func pushCodeMultiRepositoryCommit(
	session *model.AIDevSession,
	repository *model.AIDevSessionRepository,
	state codeMultiRepositoryPublishState,
	report codeDeliveryProgressReporter,
) (codeRepositoryPushResult, error) {
	result := codeRepositoryPushResult{
		Status: codePushPending, Remote: repository.RemoteName, Branch: state.RemoteBranch,
		Commit: repository.MergeCommit, Ready: true,
	}
	if !codeDeliveryHasRemote(repository.RemoteName, state.RemoteBranch) {
		result.Status = "local"
		return result, nil
	}
	if state.AlreadyPushed {
		result.Status = codePushPushed
		return result, nil
	}
	args := []string{"-c", "credential.interactive=never", "push"}
	if state.ForceLease != "" {
		args = append(args, "--force-with-lease=refs/heads/"+state.RemoteBranch+":"+state.ForceLease)
	}
	args = append(args, "--", repository.RemoteName, repository.MergeCommit+":refs/heads/"+state.RemoteBranch)
	_, err := runCodeGitWithCredential(
		repository.SourceDir, codeGitFetchTimeout, codeProjectGitCredentialID(session.ProjectID), args...,
	)
	if err != nil {
		return inspectCodeMultiRepositoryPushFailure(session, repository, state, result, err)
	}
	if report != nil {
		report(codeDeliveryStageVerifying, 85)
	}
	if _, err := fetchCodeRepositoryWithCredential(
		repository.SourceDir, repository.RemoteName, codeProjectGitCredentialID(session.ProjectID),
	); err != nil {
		return failedCodePushResult(result, err)
	}
	pushed, err := runCodeGit(
		repository.SourceDir, "rev-parse", "refs/remotes/"+repository.RemoteName+"/"+state.RemoteBranch,
	)
	if err != nil || strings.TrimSpace(pushed) != repository.MergeCommit {
		return failedCodePushResult(result, errors.New("推送后远端提交核验失败"))
	}
	result.Status = codePushPushed
	return result, nil
}

func inspectCodeMultiRepositoryPushFailure(
	session *model.AIDevSession,
	repository *model.AIDevSessionRepository,
	state codeMultiRepositoryPublishState,
	result codeRepositoryPushResult,
	pushErr error,
) (codeRepositoryPushResult, error) {
	if _, err := fetchCodeRepositoryWithCredential(
		repository.SourceDir, repository.RemoteName, codeProjectGitCredentialID(session.ProjectID),
	); err == nil {
		current, resolveErr := runCodeGit(
			repository.SourceDir, "rev-parse", "refs/remotes/"+repository.RemoteName+"/"+state.RemoteBranch,
		)
		if resolveErr == nil && strings.TrimSpace(current) == repository.MergeCommit {
			result.Status = codePushPushed
			return result, nil
		}
		if resolveErr == nil && !codeRemoteCommitCanFastForward(repository.SourceDir, current, repository.MergeCommit) {
			return failedCodePushResult(result, errCodePushRemoteAdvanced)
		}
	}
	return failedCodePushResult(result, pushErr)
}

func fastForwardCodeMultiRepositorySource(
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) error {
	applied, err := validateCodeMultiRepositorySource(repository, repositories)
	if err != nil {
		return err
	}
	if !applied {
		branch, err := runCodeGit(repository.SourceDir, "branch", "--show-current")
		if err != nil {
			return fmt.Errorf("源仓库 %s 的当前分支不可用", repository.LinkName)
		}
		if strings.TrimSpace(branch) == repository.TargetBranch {
			if _, err := runCodeGit(repository.SourceDir, "merge", "--ff-only", repository.MergeCommit); err != nil {
				return fmt.Errorf("源仓库 %s 无法安全快进：%w", repository.LinkName, err)
			}
		} else if _, err := runCodeGit(
			repository.SourceDir, "update-ref", "refs/heads/"+repository.TargetBranch,
			repository.MergeCommit, repository.SourceCommit,
		); err != nil {
			return fmt.Errorf("源仓库 %s 的目标分支无法安全推进：%w", repository.LinkName, err)
		}
	}
	updated, err := runCodeGit(repository.SourceDir, "rev-parse", "refs/heads/"+repository.TargetBranch)
	if err != nil {
		return fmt.Errorf("源仓库 %s 快进结果核验失败", repository.LinkName)
	}
	if _, err := runCodeGit(repository.SourceDir, "merge-base", "--is-ancestor", repository.MergeCommit, updated); err != nil {
		return fmt.Errorf("源仓库 %s 快进结果核验失败", repository.LinkName)
	}
	return markCodeMultiRepositorySourceApplied(repository)
}

func markCodeMultiRepositorySourceApplied(repository *model.AIDevSessionRepository) error {
	if repository.SourceAppliedAt != nil {
		return nil
	}
	appliedAt := time.Now()
	repository.SourceAppliedAt, repository.ErrorMessage = &appliedAt, ""
	return global.DB.Model(repository).Updates(map[string]any{
		"source_applied_at": appliedAt, "error_message": "",
	}).Error
}

// completeCodeMultiRepositorySources 只有在每个参与仓库的目标分支都真实包含交付提交后才标记完成。
func completeCodeMultiRepositorySources(repositories []model.AIDevSessionRepository) error {
	for index := range repositories {
		repository := &repositories[index]
		if repository.Status == codeDeliveryCompleted {
			continue
		}
		if err := verifyCodeDeliveryCommitExists(
			repository.SourceDir, repository.MergeCommit, "仓库 "+repository.LinkName,
		); err != nil {
			return err
		}
		if repository.SourceAppliedAt == nil {
			return fmt.Errorf("仓库 %s 的交付提交尚未进入本地目标分支", repository.LinkName)
		}
	}
	completedAt := time.Now()
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		for index := range repositories {
			if repositories[index].Status == codeDeliveryCompleted {
				continue
			}
			if err := tx.Model(&repositories[index]).Updates(map[string]any{
				"status": codeDeliveryCompleted, "completed_at": completedAt, "error_message": "",
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	for index := range repositories {
		if repositories[index].Status != codeDeliveryCompleted {
			repositories[index].Status, repositories[index].CompletedAt = codeDeliveryCompleted, &completedAt
		}
	}
	return nil
}

func finishCodeMultiRepositoryDelivery(
	session *model.AIDevSession,
	repositories []model.AIDevSessionRepository,
) (codeGitDeliveryResult, error) {
	results := codeStoredRepositoryDeliveryResults(repositories)
	if !codeMultiRepositoryAllCompleted(repositories) {
		err := errors.New("多仓库交付尚未全部应用到源分支")
		return codeMultiRepositoryFailure(results, err), err
	}
	if err := cleanupCodeMultiRepositoryIntegrationWorktrees(session, repositories); err != nil {
		return codeMultiRepositoryFailure(results, err), err
	}
	return codeGitDeliveryResult{
		Status: codeDeliveryMerged, ResultType: codeMultiRepositoryResultType(results), Repositories: results,
	}, nil
}

func codeStoredRepositoryDeliveryResults(
	repositories []model.AIDevSessionRepository,
) []codeRepositoryDeliveryResult {
	results := make([]codeRepositoryDeliveryResult, 0, len(repositories))
	for index := range repositories {
		results = append(results, codeStoredRepositoryDeliveryResult(&repositories[index]))
	}
	return results
}
