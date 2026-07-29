package api

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestNormalizeCodexProviderRequest(t *testing.T) {
	provider, err := normalizeCodexProviderRequest("codex", &codexProviderRequest{
		BaseURL: " https://api.example.com/v1/ ",
		APIKey:  " secret-key ",
		WireAPI: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.BaseURL != "https://api.example.com/v1" || provider.APIKey != "secret-key" || provider.WireAPI != "responses" {
		t.Fatalf("unexpected normalized provider: %#v", provider)
	}
	for _, invalid := range []*codexProviderRequest{
		{BaseURL: "file:///tmp/api", APIKey: "key", WireAPI: "responses"},
		{BaseURL: "https://api.example.com/v1?token=secret", APIKey: "key", WireAPI: "responses"},
		{BaseURL: "https://api.example.com/v1", APIKey: "key", WireAPI: "chat"},
	} {
		if _, err := normalizeCodexProviderRequest("codex", invalid); err == nil {
			t.Fatalf("expected provider to be rejected: %#v", invalid)
		}
	}
	if _, err := normalizeCodexProviderRequest("claude", &codexProviderRequest{BaseURL: "https://api.example.com/v1", APIKey: "key"}); err == nil {
		t.Fatal("expected non-Codex custom provider to be rejected")
	}
}

func TestConfigureCodexCommandUsesEncryptedSessionKey(t *testing.T) {
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })

	session := &model.AIDevSession{}
	if err := setCodexProviderOnSession(session, &codexProviderRequest{
		BaseURL: "https://api.example.com/v1",
		APIKey:  "secret-key",
		WireAPI: "responses",
	}); err != nil {
		t.Fatal(err)
	}
	if session.CodexAPIKey == "secret-key" || strings.Contains(session.CodexAPIKey, "secret-key") {
		t.Fatal("API key was stored without encryption")
	}
	encodedSession, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encodedSession), session.CodexAPIKey) || strings.Contains(string(encodedSession), "secret-key") {
		t.Fatalf("API key leaked through session JSON: %s", encodedSession)
	}
	command := exec.Command("codex", "exec", "inspect")
	command.Env = []string{"PATH=/usr/bin"}
	if err := configureCodexCommand(command, session); err != nil {
		t.Fatal(err)
	}
	joinedArgs := strings.Join(command.Args, " ")
	if strings.Contains(joinedArgs, "secret-key") || !strings.Contains(joinedArgs, `wire_api="responses"`) {
		t.Fatalf("unexpected command arguments: %s", joinedArgs)
	}
	if environmentValue(command.Env, codexSessionAPIKeyEnv) != "secret-key" {
		t.Fatal("API key was not injected through the child process environment")
	}
}
