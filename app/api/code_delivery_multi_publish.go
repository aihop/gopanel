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
	states := make([]codeMultiRepositoryPublishState, len(repositories))
	for index := range repositories {
		if repositories[index].Status == codeDeliveryCompleted {
			continue
		}
		state, inspectErr := inspectCodeMultiRepositoryPublishState(session, &repositories[index], repositories)
		if inspectErr != nil {
			return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), inspectErr), inspectErr
		}
		if state.AlreadyPushed && repositories[index].PushStatus != codePushPushed {
			result := codeRepositoryPushResult{
				Status: codePushPushed, Remote: repositories[index].RemoteName,
				Branch: state.RemoteBranch, Commit: repositories[index].MergeCommit, Ready: true,
			}
			if err := persistCodeRepositoryPushResult(&repositories[index], result); err != nil {
				return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), err), err
			}
		}
		states[index] = state
	}
	if report != nil {
		report(codeDeliveryStagePushing, 70)
	}
	for index := range repositories {
		repository := &repositories[index]
		if repository.Status == codeDeliveryCompleted {
			continue
		}
		result, pushErr := pushCodeMultiRepositoryCommit(session, repository, states[index], report)
		if persistErr := persistCodeRepositoryPushResult(repository, result); persistErr != nil {
			return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), persistErr), persistErr
		}
		if pushErr != nil {
			wrapped := fmt.Errorf("仓库 %s 推送失败：%w", repository.LinkName, pushErr)
			return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), wrapped), wrapped
		}
	}
	for index := range repositories {
		if repositories[index].Status == codeDeliveryCompleted {
			continue
		}
		state, inspectErr := inspectCodeMultiRepositoryPublishState(session, &repositories[index], repositories)
		if inspectErr != nil {
			return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), inspectErr), inspectErr
		}
		if codeDeliveryHasRemote(repositories[index].RemoteName, state.RemoteBranch) && !state.AlreadyPushed {
			return codeMultiRepositoryFailure(
				codeStoredRepositoryDeliveryResults(repositories), errors.New("多仓库推送后的远端提交核验失败"),
			), errors.New("多仓库推送后的远端提交核验失败")
		}
		states[index] = state
	}
	if report != nil {
		report(codeDeliveryStageCleaning, 90)
	}
	parentFirst, err := codeDeliveryRepositoriesInOrder(repositories, true)
	if err != nil {
		return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), err), err
	}
	for index := range parentFirst {
		if parentFirst[index].Status == codeDeliveryCompleted {
			continue
		}
		if err := fastForwardCodeMultiRepositorySource(&parentFirst[index], parentFirst); err != nil {
			syncCodeMultiRepositoryState(repositories, &parentFirst[index])
			return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), err), err
		}
		syncCodeMultiRepositoryState(repositories, &parentFirst[index])
	}
	if err := completeCodeMultiRepositorySources(repositories); err != nil {
		return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(repositories), err), err
	}
	return finishCodeMultiRepositoryDelivery(session, repositories)
}

