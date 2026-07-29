package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/aihop/gopanel/constant"
)

func TestNormalizeCodeExecutorID(t *testing.T) {
	executorID, err := normalizeCodeExecutorID(" CoDeX ")
	if err != nil {
		t.Fatal(err)
	}
	if executorID != "codex" {
		t.Fatalf("unexpected executor ID: %s", executorID)
	}
	if _, err := normalizeCodeExecutorID("codex; id"); err == nil {
		t.Fatal("expected command-like executor ID to be rejected")
	}
}

func TestTerminalExecutorIsAlwaysAvailable(t *testing.T) {
	definition, err := getCodeExecutorDefinition("terminal")
	if err != nil {
		t.Fatal(err)
	}
	status := detectCodeExecutor(definition)
	if !status.Installed || !status.Available || !status.Configured {
		t.Fatalf("terminal should always be available: %#v", status)
	}
}

func TestMissingExecutorIsUnavailable(t *testing.T) {
	status := detectCodeExecutor(codeExecutorDefinition{
		ID:                  "missing",
		Name:                "Missing",
		Command:             "gopanel-executor-that-does-not-exist",
		AutomationSupported: true,
	})
	if status.Installed || status.Available || status.Reason == "" {
		t.Fatalf("missing executor should be unavailable: %#v", status)
	}
}

func TestSubAdminCannotUseHostExecutor(t *testing.T) {
	if _, err := validateCodeExecutorAvailable("codex", constant.UserRoleSubAdmin); err == nil {
		t.Fatal("sub-admin should not be allowed to use host executors")
	}
	executorID, err := validateCodeExecutorAvailable("terminal", constant.UserRoleSubAdmin)
	if err != nil || executorID != "terminal" {
		t.Fatalf("sub-admin terminal should remain available: %s, %v", executorID, err)
	}
}

func TestBuildCodeExecutorArgsPreservesPrompt(t *testing.T) {
	prompt := `"; touch /tmp/PWNED; echo "`
	tests := map[string][]string{
		"codex":    {"--ask-for-approval", "never", "--sandbox", "workspace-write", "exec", "--skip-git-repo-check", prompt},
		"claude":   {"--print", "--permission-mode", "acceptEdits", prompt},
		"opencode": {"run", prompt},
		"aider":    {"--yes-always", "--message", prompt},
	}
	for executorID, expected := range tests {
		t.Run(executorID, func(t *testing.T) {
			args, err := buildCodeExecutorArgs(executorID, prompt)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(args, expected) {
				t.Fatalf("unexpected args: %#v", args)
			}
			for _, arg := range args {
				if arg == "sh" || arg == "-c" {
					t.Fatalf("shell execution is not allowed: %#v", args)
				}
			}
		})
	}
}

func TestBuildCodeExecutorCommandUsesWorkDir(t *testing.T) {
	command, err := buildCodeExecutorCommand(context.Background(), "codex", t.TempDir(), "inspect only")
	if err != nil {
		t.Skipf("codex is not installed: %v", err)
	}
	if command.Dir == "" {
		t.Fatal("expected command working directory")
	}
}
