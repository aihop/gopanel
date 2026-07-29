package api

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type codeExecutorDefinition struct {
	ID                  string
	Name                string
	Description         string
	Command             string
	VersionArgs         []string
	ConfigPaths         []string
	Capabilities        []string
	AutomationSupported bool
}

type codeExecutorStatus struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Installed    bool     `json:"installed"`
	Available    bool     `json:"available"`
	Version      string   `json:"version"`
	Configured   bool     `json:"configured"`
	Reason       string   `json:"reason"`
	ReasonCode   string   `json:"reasonCode"`
	Capabilities []string `json:"capabilities"`
}

var codeExecutorDefinitions = []codeExecutorDefinition{
	{ID: "terminal", Name: "Terminal", Description: "在隔离工作区中使用普通终端", Capabilities: []string{"shell"}, AutomationSupported: true},
	{ID: "codex", Name: "Codex", Description: "使用 OpenAI Codex 执行开发任务", Command: "codex", VersionArgs: []string{"--version"}, ConfigPaths: []string{".codex"}, Capabilities: []string{"code", "automation"}, AutomationSupported: true},
	{ID: "claude", Name: "Claude Code", Description: "使用 Claude Code 执行开发任务", Command: "claude", VersionArgs: []string{"--version"}, ConfigPaths: []string{".claude", ".claude.json"}, Capabilities: []string{"code", "automation"}, AutomationSupported: true},
	{ID: "opencode", Name: "OpenCode", Description: "使用 OpenCode 执行开发任务", Command: "opencode", VersionArgs: []string{"--version"}, ConfigPaths: []string{".config/opencode"}, Capabilities: []string{"code", "automation"}, AutomationSupported: true},
	{ID: "aider", Name: "Aider", Description: "使用 Aider 执行开发任务", Command: "aider", VersionArgs: []string{"--version"}, ConfigPaths: []string{".aider.conf.yml"}, Capabilities: []string{"code", "automation"}, AutomationSupported: true},
	{ID: "trae", Name: "Trae", Description: "当前 Trae CLI 仅用于启动编辑器", Command: "trae", VersionArgs: []string{"--version"}, ConfigPaths: []string{".trae"}, Capabilities: []string{"editor"}, AutomationSupported: false},
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
	status := codeExecutorStatus{ID: definition.ID, Name: definition.Name, Description: definition.Description, Capabilities: definition.Capabilities}
	if definition.ID == "terminal" {
		status.Installed = true
		status.Available = true
		status.Configured = true
		status.Version = "built-in"
		return status
	}
	commandPath, err := exec.LookPath(definition.Command)
	if err != nil {
		status.Reason = "未找到 " + definition.Command + " 命令"
		status.ReasonCode = "not_installed"
		return status
	}
	status.Installed = true
	status.Configured = hasCodeExecutorConfig(definition.ConfigPaths)
	status.Version = detectCodeExecutorVersion(commandPath, definition.VersionArgs)
	if !definition.AutomationSupported {
		status.Reason = "当前 CLI 不支持非交互自动执行"
		status.ReasonCode = "automation_unsupported"
		return status
	}
	status.Available = true
	return status
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

func detectCodeExecutorVersion(commandPath string, versionArgs []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, commandPath, versionArgs...).CombinedOutput()
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
	switch executorID {
	case "codex":
		prefix := []string{"--ask-for-approval", codexApprovalPolicy(approvalPolicy), "--sandbox", "workspace-write", "exec"}
		if nativeSessionID != "" {
			return append(prefix, "resume", "--json", "--skip-git-repo-check", nativeSessionID, prompt), nativeSessionID, nil
		}
		return append(prefix, "--json", "--skip-git-repo-check", prompt), "", nil
	case "claude":
		prefix := []string{"--print", "--permission-mode", "acceptEdits", "--output-format", "json"}
		if nativeSessionID != "" {
			return append(prefix, "--resume", nativeSessionID, prompt), nativeSessionID, nil
		}
		nativeSessionID = uuid.NewString()
		return append(prefix, "--session-id", nativeSessionID, prompt), nativeSessionID, nil
	case "opencode":
		args := []string{"run", "--format", "json"}
		if nativeSessionID != "" {
			args = append(args, "--session", nativeSessionID)
		}
		return append(args, prompt), nativeSessionID, nil
	case "aider":
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
	default:
		return nil, "", errors.New("该执行器不支持 AI 指令")
	}
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
	commandPath, err := exec.LookPath(definition.Command)
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
	command := exec.CommandContext(ctx, commandPath, args...)
	command.Dir = workDir
	if definition.ID == "codex" {
		if err := configureCodexCommand(command, session); err != nil {
			return nil, "", err
		}
	}
	return command, preparedSessionID, nil
}

func GetCodeExecutors(c fiber.Ctx) error {
	claims, _ := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	statuses := make([]codeExecutorStatus, 0, len(codeExecutorDefinitions))
	for _, definition := range codeExecutorDefinitions {
		status := detectCodeExecutor(definition)
		if claims != nil && claims.Role == constant.UserRoleSubAdmin && definition.ID != "terminal" {
			status.Available = false
			status.Reason = "子管理员只能使用隔离终端执行器"
			status.ReasonCode = "role_restricted"
		}
		statuses = append(statuses, status)
	}
	return c.JSON(e.Succ(statuses))
}
