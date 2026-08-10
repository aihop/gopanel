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
}

func publishCodeMultiRepositoryDeliveryWithProgress(
	session *model.AIDevSession,
	report codeDeliveryProgressReporter,
) (codeGitDeliveryResult, error) {
	repositories, err := loadCodeSessionRepositories(session.ID)
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
		if err := fastForwardCodeMultiRepositorySource(&repositories[index], repositories); err != nil {
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
	if err != nil || strings.TrimSpace(branch) != repository.TargetBranch {
		return false, fmt.Errorf("源仓库 %s 的交付目标分支已变化", repository.LinkName)
	}
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
	if err := validateCodeMultiRepositorySourceStatus(repository, repositories); err != nil {
		return false, err
	}
	return false, nil
}

func validateCodeMultiRepositorySourceStatus(
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) error {
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
	_, err := runCodeGitWithCredential(
		repository.SourceDir, codeGitFetchTimeout, codeProjectGitCredentialID(session.ProjectID),
		"-c", "credential.interactive=never", "push", "--", repository.RemoteName,
		repository.MergeCommit+":refs/heads/"+state.RemoteBranch,
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
		if resolveErr == nil && strings.TrimSpace(current) != state.RemoteCommit {
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
		if _, err := runCodeGit(repository.SourceDir, "merge", "--ff-only", repository.MergeCommit); err != nil {
			return fmt.Errorf("源仓库 %s 无法安全快进：%w", repository.LinkName, err)
		}
	}
	updated, err := runCodeGit(repository.SourceDir, "rev-parse", "HEAD")
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

func completeCodeMultiRepositorySources(repositories []model.AIDevSessionRepository) error {
	for index := range repositories {
		if repositories[index].Status != codeDeliveryCompleted && repositories[index].SourceAppliedAt == nil {
			return fmt.Errorf("源仓库 %s 尚未应用最终提交", repositories[index].LinkName)
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
