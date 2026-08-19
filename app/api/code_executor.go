package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeExecutorDefinition struct {
	ID                  string
	Name                string
	Description         string
	Command             string
	VersionArgs         []string
	ConfigPaths         []string
	Capabilities        []string
	ApprovalPolicies    []string
	NativeTerminal      bool
	AutomationSupported bool
	Factory             codeExecutorFactory
}

type codeExecutorStatus struct {
	ID                         string                    `json:"id"`
	Name                       string                    `json:"name"`
	Description                string                    `json:"description"`
	Installed                  bool                      `json:"installed"`
	Available                  bool                      `json:"available"`
	Version                    string                    `json:"version"`
	Configured                 bool                      `json:"configured"`
	CustomProviderConfigurable bool                      `json:"customProviderConfigurable"`
	Reason                     string                    `json:"reason"`
	ReasonCode                 string                    `json:"reasonCode"`
	Capabilities               []string                  `json:"capabilities"`
	ApprovalPolicies           []string                  `json:"approvalPolicies"`
	NativeTerminal             bool                      `json:"nativeTerminal"`
	ConfigSchema               *codeExecutorConfigSchema `json:"configSchema,omitempty"`
}

var codeExecutorDefinitions = []codeExecutorDefinition{
	{ID: "terminal", Name: "Terminal", Description: "在隔离工作区中使用普通终端", Capabilities: []string{"shell"}, ApprovalPolicies: []string{codeApprovalPolicyFullAuto}, AutomationSupported: true},
	{ID: "codex", Name: "Codex", Description: "使用 OpenAI Codex 执行开发任务", Command: "codex", VersionArgs: []string{"--version"}, ConfigPaths: []string{".codex"}, Capabilities: []string{"code", "automation", "interactive", "resume"}, ApprovalPolicies: allCodeApprovalPolicies(), NativeTerminal: true, AutomationSupported: true, Factory: codexExecutorFactory{}},
	{ID: "grok", Name: "Grok Build", Description: "使用 xAI Grok Build 执行开发任务", Command: "grok", VersionArgs: []string{"--version"}, ConfigPaths: []string{".grok/auth.json"}, Capabilities: []string{"code", "automation", "interactive", "resume"}, ApprovalPolicies: allCodeApprovalPolicies(), NativeTerminal: true, AutomationSupported: true, Factory: grokExecutorFactory{}},
	{ID: "claude", Name: "Claude Code", Description: "使用 Claude Code 执行开发任务", Command: "claude", VersionArgs: []string{"--version"}, ConfigPaths: []string{".claude", ".claude.json"}, Capabilities: []string{"code", "automation", "interactive", "resume"}, ApprovalPolicies: allCodeApprovalPolicies(), NativeTerminal: true, AutomationSupported: true, Factory: claudeExecutorFactory{}},
	{ID: "opencode", Name: "OpenCode", Description: "使用 OpenCode 执行开发任务", Command: "opencode", VersionArgs: []string{"--version"}, ConfigPaths: []string{".config/opencode", ".local/share/opencode"}, Capabilities: []string{"code", "automation", "interactive", "resume"}, ApprovalPolicies: []string{codeApprovalPolicyFullAuto}, NativeTerminal: true, AutomationSupported: true, Factory: openCodeExecutorFactory{}},
	{ID: "aider", Name: "Aider", Description: "使用 Aider 执行开发任务", Command: "aider", VersionArgs: []string{"--version"}, ConfigPaths: []string{".aider.conf.yml", ".aider"}, Capabilities: []string{"code", "automation", "interactive", "resume"}, ApprovalPolicies: []string{codeApprovalPolicyFullAuto}, NativeTerminal: true, AutomationSupported: true, Factory: aiderExecutorFactory{}},
}

func getCodeExecutorFactory(executorID string) (codeExecutorFactory, error) {
	definition, err := getCodeExecutorDefinition(executorID)
	if err != nil {
		return nil, err
	}
	if definition.Factory == nil {
		return nil, errors.New("该执行器不支持 AI 指令")
	}
	return definition.Factory, nil
}

