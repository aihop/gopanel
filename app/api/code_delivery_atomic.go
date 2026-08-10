package api

import (
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func integrateCodeDeliveryLocallyWithProgress(delivery *model.AICodeDelivery, report codeDeliveryProgressReporter) (codeGitDeliveryResult, error) {
	result := codeGitDeliveryResult{Status: "merged", ResultType: "local", Commit: delivery.MergeCommit, Branch: delivery.WorktreeBranch}
	if delivery.PushStatus == codePushPushed {
		result.ResultType = "remote_verified"
	}
	if delivery.Status == codeDeliveryPrepared {
		merged, err := mergePreparedCodeDeliveryWithProgress(delivery, report)
		if err != nil || merged.Status == "conflict" {
			return merged, err
		}
		result = merged
		result.ResultType = "local"
	}
	if delivery.Status != codeDeliveryMerged {
		return result, nil
	}
	if err := applyCodeDeliveryLocalSync(delivery); err != nil {
		return codeGitDeliveryResult{}, err
	}
	if !codeDeliveryHasRemote(delivery.RemoteName, deliveryRemoteBranch(delivery.RemoteBranch, delivery.TargetBranch)) {
		// 本地主仓没能快进时，交付提交已经产出但还没落到任何地方，不能标记为本地完成。
		if delivery.SourceAppliedAt != nil {
			if err := markCodeDeliveryLocalPush(delivery); err != nil {
				return codeGitDeliveryResult{}, err
			}
		}
	} else if delivery.PushStatus != codePushPushed {
		if err := global.DB.Model(delivery).Updates(map[string]any{
			"push_status": codePushPending, "push_error": "",
		}).Error; err != nil {
			return codeGitDeliveryResult{}, err
		}
		delivery.PushStatus, delivery.PushError = codePushPending, ""
	}
	// 与多仓路径保持一致：交付提交已产出，但既没同步本地也没推远端。
	if delivery.SourceAppliedAt == nil && delivery.PushStatus != codePushPushed {
		result.ResultType = "delivered"
	}
	return result, nil
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
