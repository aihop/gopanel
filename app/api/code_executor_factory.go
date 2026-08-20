package api

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/google/uuid"
)

const (
	codexSessionAPIKeyEnv   = "GOPANEL_CODEX_SESSION_API_KEY"
	codexNetworkConfig      = "sandbox_workspace_write.network_access=true"
	codexDisableDocsMCP     = "mcp_servers.openaiDeveloperDocs.enabled=false"
	openCodeSessionKeyEnv   = "GOPANEL_OPENCODE_SESSION_API_KEY"
	claudeSessionAPIKeyEnv  = "ANTHROPIC_API_KEY"
	claudeSessionBaseURLEnv = "ANTHROPIC_BASE_URL"
	aiderSessionAPIKeyEnv   = "OPENAI_API_KEY"
	aiderSessionBaseURLEnv  = "OPENAI_API_BASE"
)

type codeExecutorConfigRequest struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	Model   string `json:"model"`
}

type codeExecutorConfigField struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	Required    bool   `json:"required"`
}

type codeExecutorConfigSchema struct {
	Fields []codeExecutorConfigField `json:"fields"`
}

type codeExecutorFactory interface {
	BuildArgs(prompt, nativeSessionID string, sessionID uint, approvalPolicy string) ([]string, string, error)
	ConfigSchema() *codeExecutorConfigSchema
	ConfigureCommand(command *exec.Cmd, session *model.AIDevSession) error
}

type codexExecutorFactory struct{}
type grokExecutorFactory struct{}
type claudeExecutorFactory struct{}
type openCodeExecutorFactory struct{}
type aiderExecutorFactory struct{}

func providerConfigSchema(modelRequired bool) *codeExecutorConfigSchema {
	return &codeExecutorConfigSchema{Fields: []codeExecutorConfigField{
		{Key: "baseUrl", Type: "url", Label: "Base URL", Placeholder: "https://api.example.com/v1", Required: true},
		{Key: "apiKey", Type: "password", Label: "API Key", Placeholder: "API Key", Required: true},
		{Key: "model", Type: "text", Label: "Model", Placeholder: "Model ID", Required: modelRequired},
	}}
}

func (codexExecutorFactory) ConfigSchema() *codeExecutorConfigSchema {
	return providerConfigSchema(false)
}
func (grokExecutorFactory) ConfigSchema() *codeExecutorConfigSchema {
	return nil
}
func (claudeExecutorFactory) ConfigSchema() *codeExecutorConfigSchema {
	return providerConfigSchema(false)
}
func (openCodeExecutorFactory) ConfigSchema() *codeExecutorConfigSchema {
	return providerConfigSchema(true)
}
func (aiderExecutorFactory) ConfigSchema() *codeExecutorConfigSchema {
	return providerConfigSchema(false)
}

func (codexExecutorFactory) BuildArgs(prompt, nativeSessionID string, _ uint, approvalPolicy string) ([]string, string, error) {
	prefix := append(codexSandboxArgs(approvalPolicy), "exec")
	if nativeSessionID != "" {
		return append(prefix, "resume", "--json", "--skip-git-repo-check", nativeSessionID, prompt), nativeSessionID, nil
	}
	return append(prefix, "--json", "--skip-git-repo-check", prompt), "", nil
}

func codexSandboxArgs(approvalPolicy string) []string {
	return []string{
		"-c", codexNetworkConfig,
		"-c", codexDisableDocsMCP,
		"--ask-for-approval", codexApprovalPolicy(approvalPolicy),
		"--sandbox", "workspace-write",
	}
}

func (grokExecutorFactory) BuildArgs(prompt, nativeSessionID string, _ uint, approvalPolicy string) ([]string, string, error) {
	args := append([]string{"--no-auto-update"}, grokApprovalArgs(approvalPolicy)...)
	args = append(args, "--output-format", "streaming-json")
	if nativeSessionID != "" {
		return append(args, "--resume", nativeSessionID, "-p", prompt), nativeSessionID, nil
	}
	nativeSessionID = uuid.NewString()
	return append(args, "--session-id", nativeSessionID, "-p", prompt), nativeSessionID, nil
}

func grokApprovalArgs(approvalPolicy string) []string {
	switch approvalPolicy {
	case codeApprovalPolicyFullAuto:
		return []string{"--always-approve"}
	case codeApprovalPolicySafeAuto:
		return []string{"--permission-mode", "auto"}
	default:
		return []string{"--permission-mode", "default"}
	}
}

func (claudeExecutorFactory) BuildArgs(prompt, nativeSessionID string, _ uint, approvalPolicy string) ([]string, string, error) {
	prefix := []string{"--print", "--output-format", "stream-json", "--include-partial-messages"}
	prefix = append(prefix, claudeApprovalArgs(approvalPolicy)...)
	if nativeSessionID != "" {
		return append(prefix, "--resume", nativeSessionID, prompt), nativeSessionID, nil
	}
	nativeSessionID = uuid.NewString()
	return append(prefix, "--session-id", nativeSessionID, prompt), nativeSessionID, nil
}

