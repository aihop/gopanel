package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

// 多仓交付只有在每个参与仓库真实进入目标分支后才算完成。

func deliverCodeMultiRepositorySession(t *testing.T, sessionID uint) (*model.AIDevSession, []string) {
	t.Helper()
	session, sourceDirs := createRemoteCodeMultiRepositorySession(t, sessionID)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range repositories {
		commitCodeTestFile(t, repositories[index].WorktreeDir, "delivery.txt", repositories[index].LinkName+"\n")
	}
	return session, sourceDirs
}

func assertCodeDeliveryPushableToRemote(t *testing.T, session *model.AIDevSession) {
	t.Helper()
	status, err := loadCodeDeliveryPushStatus(session)
	if err != nil || !status.Available {
		t.Fatalf("delivery must stay pushable: %#v, %v", status, err)
	}
	pushResult, err := pushCodeSessionDelivery(session)
	if err != nil || pushResult.Status != codePushPushed {
		t.Fatalf("push after degraded local sync failed: %#v, %v", pushResult, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	// 按交付目标分支核验远端，而不是源仓当前签出的分支：
	// 用户可能已经把源仓切到别的分支上了。
	for index := range stored {
		repository := &stored[index]
		branch := deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch)
		output, remoteErr := runCodeGit(
			repository.SourceDir, "ls-remote", codeTestRemoteURL(t, repository.SourceDir), "refs/heads/"+branch,
		)
		fields := strings.Fields(output)
		if remoteErr != nil || len(fields) == 0 {
			t.Fatalf("repository %s has no remote ref %s: %v", repository.LinkName, branch, remoteErr)
		}
		if fields[0] != repository.MergeCommit {
			t.Fatalf("repository %s remote=%q want delivery commit %q",
				repository.LinkName, fields[0], repository.MergeCommit)
		}
	}
}

func assertAllCodeRepositoriesCompleted(t *testing.T, sessionID uint) []model.AIDevSessionRepository {
	t.Helper()
	stored, err := loadCodeSessionRepositories(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stored {
		if stored[index].Status != codeDeliveryCompleted {
			t.Fatalf("repository %s status = %q, want completed", stored[index].LinkName, stored[index].Status)
		}
		if strings.TrimSpace(stored[index].MergeCommit) == "" {
			t.Fatalf("repository %s produced no delivery commit", stored[index].LinkName)
		}
	}
	return stored
}

// 当前签出的目标分支有未提交改动时不能安全推进，整次交付必须明确失败。
func TestCodeDeliveryFailsWhenTargetRepositoryIsDirty(t *testing.T) {
	session, sourceDirs := deliverCodeMultiRepositorySession(t, 611)
	dirty := filepath.Join(sourceDirs[0], "delivery.txt")
	if err := os.WriteFile(dirty, []byte("uncommitted local edit\n"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err == nil || !strings.Contains(err.Error(), "会被交付覆盖") {
		t.Fatalf("dirty target repository must fail delivery: %v (%#v)", err, result)
	}
	stored, loadErr := loadCodeSessionRepositories(session.ID)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	for index := range stored {
		if stored[index].Status == codeDeliveryCompleted || stored[index].SourceAppliedAt != nil {
			t.Fatalf("failed batch was reported as delivered: %#v", stored[index])
		}
	}
}

func TestCodeDeliveryOnlyAdvancesAffectedRepository(t *testing.T) {
	session, _, _ := createMultiRepositorySession(t, 614)
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) != 2 {
		t.Fatalf("load repositories: %#v, %v", repositories, err)
	}
	changed := &repositories[0]
	unchanged := &repositories[1]
	unchangedBefore, err := runCodeGit(unchanged.SourceDir, "rev-parse", "refs/heads/"+unchanged.TargetBranch)
	if err != nil {
		t.Fatal(err)
	}
	commitCodeTestFile(t, changed.WorktreeDir, "delivery.txt", "changed\n")
	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil || result.Status != codeDeliveryMerged {
		t.Fatalf("affected repository delivery failed: %#v, %v", result, err)
	}
	stored, err := loadCodeSessionRepositories(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	changed = codeTestRepositoryByID(t, stored, changed.ID)
	unchanged = codeTestRepositoryByID(t, stored, unchanged.ID)
	if changed.Status != codeDeliveryCompleted || changed.SourceAppliedAt == nil || changed.MergeCommit == "" {
		t.Fatalf("changed repository was not delivered: %#v", changed)
	}
	changedTarget, err := runCodeGit(changed.SourceDir, "rev-parse", "refs/heads/"+changed.TargetBranch)
	if err != nil || changedTarget != changed.MergeCommit {
		t.Fatalf("changed target branch = %q, want %q: %v", changedTarget, changed.MergeCommit, err)
	}
	unchangedTarget, err := runCodeGit(unchanged.SourceDir, "rev-parse", "refs/heads/"+unchanged.TargetBranch)
	if err != nil || unchangedTarget != unchangedBefore {
		t.Fatalf("unchanged target branch moved: before=%q after=%q err=%v", unchangedBefore, unchangedTarget, err)
	}
	if unchanged.MergeCommit != "" || unchanged.SourceAppliedAt != nil {
		t.Fatalf("unchanged repository produced a delivery result: %#v", unchanged)
	}
}

// 用户在源仓自己提交了东西之后再交付：交付基线取当前源仓状态，
// 用户的提交必须被纳入交付，而不是让交付失败或把它丢掉。
func TestCodeDeliveryCompletesWhenSourceRepositoryAdvanced(t *testing.T) {
	session, sourceDirs := deliverCodeMultiRepositorySession(t, 612)
	advanced := commitCodeTestFile(t, sourceDirs[0], "local-work.txt", "local work\n")

	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil {
		t.Fatalf("advanced source repository must not fail delivery: %v (%#v)", err, result)
	}

	stored := assertAllCodeRepositoriesCompleted(t, session.ID)
	for index := range stored {
		repository := &stored[index]
		if filepath.Clean(repository.SourceDir) != filepath.Clean(sourceDirs[0]) {
			continue
		}
		if _, err := runCodeGit(
			repository.SourceDir, "merge-base", "--is-ancestor", advanced, repository.MergeCommit,
		); err != nil {
			t.Fatalf("delivery must build on the user's local commit %s: %v", advanced, err)
		}
	}
	assertCodeDeliveryPushableToRemote(t, session)
}

// 源仓被切到别的分支时，交付读取的是目标分支的 ref，不受当前签出影响。
func TestCodeDeliveryCompletesWhenSourceRepositorySwitchedBranch(t *testing.T) {
	session, sourceDirs := deliverCodeMultiRepositorySession(t, 613)
	if _, err := runCodeGit(sourceDirs[0], "checkout", "-b", "feature/side-quest"); err != nil {
		t.Fatal(err)
	}

	result, err := resumeCodeMultiRepositoryDelivery(session, session.UserID)
	if err != nil {
		t.Fatalf("switched source branch must not fail delivery: %v (%#v)", err, result)
	}

	assertAllCodeRepositoriesCompleted(t, session.ID)
	branch, err := runCodeGit(sourceDirs[0], "branch", "--show-current")
	if err != nil || strings.TrimSpace(branch) != "feature/side-quest" {
		t.Fatalf("delivery must not switch the user's branch back: %q, %v", branch, err)
	}
	assertCodeDeliveryPushableToRemote(t, session)
}
