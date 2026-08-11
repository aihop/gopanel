package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func codeProjectForRepository(t *testing.T, repository string) *model.AIProject {
	t.Helper()
	branch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	return &model.AIProject{
		ID: 1, CreatorID: 1, SourceDirs: []string{repository},
		PrimaryRepository: repository, DeliveryBranch: branch,
	}
}

func TestSyncCodeProjectFastForwardsRemoteChanges(t *testing.T) {
	withCodeGovernanceDB(t)
	local, remote := createCodeRemoteRepository(t)
	updater := cloneCodeRepository(t, remote)
	remoteCommit := commitCodeTestFile(t, updater, "remote.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	result, err := syncCodeProject(codeProjectForRepository(t, local), false)
	if err != nil || result.Status != "synced" || !result.Updated || len(result.Repositories) != 1 {
		t.Fatalf("unexpected sync result: %#v, %v", result, err)
	}
	localCommit, err := runCodeGit(local, "rev-parse", "HEAD")
	if err != nil || localCommit != remoteCommit {
		t.Fatalf("local repository was not fast-forwarded: %s, %v", localCommit, err)
	}
}

func TestSyncCodeProjectDoesNotModifyDirtyRepository(t *testing.T) {
	withCodeGovernanceDB(t)
	local, remote := createCodeRemoteRepository(t)
	before, _ := runCodeGit(local, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(local, "dirty.txt"), []byte("dirty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	updater := cloneCodeRepository(t, remote)
	commitCodeTestFile(t, updater, "remote.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	status, err := inspectCodeProjectSync(codeProjectForRepository(t, local))
	if err != nil || status.Status != "dirty" || !status.CanSync {
		t.Fatalf("dirty repository must allow a safe sync attempt: %#v, %v", status, err)
	}

	result, err := syncCodeProject(codeProjectForRepository(t, local), false)
	after, _ := runCodeGit(local, "rev-parse", "HEAD")
	if err != nil || result.Status != "dirty" || before != after {
		t.Fatalf("dirty repository changed: %#v before=%s after=%s err=%v", result, before, after, err)
	}
}

func TestSyncCodeProjectReconcilesWorktreeAlreadyMatchingRemote(t *testing.T) {
	withCodeGovernanceDB(t)
	local, remote := createCodeRemoteRepository(t)
	updater := cloneCodeRepository(t, remote)
	remoteCommit := commitCodeTestFile(t, updater, "delivered.txt", "delivered\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "delivered.txt"), []byte("delivered\n"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := syncCodeProject(codeProjectForRepository(t, local), false)
	if err != nil || result.Status != "synced" || !result.Updated {
		t.Fatalf("matching delivered worktree was not reconciled: %#v, %v", result, err)
	}
	localCommit, _ := runCodeGit(local, "rev-parse", "HEAD")
	status, _ := runCodeGit(local, "status", "--porcelain")
	if localCommit != remoteCommit || status != "" {
		t.Fatalf("local repository was not normalized: head=%s status=%q", localCommit, status)
	}
}

func TestSyncCodeProjectPreservesStagedChangesWhenWorktreeMatchesRemote(t *testing.T) {
	withCodeGovernanceDB(t)
	local, remote := createCodeRemoteRepository(t)
	updater := cloneCodeRepository(t, remote)
	commitCodeTestFile(t, updater, "delivered.txt", "delivered\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "staged.txt"), []byte("keep staged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(local, "add", "staged.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(local, "staged.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "delivered.txt"), []byte("delivered\n"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := syncCodeProject(codeProjectForRepository(t, local), false)
	if err != nil || result.Status != "dirty" {
		t.Fatalf("staged changes were not protected: %#v, %v", result, err)
	}
	staged, _ := runCodeGit(local, "show", ":staged.txt")
	if staged != "keep staged" {
		t.Fatalf("staged content changed: %q", staged)
	}
}

func TestSyncCodeProjectDoesNotModifyDivergedRepository(t *testing.T) {
	withCodeGovernanceDB(t)
	local, remote := createCodeRemoteRepository(t)
	localCommit := commitCodeTestFile(t, local, "local.txt", "local\n")
	updater := cloneCodeRepository(t, remote)
	commitCodeTestFile(t, updater, "remote.txt", "remote\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}

	result, err := syncCodeProject(codeProjectForRepository(t, local), false)
	after, _ := runCodeGit(local, "rev-parse", "HEAD")
	if err != nil || result.Status != "diverged" || after != localCommit {
		t.Fatalf("diverged repository changed: %#v head=%s err=%v", result, after, err)
	}
}

func TestCodeProjectRepositorySpecsFindNestedMultiDirectoryRepositories(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "workspace-a", "apps", "api")
	second := filepath.Join(root, "workspace-b", "services", "worker")
	for _, repository := range []string{first, second} {
		if err := os.MkdirAll(repository, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodeGit(repository, "init"); err != nil {
			t.Fatal(err)
		}
		commitCodeTestFile(t, repository, "README.md", "test\n")
	}
	project := &model.AIProject{SourceDirs: []string{filepath.Join(root, "workspace-a"), filepath.Join(root, "workspace-b")}}
	specs, err := codeProjectRepositorySpecs(project)
	if err != nil || len(specs) != 2 {
		t.Fatalf("nested repositories were not discovered: %#v, %v", specs, err)
	}
}

func TestCodeProjectRepositorySpecsUseConfiguredBranchWhenDetached(t *testing.T) {
	repository := createCodeGitRepository(t)
	branch, err := runCodeGit(repository, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "checkout", "--detach", "HEAD"); err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{
		SourceDirs: []string{repository}, PrimaryRepository: repository, DeliveryBranch: branch,
	}
	specs, err := codeProjectRepositorySpecs(project)
	if err != nil || len(specs) != 1 || specs[0].Branch != branch {
		t.Fatalf("configured detached repository was rejected: %#v, %v", specs, err)
	}
}

func TestValidateCodeProjectGitlinkTargetsRejectsPointerMismatch(t *testing.T) {
	parent, child := createGitlinkRepositoryTree(t)
	branch, _ := runCodeGit(parent, "branch", "--show-current")
	specs := []codeProjectRepositorySpec{
		{Path: parent, Name: "parent", Branch: branch},
		{Path: child, Name: "custom", ParentPath: parent, GitlinkPath: "themes/custom", Branch: branch},
	}
	if err := validateCodeProjectGitlinkTargets(specs); err != nil {
		t.Fatalf("matching gitlink was rejected: %v", err)
	}
	commitCodeTestFile(t, child, "next.txt", "next\n")
	if err := validateCodeProjectGitlinkTargets(specs); err == nil {
		t.Fatal("mismatched gitlink pointer should block project sync")
	}
}

func TestCodeRepositoryLeasesUseOwnerAndJobIdentity(t *testing.T) {
	withCodeGovernanceDB(t)
	acquired, err := acquireCodeRepositoryLeases("shared-runner", 101, []string{"repository-a", "repository-b"})
	if err != nil || !acquired {
		t.Fatalf("first lease failed: %v, %v", acquired, err)
	}
	acquired, err = acquireCodeRepositoryLeases("shared-runner", 102, []string{"repository-b", "repository-c"})
	if err != nil || acquired {
		t.Fatalf("different job reused the same owner lease: %v, %v", acquired, err)
	}
	if err := releaseCodeRepositoryLeases("shared-runner", []string{"repository-a", "repository-b"}); err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().Add(-time.Minute)
	lease := &model.AICodeDeliveryLease{RepositoryKey: "repository-c", JobID: 99, LeaseOwner: "old", LeaseExpiresAt: &expiresAt}
	if err := global.DB.Create(lease).Error; err != nil {
		t.Fatal(err)
	}
	acquired, err = acquireCodeRepositoryLeases("project-sync", 0, []string{"repository-c"})
	if err != nil || !acquired {
		t.Fatalf("expired lease was not recovered: %v, %v", acquired, err)
	}
}

func TestInspectCodeProjectSyncReportsActiveRepositoryLease(t *testing.T) {
	withCodeGovernanceDB(t)
	repository, _ := createCodeRemoteRepository(t)
	project := codeProjectForRepository(t, repository)
	specs, err := codeProjectRepositorySpecs(project)
	if err != nil {
		t.Fatal(err)
	}
	acquired, err := acquireCodeRepositoryLeases("delivery", 55, []string{specs[0].LeaseKey})
	if err != nil || !acquired {
		t.Fatalf("lease setup failed: %v, %v", acquired, err)
	}
	result, err := inspectCodeProjectSync(project)
	if err != nil || result.Status != "blocked" || result.CanSync {
		t.Fatalf("active lease was not reported: %#v, %v", result, err)
	}
}
