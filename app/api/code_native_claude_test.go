package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestNativeClaudeSessionExistsUsesConfiguredStorage(t *testing.T) {
	configDir := t.TempDir()
	nativeSessionID := "existing-session"
	projectDir := filepath.Join(configDir, "projects", "project")
	if err := os.MkdirAll(projectDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, nativeSessionID+".jsonl"), []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	env := []string{"CLAUDE_CONFIG_DIR=" + configDir}
	if !nativeClaudeSessionExists(nativeSessionID, env) {
		t.Fatal("expected existing Claude session to be found")
	}
	if nativeClaudeSessionExists("missing-session", env) {
		t.Fatal("unexpected missing Claude session")
	}
	if err := os.WriteFile(filepath.Join(projectDir, "empty-session.jsonl"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if nativeClaudeSessionExists("empty-session", env) {
		t.Fatal("unexpected empty Claude session")
	}
}

func TestBuildNativeClaudeCommandReplacesInvalidSession(t *testing.T) {
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("HOME", t.TempDir())
	command, nativeSessionID, err := buildNativeCodeCommand(&model.AIDevSession{
		AgentName: "claude", WorkDir: t.TempDir(), NativeSessionID: "missing-session",
		ApprovalPolicy: codeApprovalPolicySafeAuto,
	})
	if err != nil {
		t.Fatal(err)
	}
	if nativeSessionID == "" || nativeSessionID == "missing-session" {
		t.Fatalf("invalid Claude session was not replaced: %q", nativeSessionID)
	}
	if containsString(command.Args, "--resume") || !containsString(command.Args, "--session-id") {
		t.Fatalf("unexpected Claude recovery args: %#v", command.Args)
	}
}
