package api

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestParseCodeQualityCommandPreservesQuotedArguments(t *testing.T) {
	parts, err := parseCodeQualityCommand(`go test "./folder with spaces/..."`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"go", "test", "./folder with spaces/..."}
	if !reflect.DeepEqual(parts, want) {
		t.Fatalf("parts=%#v want=%#v", parts, want)
	}
}

func TestParseCodeQualityCommandRejectsShellOperators(t *testing.T) {
	for _, command := range []string{"npm test && npm run build", "go test ./... | tee output"} {
		if _, err := parseCodeQualityCommand(command); err == nil {
			t.Fatalf("shell operator was accepted: %q", command)
		}
	}
}

func TestNormalizeCodeProjectQualityChecksRejectsEscapingWorkDir(t *testing.T) {
	repository := createCodeGitRepository(t)
	checks := []model.AIProjectQualityCheck{{
		Name: "test", Kind: "test", Repository: repository, WorkDir: "../outside", Command: "go test ./...",
	}}
	if _, err := normalizeCodeProjectQualityChecks(checks, []string{repository}); err == nil || !strings.Contains(err.Error(), "仓库内") {
		t.Fatalf("unexpected escaping workdir result: %v", err)
	}
}

func TestConfiguredCodeQualityCheckUsesDeliveryRoot(t *testing.T) {
	sourceDir := createCodeGitRepository(t)
	deliveryDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(sourceDir, "api"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(deliveryDir, "api"), 0700); err != nil {
		t.Fatal(err)
	}
	configured := []model.AIProjectQualityCheck{{
		Name: "API tests", Kind: "test", Repository: sourceDir, WorkDir: "api", Command: "go test ./...",
	}}
	checks := configuredCodeQualityChecks(configured, []codeDeliveryQualityRoot{{
		WorkDir: deliveryDir, IdentityDir: sourceDir, RuntimeDir: sourceDir,
	}})
	expectedWorkDir, err := filepath.EvalSymlinks(filepath.Join(deliveryDir, "api"))
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 1 || checks[0].workDirPath != expectedWorkDir {
		t.Fatalf("unexpected configured checks: %#v", checks)
	}
}

func TestResolveCodeQualityCommandAllowsDetectedFlutterToolchain(t *testing.T) {
	toolchainDir := t.TempDir()
	flutter := filepath.Join(toolchainDir, "flutter")
	if err := os.WriteFile(flutter, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	check := newCodeQualityCheck("lint", "Flutter analyze", t.TempDir(), t.TempDir(), flutter, "analyze")
	resolved, _, err := resolveCodeQualityCommand(check)
	if err != nil || resolved != flutter {
		t.Fatalf("detected Flutter toolchain was rejected: %q, %v", resolved, err)
	}
}
