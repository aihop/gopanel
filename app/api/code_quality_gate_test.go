package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectNodeQualityChecksUsesLockfileManager(t *testing.T) {
	workDir := t.TempDir()
	packageJSON := `{"scripts":{"test":"vitest","lint":"eslint .","type-check":"vue-tsc","build":"vite build"}}`
	if err := os.WriteFile(filepath.Join(workDir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "pnpm-lock.yaml"), []byte("lockfileVersion: 9"), 0600); err != nil {
		t.Fatal(err)
	}
	checks := detectNodeQualityChecks(workDir, workDir)
	if len(checks) != 4 {
		t.Fatalf("expected 4 checks, got %d", len(checks))
	}
	if checks[0].Command != "pnpm run test" || checks[2].Command != "pnpm run type-check" {
		t.Fatalf("unexpected commands: %#v", checks)
	}
	for _, check := range checks {
		if check.workDirPath != workDir || check.WorkDir != filepath.Base(workDir) {
			t.Fatalf("unexpected work directory: %#v", check)
		}
	}
}

func TestDetectCodeQualityChecksAtSupportsGo(t *testing.T) {
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module example.com/test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	checks := detectCodeQualityChecksAt(workDir, workDir)
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
	if checks[0].Command != "go test ./..." || checks[1].Command != "go build ./..." {
		t.Fatalf("unexpected Go checks: %#v", checks)
	}
}

func TestDetectCodeQualityChecksAtSupportsFlutter(t *testing.T) {
	workDir := t.TempDir()
	pubspec := "name: example\ndependencies:\n  flutter:\n    sdk: flutter\n"
	if err := os.WriteFile(filepath.Join(workDir, "pubspec.yaml"), []byte(pubspec), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workDir, "test"), 0700); err != nil {
		t.Fatal(err)
	}
	checks := detectCodeQualityChecksAt(workDir, workDir)
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
	if checks[0].Command != "flutter analyze" || checks[1].Command != "flutter test" {
		t.Fatalf("unexpected Flutter checks: %#v", checks)
	}
	if checks[0].Label != "Flutter analyze" || checks[1].Label != "Flutter test" {
		t.Fatalf("unexpected Flutter labels: %#v", checks)
	}
}

func TestDetectCodeQualityChecksAtSupportsDartPackage(t *testing.T) {
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "pubspec.yaml"), []byte("name: example\n"), 0600); err != nil {
		t.Fatal(err)
	}
	checks := detectCodeQualityChecksAt(workDir, workDir)
	if len(checks) != 1 || checks[0].Command != "dart analyze" {
		t.Fatalf("unexpected Dart checks: %#v", checks)
	}
}

func TestTruncateCodeQualityOutputKeepsHeadAndTail(t *testing.T) {
	output := "begin-" + string(make([]byte, 128)) + "-end"
	truncated, changed := truncateCodeQualityOutput(output, 64)
	if !changed || len(truncated) > 110 {
		t.Fatalf("expected bounded output, got %d bytes", len(truncated))
	}
	if truncated[:5] != "begin" || truncated[len(truncated)-3:] != "end" {
		t.Fatalf("head or tail missing: %q", truncated)
	}
}

func TestCodeQualityFailureSummaryKeepsUsefulTail(t *testing.T) {
	summary := codeQualityFailureSummary("setup\n\nfirst failure\nsecond failure\nthird failure\nfourth failure\nfifth failure")
	if strings.Contains(summary, "first failure") || !strings.Contains(summary, "fifth failure") {
		t.Fatalf("unexpected failure summary: %q", summary)
	}
}

func TestCodeQualityFailureSummaryFindsGoTestFailureBeforeNoTestPackages(t *testing.T) {
	output := `--- FAIL: TestDelivery (0.01s)
    delivery_test.go:42: expected main to advance
FAIL
FAIL github.com/aihop/gopanel/app/api 0.12s
? github.com/aihop/gopanel/utils/toolbox [no test files]
? github.com/aihop/gopanel/utils/websocket [no test files]
FAIL`
	summary := codeQualityFailureSummary(output)
	if !strings.Contains(summary, "TestDelivery") || !strings.Contains(summary, "expected main to advance") ||
		strings.Contains(summary, "no test files") {
		t.Fatalf("unexpected Go test failure summary: %q", summary)
	}
}
