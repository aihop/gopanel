package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverCodeRepositoryInstallsLocalDSStoreExclude(t *testing.T) {
	repository := createCodeGitRepository(t)
	commonDir, err := runCodeGit(repository, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		t.Fatal(err)
	}
	excludePath := filepath.Join(commonDir, "info", "exclude")
	if err := os.WriteFile(excludePath, []byte("user-local.tmp\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repository, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(repository, ".DS_Store"), filepath.Join(repository, "nested", ".DS_Store")} {
		if err := os.WriteFile(path, []byte("finder metadata"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	for range 2 {
		candidates, discoverErr := discoverCodeRepositoryCandidates([]string{repository})
		if discoverErr != nil || len(candidates) != 1 || candidates[0].Dirty {
			t.Fatalf(".DS_Store should not dirty a Code repository: %#v, %v", candidates, discoverErr)
		}
	}
	content, err := os.ReadFile(excludePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "user-local.tmp\n") {
		t.Fatalf("existing local excludes were overwritten: %q", content)
	}
	if strings.Count(string(content), ".DS_Store") != 1 {
		t.Fatalf("Code exclude was not installed idempotently: %q", content)
	}
	ignored, err := runCodeGit(repository, "status", "--porcelain", "--untracked-files=all")
	if err != nil || strings.TrimSpace(ignored) != "" {
		t.Fatalf("ignored Finder files remained visible: %q, %v", ignored, err)
	}
}

func TestCodeLocalExcludeDoesNotHideTrackedDSStore(t *testing.T) {
	repository := createCodeGitRepository(t)
	if err := ensureCodeGitLocalExcludes(repository); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repository, ".DS_Store")
	if err := os.WriteFile(path, []byte("tracked"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "add", "-f", ".DS_Store"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repository, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "track finder file"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	candidates, err := discoverCodeRepositoryCandidates([]string{repository})
	if err != nil || len(candidates) != 1 || !candidates[0].Dirty {
		t.Fatalf("tracked .DS_Store changes must remain visible: %#v, %v", candidates, err)
	}
}
