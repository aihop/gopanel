package api

import (
	"errors"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

func loadCodeMultiRepositoryPushStatus(session *model.AIDevSession) (codePushResult, error) {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return codePushResult{Status: "unavailable", Repositories: []codeRepositoryPushResult{}}, nil
	}
	repositories, err = codeDeliveryRepositoriesInOrder(repositories, false)
	if err != nil {
		return codePushResult{}, err
	}
	results := make([]codeRepositoryPushResult, 0, len(repositories))
	for index := range repositories {
		repository := &repositories[index]
		remoteBranch := deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch)
		if strings.TrimSpace(repository.MergeCommit) == "" || !codeDeliveryHasRemote(repository.RemoteName, remoteBranch) {
			continue
		}
		results = append(results, codeRepositoryPushResult{
			RepositoryID: codeSessionRepositoryID(repository.ID), RepositoryName: repository.LinkName,
			Status: repository.PushStatus, Remote: repository.RemoteName,
			Branch: remoteBranch,
			Commit: repository.MergeCommit, ErrorMessage: repository.PushError,
			Ready: repository.Status == codeDeliveryCompleted && repository.SourceAppliedAt != nil && repository.MergeCommit != "",
		})
	}
	return summarizeCodePushResults(results), nil
}

func pushCodeMultiRepositoryDelivery(session *model.AIDevSession) (codePushResult, error) {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return codePushResult{}, errors.New("会话尚未完成多仓库本地合并交付")
	}
	repositories, err = codeDeliveryRepositoriesInOrder(repositories, false)
	if err != nil {
		return codePushResult{}, err
	}
	repositories = codeMultiRepositoryPushCandidates(repositories)
	if len(repositories) == 0 {
		return codePushResult{}, errors.New("本次交付没有需要推送的远端仓库")
	}
	states := make([]codeMultiRepositoryPublishState, len(repositories))
	for index := range repositories {
		state, inspectErr := inspectCodeMultiRepositoryManualPushState(session, &repositories[index])
		if inspectErr != nil {
			result := codeRepositoryPushResult{Status: codePushFailed, ErrorMessage: inspectErr.Error()}
			_ = persistCodeRepositoryPushResult(&repositories[index], result)
			status, _ := loadCodeMultiRepositoryPushStatus(session)
			return status, inspectErr
		}
		states[index] = state
	}
	for index := range repositories {
		repository := &repositories[index]
		result, pushErr := pushCodeMultiRepositoryCommit(session, repository, states[index], nil)
		result.RepositoryID, result.RepositoryName = codeSessionRepositoryID(repository.ID), repository.LinkName
		if persistErr := persistCodeRepositoryPushResult(repository, result); persistErr != nil {
			return codePushResult{}, persistErr
		}
		if pushErr != nil {
			status, _ := loadCodeMultiRepositoryPushStatus(session)
			return status, pushErr
		}
	}
	return loadCodeMultiRepositoryPushStatus(session)
}

func codeMultiRepositoryPushCandidates(repositories []model.AIDevSessionRepository) []model.AIDevSessionRepository {
	candidates := make([]model.AIDevSessionRepository, 0, len(repositories))
	for index := range repositories {
		remoteBranch := deliveryRemoteBranch(repositories[index].RemoteBranch, repositories[index].TargetBranch)
		if strings.TrimSpace(repositories[index].MergeCommit) != "" &&
			codeDeliveryHasRemote(repositories[index].RemoteName, remoteBranch) {
			candidates = append(candidates, repositories[index])
		}
	}
	return candidates
}

func inspectCodeMultiRepositoryManualPushState(
	session *model.AIDevSession,
	repository *model.AIDevSessionRepository,
) (codeMultiRepositoryPublishState, error) {
	state := codeMultiRepositoryPublishState{
		RemoteBranch: deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch),
	}
	if repository.Status != codeDeliveryCompleted || repository.SourceAppliedAt == nil ||
		strings.TrimSpace(repository.MergeCommit) == "" {
		return state, errors.New("仓库 " + repository.LinkName + " 尚未完成本地合并交付")
	}
	if !codeDeliveryHasRemote(repository.RemoteName, state.RemoteBranch) {
		return state, errors.New("仓库 " + repository.LinkName + " 没有可用的远端跟踪分支")
	}
	localCommit, err := runCodeGit(repository.SourceDir, "rev-parse", "refs/heads/"+repository.TargetBranch)
	if err != nil || strings.TrimSpace(localCommit) != strings.TrimSpace(repository.MergeCommit) {
		return state, errors.New("仓库 " + repository.LinkName + " 的目标分支已在交付后发生变化")
	}
	if _, err := fetchCodeRepositoryWithCredential(
		repository.SourceDir, repository.RemoteName, codeProjectGitCredentialID(session.ProjectID),
	); err != nil {
		return state, err
	}
	remoteCommit, err := runCodeGit(
		repository.SourceDir, "rev-parse", "refs/remotes/"+repository.RemoteName+"/"+state.RemoteBranch,
	)
	if err != nil {
		return state, errCodePushRemoteAdvanced
	}
	state.RemoteCommit = strings.TrimSpace(remoteCommit)
	if state.RemoteCommit == strings.TrimSpace(repository.MergeCommit) {
		state.AlreadyPushed = true
		return state, nil
	}
	if !codeRemoteCommitCanFastForward(repository.SourceDir, state.RemoteCommit, repository.MergeCommit) {
		return state, errCodePushRemoteAdvanced
	}
	return state, nil
}
