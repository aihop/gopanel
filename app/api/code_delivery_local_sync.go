package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// 单仓交付仍允许把本地快进失败记录为可恢复状态；多仓交付必须保证每个参与仓库
// 都真实进入目标分支，因此 applyCodeRepositoryLocalSync 会把快进失败返回给批次流程。

func shortCodeCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

// verifyCodeDeliveryCommitExists 只校验交付提交在仓库对象库中真实存在，
// 不校验它是否已经进入某个分支 —— 集成 Worktree 与源仓共享对象库，
// 提交造出来就能推送，不需要本地分支指向它。
//
// 原名是 ...CommitReachable，但它从来没做过可达性判断，
// 结果被 completeCodeMultiRepositorySources 当成「已落地」的前置校验用，
// 快进失败时这道校验照样通过。判断是否真的进了目标分支要看 SourceAppliedAt。
func verifyCodeDeliveryCommitExists(gitDir, commit, label string) error {
	commit = strings.TrimSpace(commit)
	if commit == "" {
		return fmt.Errorf("%s尚未产出交付提交", label)
	}
	if _, err := runCodeGit(gitDir, "cat-file", "-e", commit+"^{commit}"); err != nil {
		return fmt.Errorf("%s的交付提交 %s 在仓库对象库中不存在", label, shortCodeCommit(commit))
	}
	return nil
}

// codeDeliveryLocalSyncReason 归一化快进失败原因，只描述原因本身。
// 补救命令由 codeDeliveryLocalSyncCommand 独立给出，避免把命令拼进文案后前端还要反解。
func codeDeliveryLocalSyncReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "本地主仓未能自动快进"
	}
	return reason
}

// codeDeliveryLocalSyncCommand 给出可直接执行的手动同步命令。
func codeDeliveryLocalSyncCommand(sourceDir, mergeCommit string) string {
	sourceDir, mergeCommit = strings.TrimSpace(sourceDir), strings.TrimSpace(mergeCommit)
	if sourceDir == "" || mergeCommit == "" {
		return ""
	}
	return fmt.Sprintf("git -C %s merge --ff-only %s", sourceDir, mergeCommit)
}

func applyCodeDeliveryLocalSync(delivery *model.AICodeDelivery) error {
	if err := verifyCodeDeliveryCommitExists(delivery.SourceWorkDir, delivery.MergeCommit, "本次交付"); err != nil {
		return err
	}
	if err := fastForwardCodeDeliverySource(delivery); err != nil {
		return persistCodeDeliveryLocalSync(delivery, nil, codeDeliveryLocalSyncReason(err.Error()))
	}
	appliedAt := time.Now()
	return persistCodeDeliveryLocalSync(delivery, &appliedAt, "")
}

func persistCodeDeliveryLocalSync(delivery *model.AICodeDelivery, appliedAt *time.Time, syncError string) error {
	if err := global.DB.Model(delivery).Updates(map[string]any{
		"source_applied_at": appliedAt, "local_sync_error": syncError,
	}).Error; err != nil {
		return err
	}
	delivery.SourceAppliedAt, delivery.LocalSyncError = appliedAt, syncError
	return nil
}

func applyCodeRepositoryLocalSync(
	repository *model.AIDevSessionRepository,
	repositories []model.AIDevSessionRepository,
) error {
	label := "仓库 " + repository.LinkName
	if err := verifyCodeDeliveryCommitExists(repository.SourceDir, repository.MergeCommit, label); err != nil {
		return err
	}
	if err := fastForwardCodeMultiRepositorySource(repository, repositories); err != nil {
		syncError := codeDeliveryLocalSyncReason(err.Error())
		if persistErr := persistCodeRepositoryLocalSync(repository, nil, syncError); persistErr != nil {
			return persistErr
		}
		return fmt.Errorf("仓库 %s 未能交付到本地目标分支：%s", repository.LinkName, syncError)
	}
	return persistCodeRepositoryLocalSync(repository, repository.SourceAppliedAt, "")
}

func persistCodeRepositoryLocalSync(
	repository *model.AIDevSessionRepository,
	appliedAt *time.Time,
	syncError string,
) error {
	if err := global.DB.Model(repository).Updates(map[string]any{
		"source_applied_at": appliedAt, "local_sync_error": syncError,
	}).Error; err != nil {
		return err
	}
	repository.SourceAppliedAt, repository.LocalSyncError = appliedAt, syncError
	return nil
}