func normalizeCodeExecutorID(executorID string) (string, error) {
	executorID = strings.ToLower(strings.TrimSpace(executorID))
	for _, definition := range codeExecutorDefinitions {
		if definition.ID == executorID {
			return definition.ID, nil
		}
	}
	return "", errors.New("不支持的 Code 执行器")
}

func getCodeExecutorDefinition(executorID string) (codeExecutorDefinition, error) {
	normalizedID, err := normalizeCodeExecutorID(executorID)
	if err != nil {
		return codeExecutorDefinition{}, err
	}
	for _, definition := range codeExecutorDefinitions {
		if definition.ID == normalizedID {
			return definition, nil
		}
	}
	return codeExecutorDefinition{}, errors.New("不支持的 Code 执行器")
}

func detectCodeExecutor(definition codeExecutorDefinition) codeExecutorStatus {
	status := codeExecutorStatus{
		ID:                         definition.ID,
		Name:                       definition.Name,
		Description:                definition.Description,
		Capabilities:               definition.Capabilities,
		ApprovalPolicies:           definition.ApprovalPolicies,
		NativeTerminal:             definition.NativeTerminal && nativeTerminalPlatformSupported(),
		CustomProviderConfigurable: supportsCustomCodeProvider(definition.ID),
	}
	if definition.Factory != nil {
		status.ConfigSchema = definition.Factory.ConfigSchema()
	}
	if definition.ID == "terminal" {
		status.Installed = true
		status.Available = true
		status.Configured = true
		status.Version = "built-in"
		return status
	}
	commandPath, commandEnv, err := resolveCodeExecutorCommand(definition.Command)
	if err != nil {
		status.Reason = "未找到 " + definition.Command + " 命令"
		status.ReasonCode = "not_installed"
		return status
	}
	status.Installed = true
	status.Configured = hasCodeExecutorConfig(definition.ConfigPaths)
	status.Version = detectCodeExecutorVersion(commandPath, definition.VersionArgs, commandEnv)
	if definition.ID == "claude" {
		status.Configured = detectClaudeAuthenticated(commandPath, commandEnv)
	}
	if definition.ID == "opencode" {
		status.Configured = detectOpenCodeConfigured()
	}
	if !definition.AutomationSupported {
		status.Reason = "当前 CLI 不支持非交互自动执行"
		status.ReasonCode = "automation_unsupported"
		return status
	}
	status.Available = true
	return status
}

func detectClaudeAuthenticated(commandPath string, commandEnv []string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, commandPath, "auth", "status", "--json")
	command.Env = commandEnv
	output, err := command.Output()
	if err != nil {
		return false
	}
	var status struct {
		LoggedIn bool `json:"loggedIn"`
	}
	return json.Unmarshal(bytes.TrimSpace(output), &status) == nil && status.LoggedIn
}

func detectOpenCodeConfigured() bool {
	if nativeOpenCodeAuthFileHasCredentials() || nativeOpenCodeDatabaseHasCredentials() {
		return true
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, name := range []string{"opencode.json", "opencode.jsonc"} {
		content, readErr := os.ReadFile(filepath.Join(homeDir, ".config", "opencode", name))
		if readErr == nil && openCodeConfigHasProvider(content) {
			return true
		}
	}
	return false
}

func openCodeConfigHasProvider(content []byte) bool {
	var config struct {
		Provider map[string]json.RawMessage `json:"provider"`
	}
	return json.Unmarshal(content, &config) == nil && len(config.Provider) > 0
}

func validateCodeExecutorConfigured(executorID string, provider *codeProviderRequest) error {
	if provider != nil {
		return nil
	}
	definition, err := getCodeExecutorDefinition(executorID)
	if err != nil {
		return err
	}
	if definition.ID != "opencode" {
		return nil
	}
	status := detectCodeExecutor(definition)
	if status.Configured {
		return nil
	}
	return errors.New(status.Name + " 尚未登录或配置模型连接")
}

func hasCodeExecutorConfig(configPaths []string) bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, configPath := range configPaths {
		if _, statErr := os.Stat(filepath.Join(homeDir, configPath)); statErr == nil {
			return true
		}
	}
	return false
}

