package api

import (
	"errors"
	"strings"
)

func normalizeAIAgentName(agentName string) (string, error) {
	agentName = strings.ToLower(strings.TrimSpace(agentName))
	if agentName == "" {
		return "trae", nil
	}
	switch agentName {
	case "trae", "aider":
		return agentName, nil
	default:
		return "", errors.New("unsupported AI agent")
	}
}

func buildAIAgentExecArgs(containerName, agentName, input string) ([]string, error) {
	agentName, err := normalizeAIAgentName(agentName)
	if err != nil {
		return nil, err
	}
	return []string{"exec", "-i", containerName, agentName, "--message", input}, nil
}
