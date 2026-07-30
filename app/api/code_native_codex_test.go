package api

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestConfigureNativeTerminalEnvironmentPreservesSessionProvider(t *testing.T) {
	command := exec.Command("codex")
	command.Env = []string{
		"PATH=/managed/bin",
		codexSessionAPIKeyEnv + "=session-secret",
		"TERM=dumb",
	}
	configureNativeTerminalEnvironment(command)
	if got := environmentValue(command.Env, codexSessionAPIKeyEnv); got != "session-secret" {
		t.Fatalf("session API key was lost: %q", got)
	}
	if got := environmentValue(command.Env, "PATH"); got != "/managed/bin" {
		t.Fatalf("managed PATH was replaced: %q", got)
	}
	if got := environmentValue(command.Env, "TERM"); got != "xterm-256color" {
		t.Fatalf("TERM = %q", got)
	}
	if got := environmentValue(command.Env, "COLORTERM"); got != "truecolor" {
		t.Fatalf("COLORTERM = %q", got)
	}
}

func TestNativeCodexCustomProviderReachesChildProcess(t *testing.T) {
	binDir := t.TempDir()
	commandPath := filepath.Join(binDir, "codex")
	script := "#!/bin/sh\n" +
		"printf 'KEY=%s\\n' \"$GOPANEL_CODEX_SESSION_API_KEY\"\n" +
		"printf 'ARGS=%s\\n' \"$*\"\n"
	if err := os.WriteFile(commandPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })
	session := &model.AIDevSession{WorkDir: t.TempDir(), ApprovalPolicy: codeApprovalPolicySafeAuto}
	if err := setCodeProviderOnSession(session, &codeProviderRequest{
		BaseURL: "https://gateway.example.com/v1", APIKey: "session-secret", Model: "gpt-test",
	}); err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "provider-session.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIDevSession{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	global.DB = database
	var persistedSession model.AIDevSession
	if err := database.First(&persistedSession, session.ID).Error; err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name            string
		nativeSessionID string
		extraExpected   string
	}{
		{name: "new session"},
		{name: "resumed session", nativeSessionID: "native-session", extraExpected: "resume native-session"},
	} {
		t.Run(test.name, func(t *testing.T) {
			persistedSession.NativeSessionID = test.nativeSessionID
			command, err := buildNativeCodexCommand(&persistedSession)
			if err != nil {
				t.Fatal(err)
			}
			configureNativeTerminalEnvironment(command)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("run fake Codex: %v: %s", err, output)
			}
			result := string(output)
			for _, expected := range []string{
				"KEY=session-secret",
				`model_provider="gopanel_session"`,
				`model_providers.gopanel_session.base_url="https://gateway.example.com/v1"`,
				`model_providers.gopanel_session.env_key="GOPANEL_CODEX_SESSION_API_KEY"`,
				"--model gpt-test",
				test.extraExpected,
			} {
				if expected != "" && !strings.Contains(result, expected) {
					t.Fatalf("child process missing %q: %s", expected, result)
				}
			}
		})
	}
}

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
