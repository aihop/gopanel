package api

import (
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type codeDeliveryFact struct {
	Key    string `json:"key"`
	Status string `json:"status"`
	Count  int    `json:"count,omitempty"`
	Total  int    `json:"total,omitempty"`
}

func loadCodeDeliveryFacts(sessionID uint, repositories []codeRepositoryDeliveryResult) []codeDeliveryFact {
	if stored, err := loadCodeSessionRepositories(sessionID); err == nil && len(stored) > 0 {
		return codeMultiRepositoryDeliveryFacts(codeStoredRepositoryDeliveryResults(stored))
	}
	if len(repositories) > 0 {
		return codeMultiRepositoryDeliveryFacts(repositories)
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", sessionID).First(&delivery).Error; err != nil {
		return nil
	}
	prepared := strings.TrimSpace(delivery.WorktreeCommit) != ""
	merged := strings.TrimSpace(delivery.MergeCommit) != ""
	localApplied := merged && codeDeliverySourceContainsCommit(&delivery)
	remoteVerified := delivery.PushStatus == codePushPushed && delivery.PushedCommit == delivery.MergeCommit
	hasRemote := codeDeliveryHasRemote(delivery.RemoteName, deliveryRemoteBranch(delivery.RemoteBranch, delivery.TargetBranch))
	return []codeDeliveryFact{
		{Key: "snapshot", Status: codeDeliveryFactStatus(prepared, false)},
		{Key: "merge", Status: codeDeliveryFactStatus(merged, false)},
		{Key: "local", Status: codeDeliveryFactStatus(localApplied, false)},
		{Key: "remote", Status: codeDeliveryFactOptionalStatus(hasRemote, remoteVerified)},
	}
}

func codeDeliverySourceContainsCommit(delivery *model.AICodeDelivery) bool {
	if delivery == nil || strings.TrimSpace(delivery.SourceWorkDir) == "" || strings.TrimSpace(delivery.MergeCommit) == "" {
		return false
	}
	branch := strings.TrimSpace(delivery.TargetBranch)
	if branch == "" {
		branch, _ = runCodeGit(delivery.SourceWorkDir, "branch", "--show-current")
	}
	if strings.TrimSpace(branch) == "" {
		return false
	}
	commit, err := runCodeGit(delivery.SourceWorkDir, "rev-parse", "refs/heads/"+branch)
	if err != nil || strings.TrimSpace(commit) == "" {
		return false
	}
	_, err = runCodeGit(delivery.SourceWorkDir, "merge-base", "--is-ancestor", delivery.MergeCommit, commit)
	return err == nil
}

func codeMultiRepositoryDeliveryFacts(repositories []codeRepositoryDeliveryResult) []codeDeliveryFact {
	total := len(repositories)
	prepared, merged, localApplied, remoteVerified, remoteTotal := 0, 0, 0, 0, 0
	for _, repository := range repositories {
		if repository.SnapshotReady {
			prepared++
		}
		if repository.MergeReady || strings.TrimSpace(repository.Commit) != "" {
			merged++
		}
		if repository.SourceAppliedAt != nil || repository.Status == codeDeliveryCompleted {
			localApplied++
		}
		if codeDeliveryHasRemote(repository.Remote, repository.RemoteBranch) {
			remoteTotal++
		}
		if repository.PushStatus == codePushPushed {
			remoteVerified++
		}
	}
	return []codeDeliveryFact{
		newCodeDeliveryCountFact("snapshot", prepared, total),
		newCodeDeliveryCountFact("merge", merged, total),
		newCodeDeliveryCountFact("local", localApplied, total),
		newCodeDeliveryOptionalCountFact("remote", remoteVerified, remoteTotal),
	}
}

func newCodeDeliveryOptionalCountFact(key string, count, total int) codeDeliveryFact {
	if total == 0 {
		return codeDeliveryFact{Key: key, Status: "skipped"}
	}
	return newCodeDeliveryCountFact(key, count, total)
}

func newCodeDeliveryCountFact(key string, count, total int) codeDeliveryFact {
	return codeDeliveryFact{Key: key, Status: codeDeliveryFactStatus(count == total, count > 0), Count: count, Total: total}
}

func codeDeliveryFactStatus(done, partial bool) string {
	if done {
		return "completed"
	}
	if partial {
		return "partial"
	}
	return "pending"
}

func codeDeliveryFactOptionalStatus(applicable, done bool) string {
	if !applicable {
		return "skipped"
	}
	return codeDeliveryFactStatus(done, false)
}
