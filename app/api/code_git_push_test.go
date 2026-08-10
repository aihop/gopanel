package api

import (
	"strings"
	"testing"
)

func prepareCodePushTestRepository(t *testing.T) (string, string, string) {
	t.Helper()
	repository, remoteDir := createCodeRemoteRepository(t)
	if _, err := prepareCodeRepository(repository); err != nil {
		t.Fatal(err)
	}
	mergeCommit := commitCodeTestFile(t, repository, "delivery.txt", "delivery\n")
	return repository, remoteDir, mergeCommit
}

func TestPushCodeDeliveryRepositoryPushesExactCommit(t *testing.T) {
	repository, remoteDir, mergeCommit := prepareCodePushTestRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	result, err := pushCodeDeliveryRepository(repository, "origin", branch, mergeCommit, codePushPending)
	if err != nil || result.Status != codePushPushed || result.Commit != mergeCommit {
		t.Fatalf("unexpected push result: %#v, %v", result, err)
	}
	remoteHead, err := runCodeGit(remoteDir, "rev-parse", "refs/heads/"+branch)
	if err != nil || remoteHead != mergeCommit {
		t.Fatalf("remote commit = %q, want %q: %v", remoteHead, mergeCommit, err)
	}
}

func TestPushCodeDeliveryRepositoryRecoversAfterRemoteSuccess(t *testing.T) {
	repository, _, mergeCommit := prepareCodePushTestRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	if _, err := runCodeGit(repository, "push", "origin", mergeCommit+":refs/heads/"+branch); err != nil {
		t.Fatal(err)
	}
	result, err := pushCodeDeliveryRepository(repository, "origin", branch, mergeCommit, codePushPending)
	if err != nil || result.Status != codePushPushed || result.Commit != mergeCommit {
		t.Fatalf("remote success was not recovered: %#v, %v", result, err)
	}
}

func TestPushCodeDeliveryRepositoryRejectsRemoteUpdate(t *testing.T) {
	repository, remoteDir, mergeCommit := prepareCodePushTestRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	updater := cloneCodeRepository(t, remoteDir)
	commitCodeTestFile(t, updater, "remote-change.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	result, err := pushCodeDeliveryRepository(repository, "origin", branch, mergeCommit, codePushPending)
	if err == nil || result.Status != codePushFailed || !strings.Contains(err.Error(), "远端分支已在交付后更新") {
		t.Fatalf("remote update should be rejected: %#v, %v", result, err)
	}
}

func TestPushCodeDeliveryRepositoryAllowsIntegratedRemoteUpdate(t *testing.T) {
	repository, remoteDir := createCodeRemoteRepository(t)
	if _, err := prepareCodeRepository(repository); err != nil {
		t.Fatal(err)
	}
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	updater := cloneCodeRepository(t, remoteDir)
	remoteUpdate := commitCodeTestFile(t, updater, "remote-change.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "fetch", "origin"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "merge", "--ff-only", remoteUpdate); err != nil {
		t.Fatal(err)
	}
	mergeCommit := commitCodeTestFile(t, repository, "delivery.txt", "delivery\n")
	result, err := pushCodeDeliveryRepository(repository, "origin", branch, mergeCommit, codePushPending)
	if err != nil || result.Status != codePushPushed {
		t.Fatalf("integrated remote update should remain pushable: %#v, %v", result, err)
	}
	remoteHead, err := runCodeGit(remoteDir, "rev-parse", "refs/heads/"+branch)
	if err != nil || remoteHead != mergeCommit {
		t.Fatalf("remote commit = %q, want %q: %v", remoteHead, mergeCommit, err)
	}
}

// 交付之后本地又产生新提交，属于「下一次交付」的内容，不应该锁死本次交付。
// 推送必须仍然推出本次交付提交本身，并且不把后续提交带到远端。
func TestPushCodeDeliveryRepositoryPushesDeliveryCommitAfterLaterLocalCommit(t *testing.T) {
	repository, remoteDir, mergeCommit := prepareCodePushTestRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	laterCommit := commitCodeTestFile(t, repository, "later.txt", "later\n")
	result, err := pushCodeDeliveryRepository(repository, "origin", branch, mergeCommit, codePushPending)
	if err != nil || result.Status != codePushPushed || result.Commit != mergeCommit {
		t.Fatalf("later local commit should not block delivery push: %#v, %v", result, err)
	}
	remoteHead, err := runCodeGit(remoteDir, "rev-parse", "refs/heads/"+branch)
	if err != nil || remoteHead != mergeCommit {
		t.Fatalf("remote commit = %q, want delivery commit %q: %v", remoteHead, mergeCommit, err)
	}
	if remoteHead == laterCommit {
		t.Fatal("push must not carry the post-delivery commit to the remote")
	}
}

// 本地主仓未能快进（工作区脏、分支被切走）时，交付提交仍在共享对象库中，推送必须照常可用。
func TestPushCodeDeliveryRepositoryPushesWhenLocalBranchMovedAway(t *testing.T) {
	repository, remoteDir, mergeCommit := prepareCodePushTestRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	if _, err := runCodeGit(repository, "reset", "--hard", "HEAD~1"); err != nil {
		t.Fatal(err)
	}
	result, err := pushCodeDeliveryRepository(repository, "origin", branch, mergeCommit, codePushPending)
	if err != nil || result.Status != codePushPushed || result.Commit != mergeCommit {
		t.Fatalf("delivery commit should stay pushable after local branch moved: %#v, %v", result, err)
	}
	remoteHead, err := runCodeGit(remoteDir, "rev-parse", "refs/heads/"+branch)
	if err != nil || remoteHead != mergeCommit {
		t.Fatalf("remote commit = %q, want %q: %v", remoteHead, mergeCommit, err)
	}
}

func TestPushCodeSessionDeliveryPushesCompletedMultiRepository(t *testing.T) {
	session, _ := createRemoteCodeMultiRepositorySession(t, 101)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range repositories {
		commitCodeTestFile(t, repositories[index].WorktreeDir, "delivery.txt", repositories[index].LinkName+"\n")
	}
	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.ResultType != "local" {
		t.Fatalf("local delivery failed: %#v, %v", result, err)
	}
	status, err := loadCodeDeliveryPushStatus(session)
	if err != nil || !status.Available || status.Status != codePushPending || len(status.Repositories) != len(repositories) {
		t.Fatalf("unexpected multi-repository push status: %#v, %v", status, err)
	}
	pushResult, err := pushCodeSessionDelivery(session)
	if err != nil || pushResult.Status != codePushPushed {
		t.Fatalf("multi-repository push failed: %#v, %v", pushResult, err)
	}
	pushStatus, err := loadCodeDeliveryPushStatus(session)
	if err != nil || pushStatus.Status != codePushPushed {
		t.Fatalf("multi-repository push was not completed: %#v, %v", pushStatus, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		remoteHead, remoteErr := codeTestRemoteHead(t, stored[index].SourceDir)
		if remoteErr != nil || remoteHead != stored[index].MergeCommit {
			t.Fatalf("repository %s remote=%q want=%q: %v", stored[index].LinkName, remoteHead, stored[index].MergeCommit, remoteErr)
		}
	}
}
