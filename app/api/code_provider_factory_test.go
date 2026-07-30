package api

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestNormalizeCodeProviderRequest(t *testing.T) {
	provider, err := normalizeCodeProviderRequest("codex", &codeProviderRequest{
		BaseURL: " https://api.example.com/v1/ ",
		APIKey:  " secret-key ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.BaseURL != "https://api.example.com/v1" || provider.APIKey != "secret-key" {
		t.Fatalf("unexpected normalized provider: %#v", provider)
	}
	provider, err = normalizeCodeProviderRequest("codex", &codeProviderRequest{})
	if err != nil || provider != nil {
		t.Fatalf("empty provider should use defaults: %#v, %v", provider, err)
	}
	for _, invalid := range []*codeProviderRequest{
		{BaseURL: "file:///tmp/api", APIKey: "key"},
		{BaseURL: "https://api.example.com/v1?token=secret", APIKey: "key"},
		{BaseURL: "https://api.example.com/v1"},
		{APIKey: "key"},
	} {
		if _, err := normalizeCodeProviderRequest("codex", invalid); err == nil {
			t.Fatalf("expected provider to be rejected: %#v", invalid)
		}
	}
	if _, err := normalizeCodeProviderRequest("terminal", &codeProviderRequest{BaseURL: "https://api.example.com/v1", APIKey: "key"}); err == nil {
		t.Fatal("expected unsupported terminal provider to be rejected")
	}
}

func TestCodeExecutorFactoryConfigSchemas(t *testing.T) {
	for _, executorID := range []string{"codex", "claude", "opencode", "aider"} {
		factory, err := getCodeExecutorFactory(executorID)
		if err != nil || factory.ConfigSchema() == nil || len(factory.ConfigSchema().Fields) != 3 {
			t.Fatalf("%s config schema is unavailable: %#v, %v", executorID, factory, err)
		}
	}
	if _, err := normalizeCodeProviderRequest("opencode", &codeProviderRequest{
		BaseURL: "https://api.example.com/v1", APIKey: "key",
	}); err == nil {
		t.Fatal("expected OpenCode custom provider to require a model")
	}
	provider, err := normalizeCodeProviderRequest("opencode", &codeProviderRequest{
		BaseURL: "https://api.example.com/v1", APIKey: "key", Model: "test-model",
	})
	if err != nil || provider.Model != "test-model" {
		t.Fatalf("unexpected OpenCode provider: %#v, %v", provider, err)
	}
}

func TestCodeProviderFactoryMappings(t *testing.T) {
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })

	session := &model.AIDevSession{}
	if err := setCodeProviderOnSession(session, &codeProviderRequest{
		BaseURL: "https://api.example.com/v1",
		APIKey:  "secret-key",
	}); err != nil {
		t.Fatal(err)
	}
	if session.ProviderAPIKey == "secret-key" || strings.Contains(session.ProviderAPIKey, "secret-key") {
		t.Fatal("API key was stored without encryption")
	}

	tests := []struct {
		executorID  string
		model       string
		argsNeedle  string
		environment map[string]string
	}{
		{executorID: "codex", argsNeedle: `model_provider="gopanel_session"`, environment: map[string]string{codexSessionAPIKeyEnv: "secret-key"}},
		{executorID: "claude", environment: map[string]string{"ANTHROPIC_BASE_URL": "https://api.example.com/v1", "ANTHROPIC_API_KEY": "secret-key"}},
		{executorID: "opencode", model: "test-model", environment: map[string]string{openCodeSessionKeyEnv: "secret-key"}},
		{executorID: "aider", environment: map[string]string{"OPENAI_API_BASE": "https://api.example.com/v1", "OPENAI_API_KEY": "secret-key"}},
	}
	for _, test := range tests {
		t.Run(test.executorID, func(t *testing.T) {
			session.ProviderModel = test.model
			command := exec.Command(test.executorID, "run")
			command.Env = []string{"PATH=/usr/bin"}
			if err := configureCodeProviderCommand(test.executorID, command, session); err != nil {
				t.Fatal(err)
			}
			joinedArgs := strings.Join(command.Args, " ")
			if strings.Contains(joinedArgs, "secret-key") || (test.argsNeedle != "" && !strings.Contains(joinedArgs, test.argsNeedle)) {
				t.Fatalf("unexpected command arguments: %s", joinedArgs)
			}
			for key, expected := range test.environment {
				if got := environmentValue(command.Env, key); got != expected {
					t.Fatalf("%s = %q, want %q", key, got, expected)
				}
			}
			if test.executorID == "opencode" {
				config := environmentValue(command.Env, "OPENCODE_CONFIG_CONTENT")
				if !strings.Contains(config, "test-model") || strings.Contains(config, "secret-key") {
					t.Fatalf("unexpected OpenCode config: %s", config)
				}
			}
		})
	}
}

func TestCodeProviderFactoryLeavesDefaultEnvironmentUntouched(t *testing.T) {
	command := exec.Command("codex", "exec")
	command.Env = []string{"PATH=/usr/bin"}
	if err := configureCodeProviderCommand("codex", command, &model.AIDevSession{}); err != nil {
		t.Fatal(err)
	}
	if len(command.Env) != 1 || command.Env[0] != "PATH=/usr/bin" {
		t.Fatalf("default environment changed: %#v", command.Env)
	}
}