func (openCodeExecutorFactory) BuildArgs(prompt, nativeSessionID string, _ uint, _ string) ([]string, string, error) {
	args := []string{"run", "--format", "json", "--dangerously-skip-permissions"}
	if nativeSessionID != "" {
		args = append(args, "--session", nativeSessionID)
	}
	return append(args, prompt), nativeSessionID, nil
}

func claudeApprovalArgs(approvalPolicy string) []string {
	switch approvalPolicy {
	case codeApprovalPolicyFullAuto:
		return []string{"--dangerously-skip-permissions"}
	case codeApprovalPolicyManual:
		return []string{"--permission-mode", "manual"}
	default:
		return []string{"--permission-mode", "acceptEdits"}
	}
}

func (aiderExecutorFactory) BuildArgs(prompt, nativeSessionID string, sessionID uint, _ string) ([]string, string, error) {
	if nativeSessionID == "" {
		nativeSessionID = "gopanel-" + strconv.FormatUint(uint64(sessionID), 10)
	}
	historyDir, err := ensureAiderHistoryDir()
	if err != nil {
		return nil, "", err
	}
	chatHistory := filepath.Join(historyDir, nativeSessionID+".chat.md")
	llmHistory := filepath.Join(historyDir, nativeSessionID+".llm.log")
	args := []string{"--yes-always", "--chat-history-file", chatHistory, "--llm-history-file", llmHistory}
	if _, statErr := os.Stat(chatHistory); statErr == nil {
		args = append(args, "--restore-chat-history")
	}
	return append(args, "--message", prompt), nativeSessionID, nil
}

func normalizeCodeExecutorConfig(factory codeExecutorFactory, config *codeExecutorConfigRequest) (*codeExecutorConfigRequest, error) {
	if config == nil {
		return nil, nil
	}
	if factory == nil || factory.ConfigSchema() == nil {
		return nil, errors.New("该开发终端不支持自定义模型连接")
	}
	normalized := &codeExecutorConfigRequest{
		BaseURL: strings.TrimRight(strings.TrimSpace(config.BaseURL), "/"),
		APIKey:  strings.TrimSpace(config.APIKey),
		Model:   strings.TrimSpace(config.Model),
	}
	if normalized.BaseURL == "" && normalized.APIKey == "" && normalized.Model == "" {
		return nil, nil
	}
	if normalized.BaseURL == "" || normalized.APIKey == "" {
		return nil, errors.New("自定义模型连接必须填写 Base URL 和 API Key")
	}
	parsedURL, err := url.Parse(normalized.BaseURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.User != nil {
		return nil, errors.New("Base URL 必须是有效的 HTTP 或 HTTPS 地址")
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, errors.New("Base URL 不能包含查询参数或片段")
	}
	for _, field := range factory.ConfigSchema().Fields {
		if field.Key == "model" && field.Required && normalized.Model == "" {
			return nil, errors.New("该开发终端的自定义连接必须填写模型 ID")
		}
	}
	return normalized, nil
}

func setCodeExecutorConfigOnSession(session *model.AIDevSession, config *codeExecutorConfigRequest) error {
	if config == nil {
		return nil
	}
	ciphertext, err := encrypt.StringEncrypt(config.APIKey)
	if err != nil {
		return err
	}
	session.ProviderBaseURL = config.BaseURL
	session.ProviderModel = config.Model
	session.ProviderAPIKey = ciphertext
	return nil
}

func getSessionExecutorConfig(session *model.AIDevSession) (*codeExecutorConfigRequest, error) {
	if session == nil || strings.TrimSpace(session.ProviderBaseURL) == "" {
		return nil, nil
	}
	apiKey, err := encrypt.StringDecrypt(session.ProviderAPIKey)
	if err != nil {
		return nil, errors.New("无法解密此会话的开发终端 API Key")
	}
	return &codeExecutorConfigRequest{BaseURL: session.ProviderBaseURL, APIKey: apiKey, Model: session.ProviderModel}, nil
}

func prependCommandArgs(command *exec.Cmd, args ...string) {
	command.Args = append([]string{command.Path}, append(args, command.Args[1:]...)...)
}

func configureProviderEnvironment(command *exec.Cmd, values map[string]string) {
	commandEnv := command.Env
	if len(commandEnv) == 0 {
		commandEnv = os.Environ()
	}
	for key, value := range values {
		commandEnv = upsertEnvironment(commandEnv, key, value)
	}
	command.Env = commandEnv
}

func (codexExecutorFactory) ConfigureCommand(command *exec.Cmd, session *model.AIDevSession) error {
	config, err := getSessionExecutorConfig(session)
	if err != nil || config == nil {
		return err
	}
	providerArgs := []string{
		"-c", `model_provider="gopanel_session"`,
		"-c", `model_providers.gopanel_session.name="GoPanel Session"`,
		"-c", "model_providers.gopanel_session.base_url=" + strconv.Quote(config.BaseURL),
		"-c", "model_providers.gopanel_session.env_key=" + strconv.Quote(codexSessionAPIKeyEnv),
		"-c", `model_providers.gopanel_session.wire_api="responses"`,
		"-c", `model_providers.gopanel_session.requires_openai_auth=false`,
	}
	if config.Model != "" {
		providerArgs = append(providerArgs, "--model", config.Model)
	}
	prependCommandArgs(command, providerArgs...)
	configureProviderEnvironment(command, map[string]string{codexSessionAPIKeyEnv: config.APIKey})
	return nil
}

func (grokExecutorFactory) ConfigureCommand(_ *exec.Cmd, _ *model.AIDevSession) error {
	return nil
}

func (claudeExecutorFactory) ConfigureCommand(command *exec.Cmd, session *model.AIDevSession) error {
	config, err := getSessionExecutorConfig(session)
	if err != nil || config == nil {
		return err
	}
	if config.Model != "" {
		prependCommandArgs(command, "--model", config.Model)
	}
	configureProviderEnvironment(command, map[string]string{
		claudeSessionAPIKeyEnv:  config.APIKey,
		claudeSessionBaseURLEnv: config.BaseURL,
	})
	return nil
}

func (openCodeExecutorFactory) ConfigureCommand(command *exec.Cmd, session *model.AIDevSession) error {
	config, err := getSessionExecutorConfig(session)
	if err != nil || config == nil {
		return err
	}
	providerConfig := map[string]any{
		"model": "gopanel_session/" + config.Model,
		"provider": map[string]any{
			"gopanel_session": map[string]any{
				"npm": "@ai-sdk/openai-compatible", "name": "GoPanel Session",
				"options": map[string]string{"baseURL": config.BaseURL, "apiKey": "{env:" + openCodeSessionKeyEnv + "}"},
				"models":  map[string]any{config.Model: map[string]string{"name": config.Model}},
			},
		},
	}
	configJSON, err := json.Marshal(providerConfig)
	if err != nil {
		return err
	}
	configureProviderEnvironment(command, map[string]string{
		openCodeSessionKeyEnv: config.APIKey, "OPENCODE_CONFIG_CONTENT": string(configJSON),
	})
	return nil
}

func (aiderExecutorFactory) ConfigureCommand(command *exec.Cmd, session *model.AIDevSession) error {
	config, err := getSessionExecutorConfig(session)
	if err != nil || config == nil {
		return err
	}
	if config.Model != "" {
		prependCommandArgs(command, "--model", config.Model)
	}
	configureProviderEnvironment(command, map[string]string{
		aiderSessionAPIKeyEnv: config.APIKey, aiderSessionBaseURLEnv: config.BaseURL,
	})
	return nil
}

type codexProviderRequest struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	WireAPI string `json:"wireApi"`
}

