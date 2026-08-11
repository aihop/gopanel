package api

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
)

// 造一个带远端跟踪引用的仓库：refs/remotes/origin/main 指向当前 HEAD。
func withRemoteTrackingRef(t *testing.T) (string, string) {
	t.Helper()
	dir := createCodeGitRepository(t)
	head, err := runCodeGit(dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(dir, "update-ref", "refs/remotes/origin/main", head); err != nil {
		t.Fatal(err)
	}
	return dir, head
}

func TestReconcileCodeRepositoryPushStatusHealsStalePending(t *testing.T) {
	withCodeGovernanceDB(t)
	dir, head := withRemoteTrackingRef(t)
	// 交付提交其实已经在远端了，但库里还停在 pending —— 正是 apay 那种情况。
	repository := &model.AIDevSessionRepository{
		SourceDir: dir, MergeCommit: head, RemoteName: "origin", PushStatus: codePushPending,
	}
	if !reconcileCodeRepositoryPushStatus(repository, "main") {
		t.Fatal("提交已在远端跟踪分支上，应当补正为已推送")
	}
	if repository.PushStatus != codePushPushed || repository.PushedCommit != head {
		t.Fatalf("状态没补正: %#v", repository)
	}
	if repository.PushedAt == nil {
		t.Fatal("补正后应记录推送时间")
	}
}

func TestReconcileCodeRepositoryPushStatusLeavesUnpushedAlone(t *testing.T) {
	withCodeGovernanceDB(t)
	dir, head := withRemoteTrackingRef(t)
	// 在远端跟踪引用之后又产生了新提交：这一条确实还没推。
	if _, err := runCodeGit(dir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local",
		"commit", "--allow-empty", "-m", "later"); err != nil {
		t.Fatal(err)
	}
	later, err := runCodeGit(dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if later == head {
		t.Fatal("fixture 没有产生新提交")
	}
	repository := &model.AIDevSessionRepository{
		SourceDir: dir, MergeCommit: later, RemoteName: "origin", PushStatus: codePushPending,
	}
	if reconcileCodeRepositoryPushStatus(repository, "main") {
		t.Fatal("提交不在远端，不能谎报已推送")
	}
	if repository.PushStatus != codePushPending {
		t.Fatalf("不该改动状态: %s", repository.PushStatus)
	}
}

func TestReconcileCodeRepositoryPushStatusSkipsWithoutTrackingRef(t *testing.T) {
	withCodeGovernanceDB(t)
	dir := createCodeGitRepository(t)
	head, err := runCodeGit(dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	// 从没 fetch 过，没有跟踪引用：无从判断，保持原状而不是乱猜。
	repository := &model.AIDevSessionRepository{
		SourceDir: dir, MergeCommit: head, RemoteName: "origin", PushStatus: codePushPending,
	}
	if reconcileCodeRepositoryPushStatus(repository, "main") {
		t.Fatal("没有远端跟踪引用时不该判定为已推送")
	}
}

func TestReconcileCodeRepositoryPushStatusIgnoresAlreadyPushed(t *testing.T) {
	withCodeGovernanceDB(t)
	repository := &model.AIDevSessionRepository{PushStatus: codePushPushed}
	if reconcileCodeRepositoryPushStatus(repository, "main") {
		t.Fatal("已推送的不需要再补正")
	}
}
