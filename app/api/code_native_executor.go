package api

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/google/uuid"
)

func buildNativeCodeCommand(session *model.AIDevSession) (*exec.Cmd, string, error) {
	if session == nil {
		return nil, "", errors.New("开发会话不存在")
	}
	if session.AgentName == "codex" {
		command, err := buildNativeCodexCommand(session)
		return command, strings.TrimSpace(session.NativeSessionID), err
	}
	definition, err := getCodeExecutorDefinition(session.AgentName)
	if err != nil || !definition.NativeTerminal || definition.Factory == nil {
		return nil, "", errors.New("该执行器不支持原生终端")
	}
	commandPath, commandEnv, err := resolveCodeExecutorCommand(definition.Command)
	if err != nil {
		return nil, "", err
	}
	nativeSessionID := strings.TrimSpace(session.NativeSessionID)
	args := make([]string, 0)
	switch definition.ID {
	case "claude":
		args = append(args, claudeApprovalArgs(session.ApprovalPolicy)...)
		if nativeSessionID != "" && !nativeClaudeSessionExists(nativeSessionID, commandEnv) {
			nativeSessionID = ""
		}
		if nativeSessionID == "" {
			nativeSessionID = uuid.NewString()
			args = append(args, "--session-id", nativeSessionID)
		} else {
			args = append(args, "--resume", nativeSessionID)
		}
	case "opencode":
		if nativeSessionID != "" {
			args = append(args, "--session", nativeSessionID)
		}
	case "aider":
		args, nativeSessionID, err = buildNativeAiderArgs(nativeSessionID, session.ID, session.ApprovalPolicy)
		if err != nil {
			return nil, "", err
		}
	}
	command := exec.Command(commandPath, args...)
	command.Dir = session.WorkDir
	command.Env = commandEnv
	if err := configureCodeProviderCommand(definition.ID, command, session); err != nil {
		return nil, "", err
	}
	if definition.ID == "opencode" {
		if err := configureNativeOpenCodePermissions(command); err != nil {
			return nil, "", err
		}
	}
	return command, nativeSessionID, nil
}

func configureNativeOpenCodePermissions(command *exec.Cmd) error {
	config := map[string]any{}
	configContent := ""
	for _, item := range command.Env {
		if strings.HasPrefix(item, "OPENCODE_CONFIG_CONTENT=") {
			configContent = strings.TrimPrefix(item, "OPENCODE_CONFIG_CONTENT=")
			break
		}
	}
	if configContent != "" {
		if err := json.Unmarshal([]byte(configContent), &config); err != nil {
			return err
		}
	}
	config["permission"] = "allow"
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	command.Env = upsertEnvironment(command.Env, "OPENCODE_CONFIG_CONTENT", string(configJSON))
	return nil
}

func buildNativeAiderArgs(nativeSessionID string, sessionID uint, approvalPolicy string) ([]string, string, error) {
	if nativeSessionID == "" {
		nativeSessionID = "gopanel-" + strconv.FormatUint(uint64(sessionID), 10)
	}
	historyDir, err := ensureAiderHistoryDir()
	if err != nil {
		return nil, "", err
	}
	chatHistory := filepath.Join(historyDir, nativeSessionID+".chat.md")
	args := []string{"--chat-history-file", chatHistory, "--llm-history-file", filepath.Join(historyDir, nativeSessionID+".llm.log")}
	if approvalPolicy == codeApprovalPolicyFullAuto {
		args = append(args, "--yes-always")
	}
	if _, err := os.Stat(chatHistory); err == nil {
		args = append(args, "--restore-chat-history")
	}
	return args, nativeSessionID, nil
}
