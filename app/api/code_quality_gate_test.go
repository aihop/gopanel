package api

import (
	"os"
	"path/filepath"
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
