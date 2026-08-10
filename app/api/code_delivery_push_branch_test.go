package api

import (
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func setCodeProjectDeliveryMode(t *testing.T, projectID uint, mode string) {
	t.Helper()
	if err := global.DB.Model(&model.AIProject{}).Where("id = ?", projectID).
		Update("delivery_mode", mode).Error; err != nil {
		t.Fatal(err)
	}
}

// advanceCodeTestRemote 模拟别人往远端目标分支推了新提交。
func advanceCodeTestRemote(t *testing.T, sourceDir string) {
	t.Helper()
	updater := cloneCodeRepository(t, codeTestRemoteURL(t, sourceDir))
	commitCodeTestFile(t, updater, "remote-change.txt", "someone else\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
}

func codeTestRemoteRef(t *testing.T, sourceDir, branch string) string {
	t.Helper()
	output, err := runCodeGit(sourceDir, "ls-remote", codeTestRemoteURL(t, sourceDir), "refs/heads/"+branch)
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(output)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// 对照组：direct 模式下远端目标分支被别人推进后，交付推送会被拒绝。
func TestCodeDeliveryDirectModeBlockedWhenRemoteTargetAdvanced(t *testing.T) {
	session, sourceDirs := deliverCodeMultiRepositorySession(t, 621)
	advanceCodeTestRemote(t, sourceDirs[0])

	if _, err := resumeCodeMultiRepositoryDelivery(session, session.UserID); err != nil {
		t.Fatalf("delivery itself must still succeed: %v", err)
	}
	if _, err := pushCodeSessionDelivery(session); err == nil {
		t.Fatal("direct mode should refuse to push over an advanced remote target branch")
	}
}

// branch 模式推的是会话独占分支，远端目标分支被推进不再阻断交付推送。
func TestCodeDeliveryBranchModePushesWhenRemoteTargetAdvanced(t *testing.T) {
	session, sourceDirs := deliverCodeMultiRepositorySession(t, 622)
	advanceCodeTestRemote(t, sourceDirs[0])
	setCodeProjectDeliveryMode(t, session.ProjectID, codeDeliveryModeBranch)

	if _, err := resumeCodeMultiRepositoryDelivery(session, session.UserID); err != nil {
		t.Fatalf("delivery must succeed: %v", err)
	}
	pushResult, err := pushCodeSessionDelivery(session)
	if err != nil || pushResult.Status != codePushPushed {
		t.Fatalf("branch mode should push despite an advanced remote target: %#v, %v", pushResult, err)
	}

	sessionBranch := codeDeliverySessionBranch(session.ID)
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		repository := &stored[index]
		if remote := codeTestRemoteRef(t, repository.SourceDir, sessionBranch); remote != repository.MergeCommit {
			t.Fatalf("repository %s session branch = %q, want delivery commit %q",
				repository.LinkName, remote, repository.MergeCommit)
		}
	}
	// 交付不能动到远端目标分支：合并交给平台的 PR。
	targetRemote := codeTestRemoteRef(t, stored[0].SourceDir, stored[0].TargetBranch)
	if targetRemote == stored[0].MergeCommit {
		t.Fatal("branch mode must not push onto the remote target branch")
	}
}

// 重复推送同一会话的交付：远端已经是该提交时应识别为已推送，不重复推。
func TestCodeDeliveryBranchModePushIsIdempotent(t *testing.T) {
	session, _ := deliverCodeMultiRepositorySession(t, 623)
	setCodeProjectDeliveryMode(t, session.ProjectID, codeDeliveryModeBranch)

	if _, err := resumeCodeMultiRepositoryDelivery(session, session.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := pushCodeSessionDelivery(session); err != nil {
		t.Fatal(err)
	}
	again, err := pushCodeSessionDelivery(session)
	if err != nil || again.Status != codePushPushed {
		t.Fatalf("re-pushing an already delivered session must stay successful: %#v, %v", again, err)
	}
	sessionBranch := codeDeliverySessionBranch(session.ID)
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		if remote := codeTestRemoteRef(t, stored[index].SourceDir, sessionBranch); remote != stored[index].MergeCommit {
			t.Fatalf("repository %s session branch drifted: %q", stored[index].LinkName, remote)
		}
	}
}

// 会话分支是本次交付独占的：重新交付产生的提交与远端分叉时，
// 允许用 --force-with-lease 覆盖自己上次推送的提交。
func TestPushCodeDeliveryIsolatedBranchForcesOverOwnCommit(t *testing.T) {
	repository, remoteDir, firstCommit := prepareCodePushTestRepository(t)
	branch := codeDeliverySessionBranch(41)
	request := codeDeliveryPushRequest{
		SourceDir: repository, RemoteName: "origin", RemoteBranch: branch,
		MergeCommit: firstCommit, PushStatus: codePushPending, Isolated: true,
	}
	if result, err := pushCodeDeliveryRepositoryWithCredential(request, nil); err != nil || result.Status != codePushPushed {
		t.Fatalf("first push to a fresh session branch failed: %#v, %v", result, err)
	}

	// 交付基线倒退并转向另一条线，新交付提交不再是远端提交的后代。
	if _, err := runCodeGit(repository, "reset", "--hard", "HEAD~1"); err != nil {
		t.Fatal(err)
	}
	diverged := commitCodeTestFile(t, repository, "diverged.txt", "diverged delivery\n")
	if _, err := runCodeGit(repository, "merge-base", "--is-ancestor", firstCommit, diverged); err == nil {
		t.Fatal("test setup failed to produce a diverged delivery commit")
	}

	request.MergeCommit, request.LastPushedCommit = diverged, firstCommit
	result, err := pushCodeDeliveryRepositoryWithCredential(request, nil)
	if err != nil || result.Status != codePushPushed || result.Commit != diverged {
		t.Fatalf("session branch should accept its own diverged update: %#v, %v", result, err)
	}
	remoteHead, err := runCodeGit(remoteDir, "rev-parse", "refs/heads/"+branch)
	if err != nil || strings.TrimSpace(remoteHead) != diverged {
		t.Fatalf("session branch = %q, want %q: %v", remoteHead, diverged, err)
	}
}

// 会话分支被外部改动过时必须拒绝覆盖，避免丢掉别人的提交。
func TestPushCodeDeliveryIsolatedBranchRejectsForeignUpdate(t *testing.T) {
	repository, remoteDir, firstCommit := prepareCodePushTestRepository(t)
	branch := codeDeliverySessionBranch(42)
	request := codeDeliveryPushRequest{
		SourceDir: repository, RemoteName: "origin", RemoteBranch: branch,
		MergeCommit: firstCommit, PushStatus: codePushPending, Isolated: true,
	}
	if _, err := pushCodeDeliveryRepositoryWithCredential(request, nil); err != nil {
		t.Fatal(err)
	}

	// 别人往会话分支推了东西。
	updater := cloneCodeRepository(t, remoteDir)
	foreign := commitCodeTestFile(t, updater, "foreign.txt", "someone else\n")
	if _, err := runCodeGit(updater, "push", "--force", "origin", foreign+":refs/heads/"+branch); err != nil {
		t.Fatal(err)
	}

	if _, err := runCodeGit(repository, "reset", "--hard", "HEAD~1"); err != nil {
		t.Fatal(err)
	}
	diverged := commitCodeTestFile(t, repository, "diverged.txt", "diverged delivery\n")
	// 我们仍以为远端停在 firstCommit，实际已被改动：必须拒绝。
	request.MergeCommit, request.LastPushedCommit = diverged, firstCommit
	result, err := pushCodeDeliveryRepositoryWithCredential(request, nil)
	if err == nil || result.Status != codePushFailed {
		t.Fatalf("foreign update to the session branch must be rejected: %#v, %v", result, err)
	}
	remoteHead, headErr := runCodeGit(remoteDir, "rev-parse", "refs/heads/"+branch)
	if headErr != nil || strings.TrimSpace(remoteHead) != foreign {
		t.Fatalf("rejected push must leave the foreign commit intact: %q, %v", remoteHead, headErr)
	}
}
