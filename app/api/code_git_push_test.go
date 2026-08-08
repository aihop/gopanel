package api

import (
	"errors"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
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

func TestPushCodeSessionDeliveryRejectsMultiRepositoryBypass(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{ID: 101, ProjectID: 1}
	repositories := []model.AIDevSessionRepository{{
		SessionID: session.ID, ProjectID: session.ProjectID, SourceDir: t.TempDir(), WorktreeDir: t.TempDir(),
		LinkName: "repository", Branch: "work", TargetBranch: "main", BaseCommit: "base",
		Status: codeDeliveryMerged, PushStatus: codePushPending,
	}}
	if err := database.Create(&repositories).Error; err != nil {
		t.Fatal(err)
	}
	status, err := loadCodeDeliveryPushStatus(session)
	if err != nil || status.Available || status.Status != "unavailable" || len(status.Repositories) != 0 {
		t.Fatalf("unexpected multi-repository push status: %#v, %v", status, err)
	}
	result, err := pushCodeSessionDelivery(session)
	if !errors.Is(err, errCodeMultiRepositoryManualPush) || result.Available {
		t.Fatalf("multi-repository push bypass was not rejected: %#v, %v", result, err)
	}
	var stored model.AIDevSessionRepository
	if err := database.First(&stored, repositories[0].ID).Error; err != nil || stored.PushStatus != codePushPending {
		t.Fatalf("push bypass changed repository state: %#v, %v", stored, err)
	}
}
