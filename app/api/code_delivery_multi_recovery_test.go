package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestMultiRepositoryManualPushRecoversRemotePushBeforeDatabaseUpdate(t *testing.T) {
	session, _ := createRemoteCodeMultiRepositorySession(t, 943)
	prepared := prepareCodeMultiRepositoryRecoveryFixture(t, session)
	mergeCommits := codeTestRepositoryCommits(prepared)
	pushed := &prepared[0]
	remoteBranch := deliveryRemoteBranch(pushed.RemoteBranch, pushed.TargetBranch)
	if _, err := runCodeGit(
		pushed.SourceDir, "push", "--", pushed.RemoteName,
		pushed.MergeCommit+":refs/heads/"+remoteBranch,
	); err != nil {
		t.Fatal(err)
	}

	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != codeDeliveryMerged || result.ResultType != "local" {
		t.Fatalf("local delivery failed: %#v, %v", result, err)
	}
	pushResult, err := pushCodeSessionDelivery(session)
	if err != nil || pushResult.Status != codePushPushed {
		t.Fatalf("remote push recovery failed: %#v, %v", pushResult, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		repository := &stored[index]
		if repository.Status != codeDeliveryCompleted || repository.PushStatus != codePushPushed ||
			repository.PushedCommit != mergeCommits[repository.ID] || repository.IntegrationWorkDir != "" {
			t.Fatalf("repository recovery state is incomplete: %#v", repository)
		}
		assertCodeTestSourceState(t, repository, mergeCommits[repository.ID])
		remoteHead, remoteErr := codeTestRemoteHead(t, repository.SourceDir)
		if remoteErr != nil || remoteHead != mergeCommits[repository.ID] {
			t.Fatalf("remote repository %s head=%q want=%q err=%v", repository.LinkName, remoteHead, mergeCommits[repository.ID], remoteErr)
		}
	}
	assertCodeMultiIntegrationCleanup(t, session, stored)
}

func TestMultiRepositoryExternalPushDoesNotTriggerQualityGate(t *testing.T) {
	session, _ := createRemoteCodeMultiRepositorySession(t, 950)
	if err := global.DB.Model(&model.AIProject{}).Where("id = ?", session.ProjectID).
		Update("require_quality_gate", true).Error; err != nil {
		t.Fatal(err)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) < 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	writeAndCommitCodeTestFiles(t, repositories[0].WorktreeDir, map[string]string{
		"package.json": `{"scripts":{"test":"node verify.js"}}`,
		"verify.js":    "process.exit(1);\n",
	})
	for index := 1; index < len(repositories); index++ {
		commitCodeTestFile(t, repositories[index].WorktreeDir, "delivery.txt", repositories[index].LinkName+"\n")
	}
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	if result, err := prepareCodeMultiRepositoryDeliveryWithProgress(session, nil); err != nil || result.Status != codeDeliveryMerged {
		t.Fatalf("prepare delivery: %#v, %v", result, err)
	}
	prepared, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	pushed := &prepared[0]
	remoteBranch := deliveryRemoteBranch(pushed.RemoteBranch, pushed.TargetBranch)
	if _, err := runCodeGit(
		pushed.SourceDir, "push", "--", pushed.RemoteName,
		pushed.MergeCommit+":refs/heads/"+remoteBranch,
	); err != nil {
		t.Fatal(err)
	}

	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != codeDeliveryMerged || result.ResultType != "local" {
		t.Fatalf("external push blocked local delivery: %#v, %v", result, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		assertCodeTestSourceState(t, &stored[index], stored[index].MergeCommit)
	}
}

func TestMultiRepositoryManualPushPreflightRejectsAdvancedRemote(t *testing.T) {
	session, _ := createRemoteCodeMultiRepositorySession(t, 944)
	prepared := prepareCodeMultiRepositoryRecoveryFixture(t, session)
	pushed := &prepared[0]
	remoteBranch := deliveryRemoteBranch(pushed.RemoteBranch, pushed.TargetBranch)
	if _, err := runCodeGit(
		pushed.SourceDir, "push", "--", pushed.RemoteName,
		pushed.MergeCommit+":refs/heads/"+remoteBranch,
	); err != nil {
		t.Fatal(err)
	}
	advanced := &prepared[1]
	updater := cloneCodeRepository(t, codeTestRemoteURL(t, advanced.SourceDir))
	advancedCommit := commitCodeTestFile(t, updater, "advanced-after-preflight.txt", "advanced\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}

	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != codeDeliveryMerged || result.ResultType != "local" {
		t.Fatalf("local delivery failed: %#v, %v", result, err)
	}
	pushResult, pushErr := pushCodeSessionDelivery(session)
	if !errors.Is(pushErr, errCodePushRemoteAdvanced) || pushResult.Status != codePushFailed {
		t.Fatalf("advanced remote was not rejected before manual push: %#v, %v", pushResult, pushErr)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		repository := &stored[index]
		assertCodeTestSourceState(t, repository, repository.MergeCommit)
		if repository.ID == advanced.ID {
			remoteHead, remoteErr := codeTestRemoteHead(t, repository.SourceDir)
			if remoteErr != nil || remoteHead != advancedCommit {
				t.Fatalf("advanced remote changed unexpectedly: got=%q want=%q err=%v", remoteHead, advancedCommit, remoteErr)
			}
		}
	}
}

func TestMultiRepositoryDeliveryResumesPartiallyAppliedSources(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 945)
	prepared := prepareCodeMultiRepositoryRecoveryFixture(t, session)
	mergeCommits := codeTestRepositoryCommits(prepared)
	applied := &prepared[0]
	if _, err := runCodeGit(applied.SourceDir, "merge", "--ff-only", applied.MergeCommit); err != nil {
		t.Fatal(err)
	}

	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != codeDeliveryMerged || result.ResultType != "local" {
		t.Fatalf("partially applied source recovery failed: %#v, %v", result, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		if stored[index].Status != codeDeliveryCompleted || stored[index].IntegrationWorkDir != "" {
			t.Fatalf("source repository was not completed: %#v", stored[index])
		}
		assertCodeTestSourceState(t, &stored[index], mergeCommits[stored[index].ID])
	}
	assertCodeMultiIntegrationCleanup(t, session, stored)
}

func TestGitlinkMultiRepositoryDeliveryResumesAfterParentFastForward(t *testing.T) {
	session, parentSource, childSource := createCodeGitlinkDeliverySession(t, 946, false)
	prepared := prepareCodeMultiRepositoryRecoveryFixture(t, session)
	parent := codeTestRepositoryBySource(t, prepared, parentSource)
	child := codeTestRepositoryBySource(t, prepared, childSource)
	if _, err := runCodeGit(parent.SourceDir, "merge", "--ff-only", parent.MergeCommit); err != nil {
		t.Fatal(err)
	}
	parentStatus, err := runCodeGit(parent.SourceDir, "status", "--porcelain")
	if err != nil || !strings.Contains(parentStatus, child.GitlinkPath) {
		t.Fatalf("expected managed gitlink transition before recovery: %q, %v", parentStatus, err)
	}

	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != codeDeliveryMerged || result.ResultType != "local" {
		t.Fatalf("gitlink source recovery failed: %#v, %v", result, err)
	}
	assertCodeTestSourceState(t, parent, parent.MergeCommit)
	assertCodeTestSourceState(t, child, child.MergeCommit)
	entry, err := runCodeGit(parent.SourceDir, "ls-tree", "HEAD", "--", child.GitlinkPath)
	if err != nil || !strings.Contains(entry, child.MergeCommit) {
		t.Fatalf("recovered parent gitlink=%q want child %s: %v", entry, child.MergeCommit, err)
	}
}

func TestMultiRepositoryDeliveryCompletesCleanupWithoutRerunningQuality(t *testing.T) {
	session, project, _ := createMultiRepositorySession(t, 947)
	if err := global.DB.Model(project).Update("require_quality_gate", true).Error; err != nil {
		t.Fatal(err)
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	marker := filepath.Join(t.TempDir(), "quality-ran")
	writeAndCommitCodeTestFiles(t, repositories[0].WorktreeDir, map[string]string{
		"package.json": `{"scripts":{"test":"node verify.js"}}`,
		"verify.js":    "require('fs').writeFileSync(" + quotedCodeTestJS(marker) + ",'ran');process.exit(1);\n",
	})
	for index := 1; index < len(repositories); index++ {
		commitCodeTestFile(t, repositories[index].WorktreeDir, "delivery.txt", repositories[index].LinkName+"\n")
	}
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareCodeMultiRepositoryDeliveryWithProgress(session, nil); err != nil {
		t.Fatal(err)
	}
	prepared, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	completedAt := time.Now()
	for index := range prepared {
		if _, err := runCodeGit(prepared[index].SourceDir, "merge", "--ff-only", prepared[index].MergeCommit); err != nil {
			t.Fatal(err)
		}
		if err := global.DB.Model(&prepared[index]).Updates(map[string]any{
			"status": codeDeliveryCompleted, "source_applied_at": completedAt,
			"completed_at": completedAt, "push_status": "local",
		}).Error; err != nil {
			t.Fatal(err)
		}
		prepared[index].Status = codeDeliveryCompleted
		prepared[index].SourceAppliedAt, prepared[index].CompletedAt = &completedAt, &completedAt
	}

	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != codeDeliveryMerged || result.ResultType != "local" {
		t.Fatalf("completed delivery cleanup recovery failed: %#v, %v", result, err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("quality gate unexpectedly reran during completed recovery: %v", err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertCodeMultiIntegrationCleanup(t, session, stored)
}

func prepareCodeMultiRepositoryRecoveryFixture(
	t *testing.T,
	session *model.AIDevSession,
) []model.AIDevSessionRepository {
	t.Helper()
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) < 2 {
		t.Fatalf("load recovery repositories: %#v, %v", repositories, err)
	}
	for index := range repositories {
		commitCodeTestFile(
			t, repositories[index].WorktreeDir,
			"delivery-"+repositories[index].LinkName+".txt", repositories[index].LinkName+"\n",
		)
	}
	if err := captureCodeMultiRepositoryDeliverySnapshot(session); err != nil {
		t.Fatal(err)
	}
	if result, err := prepareCodeMultiRepositoryDeliveryWithProgress(session, nil); err != nil || result.Status != codeDeliveryMerged {
		t.Fatalf("prepare recovery delivery: %#v, %v", result, err)
	}
	repositories, err = loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	repositories, err = codeDeliveryRepositoriesInOrder(repositories, false)
	if err != nil {
		t.Fatal(err)
	}
	return repositories
}

func codeTestRepositoryCommits(repositories []model.AIDevSessionRepository) map[uint]string {
	commits := make(map[uint]string, len(repositories))
	for index := range repositories {
		commits[repositories[index].ID] = repositories[index].MergeCommit
	}
	return commits
}

func assertCodeMultiIntegrationCleanup(
	t *testing.T,
	session *model.AIDevSession,
	repositories []model.AIDevSessionRepository,
) {
	t.Helper()
	root := codeMultiDeliveryRootDir(session)
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("multi-repository integration root remained: %v", err)
	}
	for index := range repositories {
		worktrees, err := runCodeGit(repositories[index].SourceDir, "worktree", "list", "--porcelain")
		if err != nil || strings.Contains(worktrees, root) {
			t.Fatalf("repository %s retained integration worktree registration: %q, %v", repositories[index].LinkName, worktrees, err)
		}
	}
}
