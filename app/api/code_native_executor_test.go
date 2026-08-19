package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func TestBuildNativeCodeCommandSupportsInstalledExecutors(t *testing.T) {
	binDir := t.TempDir()
	for _, name := range []string{"grok", "claude", "opencode", "aider"} {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", binDir)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	claudeProjectDir := filepath.Join(homeDir, ".claude", "projects", "project")
	if err := os.MkdirAll(claudeProjectDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeProjectDir, "native-1.jsonl"), []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	grokSessionDir := filepath.Join(homeDir, ".grok", "sessions", "%2Fworkspace", "native-0")
	if err := os.MkdirAll(grokSessionDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(grokSessionDir, "summary.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		executorID      string
		nativeSessionID string
		policy          string
		expectedArgs    []string
	}{
		{executorID: "grok", nativeSessionID: "native-0", policy: codeApprovalPolicySafeAuto, expectedArgs: []string{"--no-auto-update", "--permission-mode", "auto", "--resume", "native-0"}},
		{executorID: "claude", nativeSessionID: "native-1", policy: codeApprovalPolicySafeAuto, expectedArgs: []string{"--permission-mode", "acceptEdits", "--resume", "native-1"}},
		{executorID: "opencode", nativeSessionID: "native-2", policy: codeApprovalPolicyFullAuto, expectedArgs: []string{"--session", "native-2"}},
		{executorID: "aider", nativeSessionID: "native-3", policy: codeApprovalPolicyFullAuto},
	}
	for _, test := range tests {
		t.Run(test.executorID, func(t *testing.T) {
			command, nativeSessionID, err := buildNativeCodeCommand(&model.AIDevSession{
				ID: 7, AgentName: test.executorID, WorkDir: workDir,
				NativeSessionID: test.nativeSessionID, ApprovalPolicy: test.policy,
			})
			if err != nil {
				t.Fatal(err)
			}
			if nativeSessionID != test.nativeSessionID || command.Dir != workDir {
				t.Fatalf("unexpected native command metadata: %q %q", nativeSessionID, command.Dir)
			}
			if len(test.expectedArgs) > 0 && !reflect.DeepEqual(command.Args[1:], test.expectedArgs) {
				t.Fatalf("unexpected args: %#v", command.Args)
			}
			if test.executorID == "aider" {
				joined := strings.Join(command.Args, " ")
				if !strings.Contains(joined, "--yes-always") || !strings.Contains(joined, "native-3.chat.md") {
					t.Fatalf("unexpected Aider args: %#v", command.Args)
				}
			}
			if test.executorID == "opencode" {
				config := environmentValue(command.Env, "OPENCODE_CONFIG_CONTENT")
				if !strings.Contains(config, `"permission":"allow"`) {
					t.Fatalf("OpenCode permissions were not configured: %s", config)
				}
			}
		})
	}
}

func TestBuildNativeGrokCommandRecreatesMissingSession(t *testing.T) {
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "grok"), []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("HOME", t.TempDir())
	command, nativeSessionID, err := buildNativeCodeCommand(&model.AIDevSession{
		ID: 8, AgentName: "grok", WorkDir: t.TempDir(), NativeSessionID: "missing-session",
	})
	if err != nil {
		t.Fatal(err)
	}
	if nativeSessionID != "missing-session" || !strings.Contains(strings.Join(command.Args, " "), "--session-id missing-session") {
		t.Fatalf("missing Grok session should be recreated: %q %#v", nativeSessionID, command.Args)
	}
}

func TestNativeOpenCodePermissionsPreserveProviderConfig(t *testing.T) {
	command := exec.Command("opencode")
	command.Env = []string{`OPENCODE_CONFIG_CONTENT={"model":"gopanel_session/test"}`}
	if err := configureNativeOpenCodePermissions(command); err != nil {
		t.Fatal(err)
	}
	config := environmentValue(command.Env, "OPENCODE_CONFIG_CONTENT")
	if !strings.Contains(config, `"model":"gopanel_session/test"`) || !strings.Contains(config, `"permission":"allow"`) {
		t.Fatalf("unexpected merged OpenCode config: %s", config)
	}
}

func TestFindNativeOpenCodeSessionIDMatchesNewSessionInWorkingDirectory(t *testing.T) {
	startedAt := time.Now()
	workDir := t.TempDir()
	output := []byte(`[
		{"id":"old","directory":"` + workDir + `","time_created":` + strconv.FormatInt(startedAt.Add(-time.Minute).UnixMilli(), 10) + `},
		{"id":"other","directory":"/other/project","time_created":` + strconv.FormatInt(startedAt.UnixMilli(), 10) + `},
		{"id":"new","directory":"` + workDir + `","time_created":` + strconv.FormatInt(startedAt.Add(time.Second).UnixMilli(), 10) + `}
	]`)
	if actual := findNativeOpenCodeSessionID(output, workDir, startedAt); actual != "new" {
		t.Fatalf("OpenCode session ID = %q", actual)
	}
	if actual := findNativeOpenCodeSessionID([]byte(`not-json`), workDir, startedAt); actual != "" {
		t.Fatalf("invalid output returned session ID %q", actual)
	}
}

func TestBuildNativeClaudeCommandAllocatesSessionID(t *testing.T) {
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	command, nativeSessionID, err := buildNativeCodeCommand(&model.AIDevSession{
		AgentName: "claude", WorkDir: t.TempDir(), ApprovalPolicy: codeApprovalPolicyManual,
	})
	if err != nil {
		t.Fatal(err)
	}
	if nativeSessionID == "" || !strings.Contains(strings.Join(command.Args, " "), "--session-id "+nativeSessionID) {
		t.Fatalf("Claude session ID was not allocated: %q %#v", nativeSessionID, command.Args)
	}
}
