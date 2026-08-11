package api

import (
	"strings"
	"testing"
)

func TestCodeSessionWorktreeBranchUsesShortStableNames(t *testing.T) {
	if branch := codeSessionWorktreeBranch(96, ""); branch != "gopanel/code-96" {
		t.Fatalf("single repository branch = %q", branch)
	}
	if branch := codeSessionWorktreeBranch(96, "api service"); branch != "gopanel/code-96-api-service" {
		t.Fatalf("multi repository branch = %q", branch)
	}
}

func TestCodeSessionRepositoryBranchesDeduplicatesSafeAliases(t *testing.T) {
	sources := []aiProjectWorkspaceSource{
		{Path: "/first", LinkName: "api service"},
		{Path: "/second", LinkName: "api@service"},
	}
	branches := codeSessionRepositoryBranches(97, sources)
	if branches["/first"] != "gopanel/code-97-api-service" ||
		branches["/second"] != "gopanel/code-97-api-service-2" {
		t.Fatalf("unexpected repository branches: %#v", branches)
	}
}

func TestCodeWorktreeBranchAliasIsBounded(t *testing.T) {
	alias := codeWorktreeBranchAlias(strings.Repeat("a", codeWorktreeBranchAliasMaxRunes+20))
	if len([]rune(alias)) != codeWorktreeBranchAliasMaxRunes {
		t.Fatalf("alias length = %d", len([]rune(alias)))
	}
}

func TestIsCodeSessionWorktreeBranchAcceptsShortAndLegacyNames(t *testing.T) {
	for _, branch := range []string{
		"gopanel/code-98",
		"gopanel/code-98-api",
		"gopanel/code-98-1786388903",
		"gopanel/code-98-1786388903-1",
	} {
		if !isCodeSessionWorktreeBranch(branch, 98) {
			t.Fatalf("session branch %q was not recognized", branch)
		}
	}
	for _, branch := range []string{"gopanel/code-9", "gopanel/code-980", "feature/code-98"} {
		if isCodeSessionWorktreeBranch(branch, 98) {
			t.Fatalf("foreign branch %q was recognized", branch)
		}
	}
}
