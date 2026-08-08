package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestInspectCodeProjectBranchesReturnsLocalAndRemoteBranches(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if _, err := runCodeGit(repositoryDir, "branch", "feature/sidebar"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "update-ref", "refs/remotes/origin/main", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "changed.txt"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := inspectCodeProjectBranches(&model.AIProject{SourceDirs: []string{repositoryDir}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Repositories) != 1 || result.TotalBranches != 3 {
		t.Fatalf("unexpected project branches: %#v", result)
	}
	repository := result.Repositories[0]
	if repository.CurrentBranch == "" || !repository.Dirty || repository.ChangedFiles != 1 {
		t.Fatalf("unexpected repository state: %#v", repository)
	}
	scopes := map[string]int{}
	for _, branch := range repository.Branches {
		scopes[branch.Scope]++
		if branch.Additions != 1 || branch.Deletions != 0 {
			t.Fatalf("unexpected branch line stats: %#v", branch)
		}
	}
	if scopes["local"] != 2 || scopes["remote"] != 1 {
		t.Fatalf("unexpected branch scopes: %#v", scopes)
	}
}

func TestDiscoverCodeProjectBranchRepositoriesFindsNestedRepositories(t *testing.T) {
	projectDir := t.TempDir()
	first := filepath.Join(projectDir, "apps", "api")
	second := filepath.Join(projectDir, "services", "worker")
	for _, repositoryDir := range []string{first, second} {
		if err := os.MkdirAll(repositoryDir, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodeGit(repositoryDir, "init"); err != nil {
			t.Fatal(err)
		}
	}
	repositories, err := discoverCodeProjectBranchRepositories([]string{projectDir})
	if err != nil {
		t.Fatal(err)
	}
	if len(repositories) != 2 {
		t.Fatalf("discovered repositories = %#v", repositories)
	}
}

func TestCodeProjectBranchDeletionProtectsLifecycleBranches(t *testing.T) {
	database := withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)
	currentBranch, err := runCodeGit(repositoryDir, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{
		ID: 701, Name: "branches", CreatorID: 7, SourceDirs: []string{repositoryDir}, DeliveryBranch: currentBranch,
	}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "branch", "gopanel/code-701-session"); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AIDevSession{
		ID: 701, UserID: 7, ProjectID: project.ID, SourceWorkDir: repositoryDir,
		WorktreeBranch: "gopanel/code-701-session", Status: codeSessionStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	result, err := inspectCodeProjectBranches(project)
	if err != nil {
		t.Fatal(err)
	}
	reasons := map[string]string{}
	for _, branch := range result.Repositories[0].Branches {
		reasons[branch.Name] = branch.DeleteBlockReason
	}
	if reasons[currentBranch] != codeBranchDeleteBlockCurrent ||
		reasons["gopanel/code-701-session"] != codeBranchDeleteBlockSession {
		t.Fatalf("unexpected branch protection reasons: %#v", reasons)
	}
	if err := deleteCodeProjectLocalBranch(project, repositoryDir, currentBranch, true); err == nil {
		t.Fatal("current delivery branch deletion was not blocked")
	}
	if err := deleteCodeProjectLocalBranch(project, repositoryDir, "gopanel/code-701-session", true); err == nil {
		t.Fatal("active session branch deletion was not blocked")
	}
}

func TestCodeProjectBranchDeletionProtectsDeliveryAndWorktreeBranches(t *testing.T) {
	withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)
	deliveryBranch, err := runCodeGit(repositoryDir, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "branch", "feature/current"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "switch", "feature/current"); err != nil {
		t.Fatal(err)
	}
	worktreeDir := filepath.Join(t.TempDir(), "occupied")
	if _, err := runCodeGit(repositoryDir, "worktree", "add", "-b", "gopanel/code-702-worktree", worktreeDir, deliveryBranch); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = runCodeGit(repositoryDir, "worktree", "remove", "--force", worktreeDir) })
	project := &model.AIProject{SourceDirs: []string{repositoryDir}, DeliveryBranch: deliveryBranch}

	result, err := inspectCodeProjectBranches(project)
	if err != nil {
		t.Fatal(err)
	}
	reasons := map[string]string{}
	for _, branch := range result.Repositories[0].Branches {
		reasons[branch.Name] = branch.DeleteBlockReason
	}
	if reasons[deliveryBranch] != codeBranchDeleteBlockDelivery ||
		reasons["gopanel/code-702-worktree"] != codeBranchDeleteBlockWorktree {
		t.Fatalf("unexpected delivery/worktree protections: %#v", reasons)
	}
	if err := deleteCodeProjectLocalBranch(project, repositoryDir, deliveryBranch, true); err == nil {
		t.Fatal("delivery branch deletion was not blocked")
	}
	if err := deleteCodeProjectLocalBranch(project, repositoryDir, "gopanel/code-702-worktree", true); err == nil {
		t.Fatal("worktree branch deletion was not blocked")
	}
}

