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
	mode := codeProjectDeliveryMode(session.ProjectID)
	results := make([]codeRepositoryPushResult, 0, len(repositories))
	for index := range repositories {
		repository := &repositories[index]
		remoteBranch := codeDeliveryPushTarget(mode, session.ID, repository.RemoteBranch, repository.TargetBranch)
		if strings.TrimSpace(repository.MergeCommit) == "" || !codeDeliveryHasRemote(repository.RemoteName, remoteBranch) {
			continue
		}
		// 提交可能已经在远端了（手动推的，或推成功但结果没落库），把陈旧的 pending 补正，
		// 否则界面会一直说「未推送」。只查本地的远端跟踪引用，不发网络请求。
		reconcileCodeRepositoryPushStatus(repository, remoteBranch)
		results = append(results, codeRepositoryPushResult{
			RepositoryID: codeSessionRepositoryID(repository.ID), RepositoryName: repository.LinkName,
			Status: repository.PushStatus, Remote: repository.RemoteName,
			Branch: remoteBranch,
			Commit: repository.MergeCommit, ErrorMessage: repository.PushError,
			LocalSynced: repository.SourceAppliedAt != nil, LocalSyncError: repository.LocalSyncError,
			LocalSyncCommand: codeDeliveryLocalSyncCommand(repository.SourceDir, repository.MergeCommit),
			Ready:            repository.Status == codeDeliveryCompleted && strings.TrimSpace(repository.MergeCommit) != "",
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
	mode := codeProjectDeliveryMode(session.ProjectID)
	state := codeMultiRepositoryPublishState{
		RemoteBranch: codeDeliveryPushTarget(mode, session.ID, repository.RemoteBranch, repository.TargetBranch),
		Isolated:     codeDeliveryPushIsolated(mode),
	}
	if repository.Status != codeDeliveryCompleted {
		return state, errors.New("仓库 " + repository.LinkName + " 尚未完成合并交付")
	}
	if !codeDeliveryHasRemote(repository.RemoteName, state.RemoteBranch) {
		return state, errors.New("仓库 " + repository.LinkName + " 没有可用的远端跟踪分支")
	}
	// 推送的是交付提交本身（commit-ish），不依赖本地分支指向它：
	// 本地主仓未能快进时，交付提交仍在共享对象库中，推送照常可用。
	if err := verifyCodeDeliveryCommitExists(
		repository.SourceDir, repository.MergeCommit, "仓库 "+repository.LinkName,
	); err != nil {
		return state, err
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
		// 独占的会话交付分支首次推送时远端还不存在，这是正常的。
		if state.Isolated {
			return state, nil
		}
		return state, errCodePushRemoteAdvanced
	}
	state.RemoteCommit = strings.TrimSpace(remoteCommit)
	if state.RemoteCommit == strings.TrimSpace(repository.MergeCommit) {
		state.AlreadyPushed = true
		return state, nil
	}
	if codeRemoteCommitCanFastForward(repository.SourceDir, state.RemoteCommit, repository.MergeCommit) {
		return state, nil
	}
	if !state.Isolated {
		return state, errCodePushRemoteAdvanced
	}
	// 会话分支重新交付后基线可能变化，允许覆盖自己上次推送的提交；
	// 分支被外部改动过则拒绝，避免丢掉别人的工作。
	if strings.TrimSpace(repository.PushedCommit) != state.RemoteCommit {
		return state, errCodeDeliveryBranchAdvanced
	}
	state.ForceLease = state.RemoteCommit
	return state, nil
}
