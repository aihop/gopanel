package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

const codeDeliveryPushAttempts = 3

func integrateAndPushCodeDelivery(delivery *model.AICodeDelivery) (codeGitDeliveryResult, error) {
	return integrateAndPushCodeDeliveryWithProgress(delivery, nil)
}

func integrateAndPushCodeDeliveryWithProgress(delivery *model.AICodeDelivery, report codeDeliveryProgressReporter) (codeGitDeliveryResult, error) {
	result := codeGitDeliveryResult{Status: "merged", ResultType: "local", Commit: delivery.MergeCommit, Branch: delivery.WorktreeBranch}
	if delivery.PushStatus == codePushPushed {
		result.ResultType = "remote_verified"
	}
	for attempt := 0; attempt < codeDeliveryPushAttempts; attempt++ {
		if delivery.Status == codeDeliveryPrepared {
			merged, err := mergePreparedCodeDeliveryWithProgress(delivery, report)
			if err != nil || merged.Status == "conflict" {
				return merged, err
			}
			result = merged
		}
		if delivery.Status != codeDeliveryMerged {
			return result, nil
		}
		if !codeDeliveryHasRemote(delivery.RemoteName, deliveryRemoteBranch(delivery.RemoteBranch, delivery.TargetBranch)) {
			if err := fastForwardCodeDeliverySource(delivery); err != nil {
				return codeGitDeliveryResult{}, err
			}
			if err := markCodeDeliveryLocalPush(delivery); err != nil {
				return codeGitDeliveryResult{}, err
			}
			return result, nil
		}
		if report != nil {
			report(codeDeliveryStagePushing, 70)
		}
		pushResult, err := pushCodeIntegratedDeliveryWithProgress(delivery, report)
		if persistErr := persistCodeDeliveryPushResult(delivery, pushResult); persistErr != nil {
			return codeGitDeliveryResult{}, persistErr
		}
		if err == nil {
			if err := fastForwardCodeDeliverySource(delivery); err != nil {
				return codeGitDeliveryResult{}, err
			}
			result.ResultType = "remote_verified"
			return result, nil
		}
		if !errors.Is(err, errCodePushRemoteAdvanced) || attempt == codeDeliveryPushAttempts-1 {
			return codeGitDeliveryResult{}, err
		}
		if err := resetCodeDeliveryToRemote(delivery); err != nil {
			return codeGitDeliveryResult{}, err
		}
	}
	return codeGitDeliveryResult{}, errCodePushRemoteAdvanced
}

func codeDeliveryHasRemote(remoteName, remoteBranch string) bool {
	return strings.TrimSpace(remoteName) != "" && strings.TrimSpace(remoteBranch) != ""
}

func persistCodeDeliveryPushResult(delivery *model.AICodeDelivery, result codeRepositoryPushResult) error {
	updates := map[string]any{"push_status": result.Status, "push_error": result.ErrorMessage}
	if result.Status == codePushPushed {
		now := time.Now()
		updates["pushed_commit"], updates["pushed_at"] = result.Commit, now
		delivery.PushedCommit, delivery.PushedAt = result.Commit, &now
	}
	if err := global.DB.Model(delivery).Updates(updates).Error; err != nil {
		return err
	}
	delivery.PushStatus, delivery.PushError = result.Status, result.ErrorMessage
	return nil
}

func resetCodeDeliveryToRemote(delivery *model.AICodeDelivery) error {
	remoteBranch := deliveryRemoteBranch(delivery.RemoteBranch, delivery.TargetBranch)
	remoteRef := "refs/remotes/" + delivery.RemoteName + "/" + remoteBranch
	remoteCommit, err := runCodeGit(delivery.SourceWorkDir, "rev-parse", remoteRef)
	if err != nil {
		return err
	}
	if err := resetCodeDeliveryWorktree(delivery, remoteCommit); err != nil {
		return fmt.Errorf("恢复最新远端基线失败：%w", err)
	}
	updates := map[string]any{
		"status": codeDeliveryPrepared, "remote_commit": remoteCommit,
		"merge_commit": "", "merged_at": nil, "push_status": codePushPending, "push_error": "",
	}
	if err := global.DB.Model(delivery).Updates(updates).Error; err != nil {
		return err
	}
	delivery.Status, delivery.RemoteCommit = codeDeliveryPrepared, remoteCommit
	delivery.MergeCommit, delivery.MergedAt = "", nil
	delivery.PushStatus, delivery.PushError = codePushPending, ""
	return nil
}