func TestCodeProjectBranchDeletionRequiresForceForUnmergedBranch(t *testing.T) {
	withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)
	currentBranch, err := runCodeGit(repositoryDir, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{SourceDirs: []string{repositoryDir}, DeliveryBranch: currentBranch}
	if _, err := runCodeGit(repositoryDir, "switch", "-c", "feature/unmerged"); err != nil {
		t.Fatal(err)
	}
	commitCodeTestFile(t, repositoryDir, "feature.txt", "feature\n")
	if _, err := runCodeGit(repositoryDir, "switch", currentBranch); err != nil {
		t.Fatal(err)
	}

	err = deleteCodeProjectLocalBranch(project, repositoryDir, "feature/unmerged", false)
	if err == nil || !strings.Contains(err.Error(), "尚未合并") {
		t.Fatalf("unmerged branch safe deletion error = %v", err)
	}
	if _, err := runCodeGit(repositoryDir, "show-ref", "--verify", "refs/heads/feature/unmerged"); err != nil {
		t.Fatalf("safe deletion removed unmerged branch: %v", err)
	}
	if err := deleteCodeProjectLocalBranch(project, repositoryDir, "feature/unmerged", true); err != nil {
		t.Fatalf("force delete unmerged branch: %v", err)
	}
	if _, err := runCodeGit(repositoryDir, "show-ref", "--verify", "refs/heads/feature/unmerged"); err == nil {
		t.Fatal("force deletion retained unmerged branch")
	}
}

func TestCodeProjectBranchDeletionRemovesMergedBranchAndRejectsForeignRepository(t *testing.T) {
	withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)
	currentBranch, err := runCodeGit(repositoryDir, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{SourceDirs: []string{repositoryDir}, DeliveryBranch: currentBranch}
	if _, err := runCodeGit(repositoryDir, "branch", "feature/merged"); err != nil {
		t.Fatal(err)
	}
	if err := deleteCodeProjectLocalBranch(project, repositoryDir, "feature/merged", false); err != nil {
		t.Fatalf("delete merged branch: %v", err)
	}
	foreign := createCodeGitRepository(t)
	if err := deleteCodeProjectLocalBranch(project, foreign, currentBranch, true); err == nil ||
		!strings.Contains(err.Error(), "不属于") {
		t.Fatalf("foreign repository error = %v", err)
	}
}

func TestCodeProjectBranchDeletionUsesRepositoryLease(t *testing.T) {
	withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)
	currentBranch, err := runCodeGit(repositoryDir, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{SourceDirs: []string{repositoryDir}, DeliveryBranch: currentBranch}
	if _, err := runCodeGit(repositoryDir, "branch", "feature/leased"); err != nil {
		t.Fatal(err)
	}
	specs, err := codeProjectRepositorySpecs(project)
	if err != nil || len(specs) != 1 {
		t.Fatalf("repository specs: %#v, %v", specs, err)
	}
	owner := newCodeRepositoryLeaseOwner("branch-test")
	acquired, err := acquireCodeRepositoryLeases(owner, 0, []string{specs[0].LeaseKey})
	if err != nil || !acquired {
		t.Fatalf("acquire repository lease: %v, %v", acquired, err)
	}
	defer func() { _ = releaseCodeRepositoryLeases(owner, []string{specs[0].LeaseKey}) }()

	err = deleteCodeProjectLocalBranch(project, repositoryDir, "feature/leased", false)
	if err == nil || !strings.Contains(err.Error(), "正在同步或交付") {
		t.Fatalf("busy repository deletion error = %v", err)
	}
	if _, err := runCodeGit(repositoryDir, "show-ref", "--verify", "refs/heads/feature/leased"); err != nil {
		t.Fatalf("busy deletion removed branch: %v", err)
	}
}

func TestCodeProjectBranchMergedStateUsesDeliveryBranch(t *testing.T) {
	withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)
	deliveryBranch, err := runCodeGit(repositoryDir, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "branch", "feature/merged", deliveryBranch); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "switch", "-c", "feature/current"); err != nil {
		t.Fatal(err)
	}
	commitCodeTestFile(t, repositoryDir, "current.txt", "current\n")
	project := &model.AIProject{
		SourceDirs: []string{repositoryDir}, PrimaryRepository: repositoryDir, DeliveryBranch: deliveryBranch,
	}

	result, err := inspectCodeProjectBranches(project)
	if err != nil {
		t.Fatal(err)
	}
	merged := map[string]bool{}
	for _, branch := range result.Repositories[0].Branches {
		merged[branch.Name] = branch.Merged
	}
	if !merged["feature/merged"] || merged["feature/current"] {
		t.Fatalf("merged state was not based on delivery branch %s: %#v", deliveryBranch, merged)
	}
	if err := deleteCodeProjectLocalBranch(project, repositoryDir, "feature/merged", false); err != nil {
		t.Fatalf("delete branch merged into non-current delivery branch: %v", err)
	}
}
