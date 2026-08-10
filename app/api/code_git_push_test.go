package api

import (
	"strings"
	"testing"
)

func prepareCodePushTestRepository(t *testing.T) (string, string, string, string) {
	t.Helper()
	repository, remoteDir := createCodeRemoteRepository(t)
	prepared, err := prepareCodeRepository(repository)
	if err != nil {
		t.Fatal(err)
	}
	mergeCommit := commitCodeTestFile(t, repository, "delivery.txt", "delivery\n")
	return repository, remoteDir, prepared.RemoteCommit, mergeCommit
}

func TestPushCodeDeliveryRepositoryPushesExactCommit(t *testing.T) {
	repository, remoteDir, remoteCommit, mergeCommit := prepareCodePushTestRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	result, err := pushCodeDeliveryRepository(repository, branch, "origin", branch, remoteCommit, mergeCommit, codePushPending)
	if err != nil || result.Status != codePushPushed || result.Commit != mergeCommit {
		t.Fatalf("unexpected push result: %#v, %v", result, err)
	}
	remoteHead, err := runCodeGit(remoteDir, "rev-parse", "refs/heads/"+branch)
	if err != nil || remoteHead != mergeCommit {
		t.Fatalf("remote commit = %q, want %q: %v", remoteHead, mergeCommit, err)
	}
}

func TestPushCodeDeliveryRepositoryRecoversAfterRemoteSuccess(t *testing.T) {
	repository, _, remoteCommit, mergeCommit := prepareCodePushTestRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	if _, err := runCodeGit(repository, "push", "origin", mergeCommit+":refs/heads/"+branch); err != nil {
		t.Fatal(err)
	}
	result, err := pushCodeDeliveryRepository(repository, branch, "origin", branch, remoteCommit, mergeCommit, codePushPending)
	if err != nil || result.Status != codePushPushed || result.Commit != mergeCommit {
		t.Fatalf("remote success was not recovered: %#v, %v", result, err)
	}
}

func TestPushCodeDeliveryRepositoryRejectsRemoteUpdate(t *testing.T) {
	repository, remoteDir, remoteCommit, mergeCommit := prepareCodePushTestRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	updater := cloneCodeRepository(t, remoteDir)
	commitCodeTestFile(t, updater, "remote-change.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	result, err := pushCodeDeliveryRepository(repository, branch, "origin", branch, remoteCommit, mergeCommit, codePushPending)
	if err == nil || result.Status != codePushFailed || !strings.Contains(err.Error(), "远端分支已在交付后更新") {
		t.Fatalf("remote update should be rejected: %#v, %v", result, err)
	}
}

func TestPushCodeDeliveryRepositoryRejectsLaterLocalCommit(t *testing.T) {
	repository, _, remoteCommit, mergeCommit := prepareCodePushTestRepository(t)
	branch, _ := runCodeGit(repository, "branch", "--show-current")
	commitCodeTestFile(t, repository, "later.txt", "later\n")
	result, err := pushCodeDeliveryRepository(repository, branch, "origin", branch, remoteCommit, mergeCommit, codePushPending)
	if err == nil || result.Status != codePushFailed || !strings.Contains(err.Error(), "交付后发生变化") {
		t.Fatalf("later local commit should be rejected: %#v, %v", result, err)
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
