package api

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeProjectRepositoryOptionsUseDiscoveredPaths(t *testing.T) {
	first := createCodeGitRepository(t)
	second := createCodeGitRepository(t)
	options := codeProjectRepositoryOptions([]codeRepositoryCandidate{
		{SourceDir: first},
		{SourceDir: second},
	})
	if len(options) != 2 || options[0].Path != first || options[0].Name != filepath.Base(first) {
		t.Fatalf("options = %#v", options)
	}
}

func TestNormalizeCodeDeliveryPolicyExplainsUnavailableRepository(t *testing.T) {
	repository := createCodeGitRepository(t)
	missing := filepath.Join(t.TempDir(), "removed")
	_, err := normalizeCodeDeliveryPolicyWithCandidates(
		[]string{repository},
		[]codeRepositoryCandidate{{SourceDir: repository}},
		missing,
		"main",
	)
	if err == nil || !strings.Contains(err.Error(), "重新选择项目内的 Git 仓库") {
		t.Fatalf("error = %v", err)
	}
}
