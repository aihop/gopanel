package api

import (
	"os"
	"path/filepath"
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

	result, err := inspectCodeProjectBranches(&model.AIGroup{SourceDirs: []string{repositoryDir}})
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