func normalizeCodexProviderRequest(executorID string, provider *codexProviderRequest) (*codexProviderRequest, error) {
	if provider == nil {
		return nil, nil
	}
	if executorID != "codex" {
		return nil, errors.New("自定义 Codex 连接仅支持 Codex 执行器")
	}
	config, err := normalizeCodeExecutorConfig(codexExecutorFactory{}, &codeExecutorConfigRequest{BaseURL: provider.BaseURL, APIKey: provider.APIKey})
	if err != nil {
		return nil, err
	}
	wireAPI := strings.ToLower(strings.TrimSpace(provider.WireAPI))
	if wireAPI == "" {
		wireAPI = "responses"
	}
	if wireAPI != "responses" {
		return nil, errors.New("当前 Codex CLI 仅支持 Responses API 协议")
	}
	return &codexProviderRequest{BaseURL: config.BaseURL, APIKey: config.APIKey, WireAPI: wireAPI}, nil
}

func setCodexProviderOnSession(session *model.AIDevSession, provider *codexProviderRequest) error {
	if provider == nil {
		return nil
	}
	return setCodeExecutorConfigOnSession(session, &codeExecutorConfigRequest{BaseURL: provider.BaseURL, APIKey: provider.APIKey})
}

func configureCodexCommand(command *exec.Cmd, session *model.AIDevSession) error {
	return (codexExecutorFactory{}).ConfigureCommand(command, session)
}

type codeProviderRequest = codeExecutorConfigRequest

func supportsCustomCodeProvider(executorID string) bool {
	factory, err := getCodeExecutorFactory(executorID)
	return err == nil && factory.ConfigSchema() != nil
}

func normalizeCodeProviderRequest(executorID string, provider *codeProviderRequest) (*codeProviderRequest, error) {
	if provider == nil {
		return nil, nil
	}
	factory, err := getCodeExecutorFactory(executorID)
	if err != nil {
		return nil, errors.New("该开发终端不支持自定义模型连接")
	}
	return normalizeCodeExecutorConfig(factory, provider)
}

func setCodeProviderOnSession(session *model.AIDevSession, provider *codeProviderRequest) error {
	return setCodeExecutorConfigOnSession(session, provider)
}

func configureCodeProviderCommand(executorID string, command *exec.Cmd, session *model.AIDevSession) error {
	factory, err := getCodeExecutorFactory(executorID)
	if err != nil {
		if session == nil || strings.TrimSpace(session.ProviderBaseURL) == "" {
			return nil
		}
		return err
	}
	return factory.ConfigureCommand(command, session)
}
