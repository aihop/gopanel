package api

import (
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
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

func TestPushCodeMultiRepositoryDeliveryResumes(t *testing.T) {
	database := withCodeGovernanceDB(t)
	firstDir, _, firstRemoteCommit, firstMergeCommit := prepareCodePushTestRepository(t)
	secondDir, _, secondRemoteCommit, secondMergeCommit := prepareCodePushTestRepository(t)
	firstBranch, _ := runCodeGit(firstDir, "branch", "--show-current")
	secondBranch, _ := runCodeGit(secondDir, "branch", "--show-current")
	repositories := []model.AIDevSessionRepository{
		{SessionID: 101, ProjectID: 1, SourceDir: firstDir, WorktreeDir: firstDir, LinkName: "a", Branch: "work-a", TargetBranch: firstBranch, RemoteName: "origin", RemoteBranch: firstBranch, RemoteCommit: firstRemoteCommit, BaseCommit: firstRemoteCommit, MergeCommit: firstMergeCommit, Status: codeDeliveryCompleted, PushStatus: codePushPending},
		{SessionID: 101, ProjectID: 1, SourceDir: secondDir, WorktreeDir: secondDir, LinkName: "b", Branch: "work-b", TargetBranch: secondBranch, RemoteName: "", RemoteBranch: secondBranch, RemoteCommit: secondRemoteCommit, BaseCommit: secondRemoteCommit, MergeCommit: secondMergeCommit, Status: codeDeliveryCompleted, PushStatus: codePushPending},
	}
	if err := database.Create(&repositories).Error; err != nil {
		t.Fatal(err)
	}
	first, err := pushCodeMultiRepositoryDelivery(101)
	if err == nil || first.Status != codePushFailed || len(first.Repositories) != 2 || first.Repositories[0].Status != codePushPushed {
		t.Fatalf("unexpected partial push: %#v, %v", first, err)
	}
	if err := global.DB.Model(&repositories[1]).Update("remote_name", "origin").Error; err != nil {
		t.Fatal(err)
	}
	second, err := pushCodeMultiRepositoryDelivery(101)
	if err != nil || second.Status != codePushPushed || len(second.Repositories) != 2 {
		t.Fatalf("push did not resume: %#v, %v", second, err)
	}
}
