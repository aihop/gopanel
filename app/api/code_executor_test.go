package api

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
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
		"codex":    {"--ask-for-approval", "never", "--sandbox", "workspace-write", "exec", "--json", "--skip-git-repo-check", prompt},
		"opencode": {"run", "--format", "json", prompt},
	}
	for executorID, expected := range tests {
		t.Run(executorID, func(t *testing.T) {
			args, _, err := buildCodeExecutorArgs(executorID, prompt, "", 42)
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

func TestBuildCodeExecutorArgsResumesNativeSession(t *testing.T) {
	prompt := "continue"
	tests := map[string][]string{
		"codex":    {"--ask-for-approval", "never", "--sandbox", "workspace-write", "exec", "resume", "--json", "--skip-git-repo-check", "native-1", prompt},
		"claude":   {"--print", "--permission-mode", "acceptEdits", "--output-format", "json", "--resume", "native-1", prompt},
		"opencode": {"run", "--format", "json", "--session", "native-1", prompt},
	}
	for executorID, expected := range tests {
		t.Run(executorID, func(t *testing.T) {
			args, nativeSessionID, err := buildCodeExecutorArgs(executorID, prompt, "native-1", 42)
			if err != nil {
				t.Fatal(err)
			}
			if nativeSessionID != "native-1" || !reflect.DeepEqual(args, expected) {
				t.Fatalf("unexpected resume args: %s %#v", nativeSessionID, args)
			}
		})
	}
}

func TestBuildAiderExecutorArgsUsesSessionHistory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	args, nativeSessionID, err := buildCodeExecutorArgs("aider", "continue", "", 42)
	if err != nil {
		t.Fatal(err)
	}
	if nativeSessionID != "gopanel-42" {
		t.Fatalf("unexpected native session ID: %s", nativeSessionID)
	}
	joinedArgs := strings.Join(args, " ")
	if !strings.Contains(joinedArgs, filepath.Join(".gopanel", "code", "aider", "gopanel-42.chat.md")) ||
		!strings.Contains(joinedArgs, filepath.Join(".gopanel", "code", "aider", "gopanel-42.llm.log")) {
		t.Fatalf("expected isolated aider history files: %#v", args)
	}
	if strings.Contains(joinedArgs, "--restore-chat-history") {
		t.Fatalf("new aider session should not restore missing history: %#v", args)
	}
}

func TestBuildCodeExecutorCommandUsesWorkDir(t *testing.T) {
	command, _, err := buildCodeExecutorCommand(context.Background(), "codex", t.TempDir(), "inspect only", "", 1, nil)
	if err != nil {
		t.Skipf("codex is not installed: %v", err)
	}
	if command.Dir == "" {
		t.Fatal("expected command working directory")
	}
}
