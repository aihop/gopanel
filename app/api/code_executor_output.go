package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

type codeExecutorOutput struct {
	Message            string
	RawOutput          string
	NativeSessionID    string
	Model              string
	InputTokens        int64
	OutputTokens       int64
	CachedInputTokens  int64
	ReasoningTokens    int64
	TotalTokens        int64
	TokenUsageReported bool
}

func parseCodeExecutorOutput(executorID string, rawOutput []byte, preparedSessionID string) codeExecutorOutput {
	result := codeExecutorOutput{RawOutput: string(rawOutput), NativeSessionID: preparedSessionID}
	switch executorID {
	case "codex":
		parseCodexOutput(rawOutput, &result)
	case "grok":
		parseGrokOutput(rawOutput, &result)
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
	if result.TotalTokens == 0 {
		result.TotalTokens = result.InputTokens + result.OutputTokens
	}
	return result
}

func parseGrokOutput(rawOutput []byte, result *codeExecutorOutput) {
	var messages []string
	scanJSONLines(rawOutput, func(event map[string]any) {
		applyCodeUsageMap(result, event)
		switch eventType, _ := event["type"].(string); eventType {
		case "text":
			if text, ok := event["data"].(string); ok {
				messages = append(messages, text)
			}
		case "end":
			if sessionID, ok := firstCodeString(event, "sessionId", "session_id"); ok {
				result.NativeSessionID = sessionID
			}
		}
	})
	result.Message = strings.Join(messages, "")
}

func parseCodexOutput(rawOutput []byte, result *codeExecutorOutput) {
	scanJSONLines(rawOutput, func(event map[string]any) {
		eventType, _ := event["type"].(string)
		if eventType == "thread.started" {
			if threadID, ok := event["thread_id"].(string); ok {
				result.NativeSessionID = threadID
			}
		}
		applyCodeUsageMap(result, event)
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
		Result    string         `json:"result"`
		SessionID string         `json:"session_id"`
		Model     string         `json:"model"`
		Usage     map[string]any `json:"usage"`
	}
	if json.Unmarshal(bytes.TrimSpace(rawOutput), &payload) == nil {
		result.Message = payload.Result
		result.Model = payload.Model
		applyCodeUsageMap(result, payload.Usage)
		if payload.SessionID != "" {
			result.NativeSessionID = payload.SessionID
		}
	}
}

func parseOpenCodeOutput(rawOutput []byte, result *codeExecutorOutput) {
	var messages []string
	scanJSONLines(rawOutput, func(event map[string]any) {
		applyCodeUsageMap(result, event)
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

func applyCodeUsageMap(result *codeExecutorOutput, payload map[string]any) {
	if result == nil || payload == nil {
		return
	}
	if model, ok := firstCodeString(payload, "model", "model_id", "modelID"); ok {
		result.Model = model
	}
	for _, key := range []string{
		"input_tokens", "inputTokens", "output_tokens", "outputTokens",
		"cached_input_tokens", "cachedInputTokens", "cache_read_input_tokens", "cacheReadInputTokens",
		"reasoning_tokens", "reasoningTokens", "reasoning_output_tokens", "total_tokens", "totalTokens",
	} {
		if _, exists := payload[key]; exists {
			result.TokenUsageReported = true
			break
		}
	}
	if codeNumberExists(payload, "input") && codeNumberExists(payload, "output") {
		result.TokenUsageReported = true
	}
	for _, key := range []string{"usage", "token_usage", "tokenUsage", "tokens", "part"} {
		if nested, ok := payload[key].(map[string]any); ok {
			applyCodeUsageMap(result, nested)
		}
	}
	result.InputTokens = max(result.InputTokens, firstCodeInt(payload, "input_tokens", "inputTokens", "input"))
	result.OutputTokens = max(result.OutputTokens, firstCodeInt(payload, "output_tokens", "outputTokens", "output"))
	result.ReasoningTokens = max(result.ReasoningTokens, firstCodeInt(payload, "reasoning_tokens", "reasoningTokens", "reasoning_output_tokens", "reasoning"))
	cached := firstCodeInt(payload, "cached_input_tokens", "cachedInputTokens", "cache_read_input_tokens", "cacheReadInputTokens")
	if cache, ok := payload["cache"].(map[string]any); ok {
		cached = max(cached, firstCodeInt(cache, "read", "read_tokens", "readTokens"))
	}
	result.CachedInputTokens = max(result.CachedInputTokens, cached)
	result.TotalTokens = max(result.TotalTokens, firstCodeInt(payload, "total_tokens", "totalTokens", "total"))
}

func codeNumberExists(payload map[string]any, key string) bool {
	switch payload[key].(type) {
	case float64, json.Number:
		return true
	default:
		return false
	}
}

func firstCodeString(payload map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return value, true
		}
	}
	return "", false
}

func firstCodeInt(payload map[string]any, keys ...string) int64 {
	for _, key := range keys {
		switch value := payload[key].(type) {
		case float64:
			if value > 0 {
				return int64(value)
			}
		case json.Number:
			if parsed, err := value.Int64(); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return 0
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