func persistCodeRepositoryPushResult(repository *model.AIDevSessionRepository, result codeRepositoryPushResult) error {
	updates := map[string]any{"push_status": result.Status, "push_error": result.ErrorMessage}
	if result.Status == codePushPushed {
		now := time.Now()
		updates["pushed_commit"], updates["pushed_at"] = result.Commit, now
		repository.PushedCommit, repository.PushedAt = result.Commit, &now
	}
	if err := global.DB.Model(repository).Updates(updates).Error; err != nil {
		return err
	}
	repository.PushStatus, repository.PushError = result.Status, result.ErrorMessage
	return nil
}

func resetCodeRepositoryToRemote(repository *model.AIDevSessionRepository) error {
	remoteBranch := deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch)
	remoteRef := "refs/remotes/" + repository.RemoteName + "/" + remoteBranch
	remoteCommit, err := runCodeGit(repository.SourceDir, "rev-parse", remoteRef)
	if err != nil {
		return err
	}
	localCommit, err := runCodeGit(repository.SourceDir, "rev-parse", "refs/heads/"+repository.TargetBranch)
	if err != nil || localCommit != repository.MergeCommit {
		return errors.New("目标分支已在交付后发生变化，无法自动重试")
	}
	if _, err := runCodeGit(repository.SourceDir, "reset", "--hard", remoteCommit); err != nil {
		return fmt.Errorf("恢复仓库 %s 最新远端基线失败：%w", repository.LinkName, err)
	}
	updates := map[string]any{
		"status": codeDeliveryPrepared, "remote_commit": remoteCommit,
		"merge_commit": "", "merged_at": nil, "push_status": codePushPending, "push_error": "",
	}
	if err := global.DB.Model(repository).Updates(updates).Error; err != nil {
		return err
	}
	repository.Status, repository.RemoteCommit = codeDeliveryPrepared, remoteCommit
	repository.MergeCommit, repository.MergedAt = "", nil
	repository.PushStatus, repository.PushError = codePushPending, ""
	return nil
}

func integrateAndPushCodeRepository(session *model.AIDevSession, repository *model.AIDevSessionRepository, repositories []model.AIDevSessionRepository) (codeRepositoryDeliveryResult, error) {
	return integrateAndPushCodeRepositoryWithProgress(session, repository, repositories, nil)
}

func integrateAndPushCodeRepositoryWithProgress(session *model.AIDevSession, repository *model.AIDevSessionRepository, repositories []model.AIDevSessionRepository, report codeDeliveryProgressReporter) (codeRepositoryDeliveryResult, error) {
	result := codeRepositoryDeliveryResult{
		RepositoryID: codeSessionRepositoryID(repository.ID), RepositoryName: repository.LinkName,
		Status: repository.Status, Branch: repository.Branch, TargetBranch: repository.TargetBranch,
		Remote: repository.RemoteName, RemoteBranch: deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch),
		Commit: repository.MergeCommit, PushStatus: repository.PushStatus,
		PushedCommit: repository.PushedCommit, ErrorMessage: repository.PushError,
	}
	for attempt := 0; attempt < codeDeliveryPushAttempts; attempt++ {
		if report != nil {
			report(codeDeliveryStageMerging, 55)
		}
		merged, err := mergeCodeSessionRepository(repository, repositories)
		if err != nil || merged.Status == "conflict" {
			return merged, err
		}
		result = merged
		result.TargetBranch = repository.TargetBranch
		result.Remote = repository.RemoteName
		result.RemoteBranch = deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch)
		remoteBranch := deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch)
		if !codeDeliveryHasRemote(repository.RemoteName, remoteBranch) {
			result.PushStatus = "local"
			return result, nil
		}
		if report != nil {
			report(codeDeliveryStagePushing, 70)
		}
		pushResult, pushErr := pushCodeDeliveryRepositoryWithCredential(
			repository.SourceDir, repository.TargetBranch, repository.RemoteName, remoteBranch,
			repository.RemoteCommit, repository.MergeCommit, repository.PushStatus,
			codeProjectGitCredentialID(session.ProjectID), report,
		)
		if err := persistCodeRepositoryPushResult(repository, pushResult); err != nil {
			return result, err
		}
		result.PushStatus, result.PushedCommit, result.ErrorMessage = pushResult.Status, pushResult.Commit, pushResult.ErrorMessage
		if pushErr == nil {
			return result, nil
		}
		if !errors.Is(pushErr, errCodePushRemoteAdvanced) || attempt == codeDeliveryPushAttempts-1 {
			return result, pushErr
		}
		if err := resetCodeRepositoryToRemote(repository); err != nil {
			return result, err
		}
	}
	return result, errCodePushRemoteAdvanced
}
