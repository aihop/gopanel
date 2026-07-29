package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func TestBuildNativeCodexCommandStartsAndResumesInteractiveSession(t *testing.T) {
	workDir := t.TempDir()
	newSession := &model.AIDevSession{WorkDir: workDir, ApprovalPolicy: codeApprovalPolicySafeAuto}
	command, err := buildNativeCodexCommand(newSession)
	if err != nil {
		t.Skipf("codex is not installed: %v", err)
	}
	expected := []string{
		command.Path,
		"--ask-for-approval", "on-request",
		"--sandbox", "workspace-write",
		"--no-alt-screen",
		"--cd", workDir,
	}
	if !reflect.DeepEqual(command.Args, expected) || command.Dir != workDir {
		t.Fatalf("unexpected native command: dir=%s args=%#v", command.Dir, command.Args)
	}

	newSession.NativeSessionID = "native-session"
	command, err = buildNativeCodexCommand(newSession)
	if err != nil {
		t.Fatal(err)
	}
	if got := command.Args[len(command.Args)-2:]; !reflect.DeepEqual(got, []string{"resume", "native-session"}) {
		t.Fatalf("unexpected resume args: %#v", command.Args)
	}
}

func TestFindNativeCodexSessionIDMatchesWorkingDirectory(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	workDir := t.TempDir()
	startedAt := time.Now().Add(-time.Second)
	sessionDir := filepath.Join(homeDir, ".codex", "sessions", "2026", "07", "29")
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sessionDir, "rollout.jsonl")
	payload := map[string]any{
		"type": "session_meta",
		"payload": map[string]any{
			"session_id": "native-session",
			"cwd":        workDir,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if got := findNativeCodexSessionID(workDir, 0, startedAt); got != "native-session" {
		t.Fatalf("native session ID = %q", got)
	}
}

func TestNativeCodeTerminalKeepsBoundedReconnectHistory(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.publish([]byte(strings.Repeat("a", nativeTerminalHistoryLimit)))
	terminal.publish([]byte("tail"))
	subscription, baseline := terminal.subscribe(0)
	defer terminal.unsubscribe(subscription)
	if len(baseline.Data) != nativeTerminalHistoryLimit || string(baseline.Data[len(baseline.Data)-4:]) != "tail" {
		t.Fatalf("unexpected reconnect history: len=%d tail=%q", len(baseline.Data), baseline.Data[len(baseline.Data)-4:])
	}
	terminal.publish([]byte("live"))
	select {
	case event := <-subscription.Events:
		if event.Type != "output" || string(event.Data) != "live" {
			t.Fatalf("unexpected live output: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected live terminal output")
	}
}
