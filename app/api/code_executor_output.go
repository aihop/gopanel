package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

type codeExecutorOutput struct {
	Message         string
	RawOutput       string
	NativeSessionID string
}

func parseCodeExecutorOutput(executorID string, rawOutput []byte, preparedSessionID string) codeExecutorOutput {
	result := codeExecutorOutput{RawOutput: string(rawOutput), NativeSessionID: preparedSessionID}
	switch executorID {
	case "codex":
		parseCodexOutput(rawOutput, &result)
	case "claude":
		parseClaudeOutput(rawOutput, &result)
	case "opencode":
		parseOpenCodeOutput(rawOutput, &result)
	default:
		result.Message = strings.TrimSpace(string(rawOutput))
	}
	if strings.TrimSpace(result.Message) == "" {
		result.Message = strings.TrimSpace(string(rawOutput))
	}
	return result
}

func parseCodexOutput(rawOutput []byte, result *codeExecutorOutput) {
	scanJSONLines(rawOutput, func(event map[string]any) {
		eventType, _ := event["type"].(string)
		if eventType == "thread.started" {
			if threadID, ok := event["thread_id"].(string); ok {
				result.NativeSessionID = threadID
			}
		}
		if eventType != "item.completed" {
			return
		}
		item, _ := event["item"].(map[string]any)
		if item["type"] != "agent_message" {
			return
		}
		if text, ok := item["text"].(string); ok {
			result.Message = text
		}
	})
}

func parseClaudeOutput(rawOutput []byte, result *codeExecutorOutput) {
	var payload struct {
		Result    string `json:"result"`
		SessionID string `json:"session_id"`
	}
	if json.Unmarshal(bytes.TrimSpace(rawOutput), &payload) == nil {
		result.Message = payload.Result
		if payload.SessionID != "" {
			result.NativeSessionID = payload.SessionID
		}
	}
}

func parseOpenCodeOutput(rawOutput []byte, result *codeExecutorOutput) {
	var messages []string
	scanJSONLines(rawOutput, func(event map[string]any) {
		if sessionID, ok := event["sessionID"].(string); ok && sessionID != "" {
			result.NativeSessionID = sessionID
		}
		if event["type"] != "text" {
			return
		}
		part, _ := event["part"].(map[string]any)
		if text, ok := part["text"].(string); ok && strings.TrimSpace(text) != "" {
			messages = append(messages, text)
		}
	})
	result.Message = strings.Join(messages, "")
}

func scanJSONLines(rawOutput []byte, visit func(map[string]any)) {
	scanner := bufio.NewScanner(bytes.NewReader(rawOutput))
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, 4*1024*1024)
	for scanner.Scan() {
		var event map[string]any
		if json.Unmarshal(scanner.Bytes(), &event) == nil {
			visit(event)
		}
	}
}
