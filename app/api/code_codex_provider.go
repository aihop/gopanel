package api

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/utils/encrypt"
)

const codexSessionAPIKeyEnv = "GOPANEL_CODEX_SESSION_API_KEY"

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
		return nil, errors.New("自定义模型连接仅支持 Codex 执行器")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	apiKey := strings.TrimSpace(provider.APIKey)
	wireAPI := strings.ToLower(strings.TrimSpace(provider.WireAPI))
	if baseURL == "" || apiKey == "" {
		return nil, errors.New("自定义 Codex 连接必须填写 Base URL 和 API Key")
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.User != nil {
		return nil, errors.New("Base URL 必须是有效的 HTTP 或 HTTPS 地址")
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, errors.New("Base URL 不能包含查询参数或片段")
	}
	if wireAPI == "" {
		wireAPI = "responses"
	}
	if wireAPI != "responses" {
		return nil, errors.New("当前 Codex CLI 仅支持 Responses API 协议")
	}
	return &codexProviderRequest{BaseURL: baseURL, APIKey: apiKey, WireAPI: wireAPI}, nil
}

func setCodexProviderOnSession(session *model.AIDevSession, provider *codexProviderRequest) error {
	if provider == nil {
		return nil
	}
	ciphertext, err := encrypt.StringEncrypt(provider.APIKey)
	if err != nil {
		return err
	}
	session.CodexBaseURL = provider.BaseURL
	session.CodexWireAPI = provider.WireAPI
	session.CodexAPIKey = ciphertext
	return nil
}

func configureCodexCommand(command *exec.Cmd, session *model.AIDevSession) error {
	if session == nil || strings.TrimSpace(session.CodexBaseURL) == "" {
		return nil
	}
	apiKey, err := encrypt.StringDecrypt(session.CodexAPIKey)
	if err != nil {
		return errors.New("无法解密此会话的 Codex API Key")
	}
	providerArgs := []string{
		"-c", `model_provider="gopanel_session"`,
		"-c", `model_providers.gopanel_session.name="GoPanel Session"`,
		"-c", "model_providers.gopanel_session.base_url=" + strconv.Quote(session.CodexBaseURL),
		"-c", "model_providers.gopanel_session.env_key=" + strconv.Quote(codexSessionAPIKeyEnv),
		"-c", "model_providers.gopanel_session.wire_api=" + strconv.Quote(session.CodexWireAPI),
	}
	command.Args = append([]string{command.Path}, append(providerArgs, command.Args[1:]...)...)
	command.Env = append(os.Environ(), codexSessionAPIKeyEnv+"="+apiKey)
	return nil
}
