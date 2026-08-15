package api

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
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

func TestClaudeConfigurationUsesAuthHealthCheck(t *testing.T) {
	binDir := t.TempDir()
	commandPath := filepath.Join(binDir, "claude")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nif [ \"$1\" = auth ]; then printf '{\"loggedIn\":true}'; else printf '2.1.0'; fi\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	definition, err := getCodeExecutorDefinition("claude")
	if err != nil {
		t.Fatal(err)
	}
	if status := detectCodeExecutor(definition); !status.Available || !status.Configured {
		t.Fatalf("authenticated Claude was not configured: %#v", status)
	}
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nif [ \"$1\" = auth ]; then printf '{\"loggedIn\":false}'; else printf '2.1.0'; fi\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if status := detectCodeExecutor(definition); !status.Available || status.Configured {
		t.Fatalf("logged-out Claude status is wrong: %#v", status)
	}
}

func TestClaudeDefaultConnectionIsNotBlockedByAuthDetection(t *testing.T) {
	if err := validateCodeExecutorConfigured("claude", nil); err != nil {
		t.Fatalf("Claude default connection should remain user-selectable: %v", err)
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
		"codex":    {"-c", codexNetworkConfig, "-c", codexDisableDocsMCP, "--ask-for-approval", "on-request", "--sandbox", "workspace-write", "exec", "--json", "--skip-git-repo-check", prompt},
		"opencode": {"run", "--format", "json", "--dangerously-skip-permissions", prompt},
	}
	for executorID, expected := range tests {
		t.Run(executorID, func(t *testing.T) {
			args, _, err := buildCodeExecutorArgs(executorID, prompt, "", 42, codeApprovalPolicySafeAuto)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(args, expected) {
				t.Fatalf("unexpected args: %#v", args)
			}
			for index, arg := range args {
				validCodexConfig := executorID == "codex" && arg == "-c" && index+1 < len(args) && (args[index+1] == codexNetworkConfig || args[index+1] == codexDisableDocsMCP)
				if arg == "sh" || arg == "-c" && !validCodexConfig {
					t.Fatalf("shell execution is not allowed: %#v", args)
				}
			}
		})
	}
}

func TestBuildCodeExecutorArgsResumesNativeSession(t *testing.T) {
	prompt := "continue"
	tests := map[string][]string{
		"codex":    {"-c", codexNetworkConfig, "-c", codexDisableDocsMCP, "--ask-for-approval", "on-request", "--sandbox", "workspace-write", "exec", "resume", "--json", "--skip-git-repo-check", "native-1", prompt},
		"claude":   {"--print", "--output-format", "json", "--permission-mode", "acceptEdits", "--resume", "native-1", prompt},
		"opencode": {"run", "--format", "json", "--dangerously-skip-permissions", "--session", "native-1", prompt},
	}
	for executorID, expected := range tests {
		t.Run(executorID, func(t *testing.T) {
			args, nativeSessionID, err := buildCodeExecutorArgs(executorID, prompt, "native-1", 42, codeApprovalPolicySafeAuto)
			if err != nil {
				t.Fatal(err)
			}
			if nativeSessionID != "native-1" || !reflect.DeepEqual(args, expected) {
				t.Fatalf("unexpected resume args: %s %#v", nativeSessionID, args)
			}
		})
	}
}

func TestBuildExecutorArgsMapsApprovalPolicies(t *testing.T) {
	tests := []struct {
		executorID string
		policy     string
		expected   []string
	}{
		{executorID: "claude", policy: codeApprovalPolicyManual, expected: []string{"--permission-mode", "manual"}},
		{executorID: "claude", policy: codeApprovalPolicySafeAuto, expected: []string{"--permission-mode", "acceptEdits"}},
		{executorID: "claude", policy: codeApprovalPolicyFullAuto, expected: []string{"--dangerously-skip-permissions"}},
		{executorID: "opencode", policy: codeApprovalPolicyFullAuto, expected: []string{"--dangerously-skip-permissions"}},
	}
	for _, test := range tests {
		t.Run(test.executorID+"_"+test.policy, func(t *testing.T) {
			args, _, err := buildCodeExecutorArgs(test.executorID, "run", "", 42, test.policy)
			if err != nil {
				t.Fatal(err)
			}
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, strings.Join(test.expected, " ")) {
				t.Fatalf("approval args missing from %#v", args)
			}
		})
	}
}

func TestExecutorCapabilitiesExposeNativeTerminalAndApprovals(t *testing.T) {
	for _, executorID := range []string{"codex", "claude", "opencode", "aider"} {
		definition, err := getCodeExecutorDefinition(executorID)
		if err != nil || !definition.NativeTerminal || len(definition.ApprovalPolicies) == 0 {
			t.Fatalf("incomplete %s capabilities: %#v, %v", executorID, definition, err)
		}
		if !supportsNativeCodeTerminal(executorID) {
			t.Fatalf("%s native terminal is unavailable", executorID)
		}
	}
}

func TestBuildAiderExecutorArgsUsesSessionHistory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	args, nativeSessionID, err := buildCodeExecutorArgs("aider", "continue", "", 42, codeApprovalPolicySafeAuto)
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

func TestBuildCodeExecutorCommandAddsWorktreeGitWritableDirs(t *testing.T) {
	withAIProjectBaseDir(t)
	repositoryDir := createCodeGitRepository(t)
	session := &model.AIDevSession{ID: 35, UserID: 7, ProjectID: 9, ApprovalPolicy: codeApprovalPolicySafeAuto}
	if err := createCodeSessionWorktree(session, &model.AIProject{SourceDirs: []string{repositoryDir}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackCodeSessionWorktree(session) })
	writableDirs, err := codexWritableDirsForSession(session)
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := buildCodeExecutorCommand(context.Background(), "codex", session.WorkDir, "inspect only", "", session.ID, session)
	if err != nil {
		t.Skipf("codex is not installed: %v", err)
	}
	assertCodexWritableDirArgs(t, command.Args, writableDirs)
}