func inspectCodeMultiRepositoryPublishState(
	session *model.AIDevSession,
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) (codeMultiRepositoryPublishState, error) {
	if (repository.Status != codeDeliveryMerged && repository.Status != codeDeliveryCompleted) ||
		strings.TrimSpace(repository.SourceCommit) == "" ||
		strings.TrimSpace(repository.MergeCommit) == "" || strings.TrimSpace(repository.IntegrationWorkDir) == "" {
		return codeMultiRepositoryPublishState{}, fmt.Errorf("仓库 %s 的最终集成提交不可用", repository.LinkName)
	}
	if err := verifyCodeDeliveryCommit(
		repository.IntegrationWorkDir, repository.MergeCommit, "仓库 "+repository.LinkName+" 的最终集成提交",
	); err != nil {
		return codeMultiRepositoryPublishState{}, err
	}
	if _, err := validateCodeMultiRepositorySource(repository, repositories); err != nil {
		return codeMultiRepositoryPublishState{}, err
	}
	if _, err := runCodeGit(repository.SourceDir, "merge-base", "--is-ancestor", repository.SourceCommit, repository.MergeCommit); err != nil {
		return codeMultiRepositoryPublishState{}, fmt.Errorf("仓库 %s 的最终提交不包含源分支基线", repository.LinkName)
	}
	state := codeMultiRepositoryPublishState{
		RemoteBranch: deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch),
	}
	if !codeDeliveryHasRemote(repository.RemoteName, state.RemoteBranch) {
		return state, nil
	}
	if _, err := fetchCodeRepositoryWithCredential(
		repository.SourceDir, repository.RemoteName, codeProjectGitCredentialID(session.ProjectID),
	); err != nil {
		return state, err
	}
	remoteRef := "refs/remotes/" + repository.RemoteName + "/" + state.RemoteBranch
	remoteCommit, err := runCodeGit(repository.SourceDir, "rev-parse", remoteRef)
	if err != nil {
		return state, errCodePushRemoteAdvanced
	}
	state.RemoteCommit = strings.TrimSpace(remoteCommit)
	if state.RemoteCommit == repository.MergeCommit {
		state.AlreadyPushed = true
		return state, nil
	}
	if state.RemoteCommit != strings.TrimSpace(repository.RemoteCommit) {
		return state, errCodePushRemoteAdvanced
	}
	if _, err := runCodeGit(repository.SourceDir, "merge-base", "--is-ancestor", state.RemoteCommit, repository.MergeCommit); err != nil {
		return state, errCodePushRemoteAdvanced
	}
	return state, nil
}

func validateCodeMultiRepositorySource(
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) (bool, error) {
	if err := validateCodeMultiRepositorySourceStatus(repository, repositories); err != nil {
		return false, err
	}
	branch, err := runCodeGit(repository.SourceDir, "branch", "--show-current")
	if err != nil || strings.TrimSpace(branch) != repository.TargetBranch {
		return false, fmt.Errorf("源仓库 %s 的交付目标分支已变化", repository.LinkName)
	}
	commit, err := runCodeGit(repository.SourceDir, "rev-parse", "refs/heads/"+repository.TargetBranch)
	if err != nil {
		return false, fmt.Errorf("源仓库 %s 的交付目标提交不可用", repository.LinkName)
	}
	commit = strings.TrimSpace(commit)
	if commit == strings.TrimSpace(repository.MergeCommit) {
		return true, nil
	}
	if commit != strings.TrimSpace(repository.SourceCommit) {
		return false, fmt.Errorf("源仓库 %s 在交付期间已推进", repository.LinkName)
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
		childStatus, err := runCodeGit(child.SourceDir, "status", "--porcelain")
		if err != nil || strings.TrimSpace(childStatus) != "" {
			return fmt.Errorf("源仓库 %s 在交付期间出现未提交变更", child.LinkName)
		}
		childHead, err := runCodeGit(child.SourceDir, "rev-parse", "HEAD")
		if err != nil || (strings.TrimSpace(childHead) != strings.TrimSpace(child.SourceCommit) &&
			strings.TrimSpace(childHead) != strings.TrimSpace(child.MergeCommit)) {
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
	remoteRef := "refs/heads/" + state.RemoteBranch
	_, err := runCodeGitWithCredential(
		repository.SourceDir, codeGitFetchTimeout, codeProjectGitCredentialID(session.ProjectID),
		"-c", "credential.interactive=never", "push",
		"--force-with-lease="+remoteRef+":"+state.RemoteCommit,
		"--", repository.RemoteName, repository.MergeCommit+":"+remoteRef,
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
	if err != nil || strings.TrimSpace(updated) != repository.MergeCommit {
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

func syncCodeMultiRepositoryState(
	repositories []model.AIDevSessionRepository,
	updated *model.AIDevSessionRepository,
) {
	for index := range repositories {
		if repositories[index].ID == updated.ID {
			repositories[index] = *updated
			return
		}
	}
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