func detectCodeExecutorVersion(commandPath string, versionArgs, commandEnv []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, commandPath, versionArgs...)
	command.Env = commandEnv
	output, err := command.CombinedOutput()
	if err != nil && len(output) == 0 {
		return ""
	}
	version := strings.TrimSpace(string(output))
	if lineEnd := strings.IndexByte(version, '\n'); lineEnd >= 0 {
		version = version[:lineEnd]
	}
	if len([]rune(version)) > 120 {
		version = string([]rune(version)[:120])
	}
	return version
}

func validateCodeExecutorAvailable(executorID, role string) (string, error) {
	definition, err := getCodeExecutorDefinition(executorID)
	if err != nil {
		return "", err
	}
	if role == constant.UserRoleSubAdmin && definition.ID != "terminal" {
		return "", errors.New("子管理员只能使用隔离终端执行器")
	}
	status := detectCodeExecutor(definition)
	if !status.Available {
		if status.Reason == "" {
			status.Reason = "执行器当前不可用"
			status.ReasonCode = "unavailable"
		}
		return "", errors.New(status.Name + " 不可用：" + status.Reason)
	}
	return definition.ID, nil
}

func buildCodeExecutorArgs(executorID, prompt, nativeSessionID string, sessionID uint, approvalPolicy string) ([]string, string, error) {
	factory, err := getCodeExecutorFactory(executorID)
	if err != nil {
		return nil, "", err
	}
	return factory.BuildArgs(prompt, nativeSessionID, sessionID, approvalPolicy)
}

func ensureAiderHistoryDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	historyDir := filepath.Join(homeDir, ".gopanel", "code", "aider")
	if err := os.MkdirAll(historyDir, 0700); err != nil {
		return "", err
	}
	return historyDir, nil
}

func buildCodeExecutorCommand(ctx context.Context, executorID, workDir, prompt, nativeSessionID string, sessionID uint, session *model.AIDevSession) (*exec.Cmd, string, error) {
	definition, err := getCodeExecutorDefinition(executorID)
	if err != nil {
		return nil, "", err
	}
	if !definition.AutomationSupported || definition.Command == "" {
		return nil, "", errors.New("该执行器不支持 AI 指令")
	}
	commandPath, commandEnv, err := resolveCodeExecutorCommand(definition.Command)
	if err != nil {
		return nil, "", err
	}
	approvalPolicy := ""
	if session != nil {
		approvalPolicy = session.ApprovalPolicy
	}
	args, preparedSessionID, err := buildCodeExecutorArgs(definition.ID, prompt, nativeSessionID, sessionID, approvalPolicy)
	if err != nil {
		return nil, "", err
	}
	if definition.ID == "codex" {
		writableDirs, writableErr := codexWritableDirsForSessionWithRepair(session)
		if writableErr != nil {
			return nil, "", writableErr
		}
		args = addCodexWritableDirArgs(args, writableDirs)
	}
	command := exec.CommandContext(ctx, commandPath, args...)
	command.Dir = workDir
	command.Env = commandEnv
	if err := configureCodeProviderCommand(definition.ID, command, session); err != nil {
		return nil, "", err
	}
	return command, preparedSessionID, nil
}

func GetCodeExecutors(c fiber.Ctx) error {
	claims, _ := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	statuses := loadCodeExecutorStatuses()
	for index := range statuses {
		status := &statuses[index]
		definition := codeExecutorDefinitions[index]
		if claims != nil && claims.Role == constant.UserRoleSubAdmin && definition.ID != "terminal" {
			status.Available = false
			status.Reason = "子管理员只能使用隔离终端执行器"
			status.ReasonCode = "role_restricted"
		}
	}
	return c.JSON(e.Succ(statuses))
}
