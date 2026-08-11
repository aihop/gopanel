package api

import (
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// 推送状态自愈。
//
// push_status 是一次性写入的标记，写完就不再核对。于是出现过这种情况：
// 交付提交其实已经在远端分支上了（用户手动推的、或推送成功但结果没落库），
// 数据库里却一直停在 pending，界面显示「未推送」。
//
// 判据用本地的远端跟踪引用（refs/remotes/<remote>/<branch>），不发网络请求 ——
// 它反映的是最近一次 fetch 的远端状态，对「这个提交推上去了没有」这个问题足够可靠，
// 而且不会让一个查询接口挂在网络超时上。

// reconcileCodeRepositoryPushStatus 检查单个仓库：提交已在远端跟踪分支上就把状态补正。
// 返回是否发生了修正。
func reconcileCodeRepositoryPushStatus(repository *model.AIDevSessionRepository, remoteBranch string) bool {
	if repository == nil || repository.PushStatus == codePushPushed {
		return false
	}
	mergeCommit := strings.TrimSpace(repository.MergeCommit)
	remote := strings.TrimSpace(repository.RemoteName)
	remoteBranch = strings.TrimSpace(remoteBranch)
	if mergeCommit == "" || remote == "" || remoteBranch == "" {
		return false
	}
	trackingRef := "refs/remotes/" + remote + "/" + remoteBranch
	if _, err := runCodeGit(repository.SourceDir, "rev-parse", "--verify", "--quiet", trackingRef); err != nil {
		// 没有跟踪引用说明本地从没 fetch 过这个分支，无从判断，保持原状。
		return false
	}
	if _, err := runCodeGit(repository.SourceDir, "merge-base", "--is-ancestor", mergeCommit, trackingRef); err != nil {
		return false
	}
	pushedAt := time.Now()
	repository.PushStatus, repository.PushedCommit, repository.PushError = codePushPushed, mergeCommit, ""
	repository.PushedAt = &pushedAt
	// ID 为 0 时 Model(...).Updates 会生成没有 WHERE 的 UPDATE，
	// GORM 会拒绝执行 —— 但更要紧的是那本来就该拒绝，不能全表刷一遍。
	if global.DB == nil || repository.ID == 0 {
		return true
	}
	if err := global.DB.Model(repository).Where("id = ?", repository.ID).Updates(map[string]any{
		"push_status": codePushPushed, "pushed_commit": mergeCommit,
		"pushed_at": pushedAt, "push_error": "",
	}).Error; err != nil && global.LOG != nil {
		// 落库失败不影响本次返回的结果：状态已经在内存里修正，下次进来会再试一遍。
		global.LOG.Warnf("Reconcile push status for repository %d failed: %v", repository.ID, err)
	}
	return true
}
