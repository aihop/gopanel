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

func codeEventMap(value any) map[string]any {
	event, _ := value.(map[string]any)
	return event
}

func conversationAssistantUpdate(executorID string, event map[string]any) (text string, replace bool) {
	if event == nil {
		return "", false
	}
	if text, replace, handled := conversationAssistantStreamEvent(event); handled {
		return text, replace
	}
	if text, replace, handled := conversationCodexAssistantUpdate(event); handled {
		return text, replace
	}
	switch executorID {
	case "grok":
		if event["type"] == "text" {
			text, _ = event["data"].(string)
		}
	case "codex":
		return "", false
	case "claude":
		text, _ = event["result"].(string)
		return text, text != ""
	case "opencode":
		if event["type"] != "text" {
			return "", false
		}
		text, _ = codeEventMap(event["part"])["text"].(string)
	}
	return text, false
}

func conversationAssistantStreamEvent(event map[string]any) (string, bool, bool) {
	method, _ := event["method"].(string)
	if method == "item/agentMessage/delta" {
		return firstCodeStringOrEmpty(eventParams(event), "delta"), false, true
	}
	if method == "session/update" || strings.HasSuffix(method, "/session/update") {
		update := codeEventMap(codeEventMap(event["params"])["update"])
		if firstCodeStringOrEmpty(update, "sessionUpdate") != "agent_message_chunk" {
			return "", false, true
		}
		return firstCodeStringOrEmpty(codeEventMap(update["content"]), "text"), false, true
	}
	switch event["type"] {
	case "content_block_delta":
		return firstCodeStringOrEmpty(codeEventMap(event["delta"]), "text"), false, true
	case "stream_event":
		nested := codeEventMap(event["event"])
		if nested == nil {
			nested = codeEventMap(event["data"])
		}
		if nested == nil {
			return "", false, true
		}
		if text, replace, handled := conversationAssistantStreamEvent(nested); handled {
			return text, replace, true
		}
		text, replace := conversationAssistantUpdate("", nested)
		return text, replace, true
	case "assistant":
		text := claudeAssistantMessageText(event)
		return text, true, text != ""
	case "item.delta":
		if delta, ok := event["delta"].(string); ok {
			return delta, false, true
		}
		return firstCodeStringOrEmpty(codeEventMap(event["delta"]), "text"), false, true
	default:
		return "", false, false
	}
}

func eventParams(event map[string]any) map[string]any {
	return codeEventMap(event["params"])
}

func conversationCodexAssistantUpdate(event map[string]any) (string, bool, bool) {
	eventType, _ := event["type"].(string)
	item := codeEventMap(event["item"])
	if firstCodeStringOrEmpty(item, "type") == "agent_message" {
		text := firstCodeStringOrEmpty(item, "text")
		if text == "" {
			text = assistantContentList(item["content"])
		}
		if text == "" {
			return "", false, eventType == "item.started" || eventType == "item.updated" || eventType == "item.completed"
		}
		return text, eventType != "item.started", true
	}
	payload := codeEventMap(event["payload"])
	payloadType := firstCodeStringOrEmpty(payload, "type")
	if eventType == "event_msg" && payloadType == "agent_message" {
		return anyEventString(payload["message"]), true, true
	}
	if (eventType == "event_msg" && payloadType == "agent_message_content_delta") || eventType == "agent_message_content_delta" {
		text := firstCodeStringOrEmpty(payload, "delta", "text")
		if text == "" {
			text = anyEventString(event["delta"])
		}
		return text, false, true
	}
	if eventType == "response_item" && payloadType == "message" && firstCodeStringOrEmpty(payload, "role") == "assistant" {
		return assistantContentList(payload["content"]), true, true
	}
	return "", false, false
}

func assistantContentList(value any) string {
	parts, ok := value.([]any)
	if !ok {
		return ""
	}
	var builder strings.Builder
	for _, part := range parts {
		item := codeEventMap(part)
		kind := firstCodeStringOrEmpty(item, "type")
		if kind != "" && kind != "text" && kind != "output_text" {
			continue
		}
		builder.WriteString(firstCodeStringOrEmpty(item, "text"))
	}
	return builder.String()
}

func anyEventString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		return firstCodeStringOrEmpty(typed, "text", "message", "delta")
	default:
		return ""
	}
}

func claudeAssistantMessageText(event map[string]any) string {
	message := codeEventMap(event["message"])
	parts, _ := message["content"].([]any)
	var builder strings.Builder
	for _, part := range parts {
		item := codeEventMap(part)
		if firstCodeStringOrEmpty(item, "type") != "text" {
			continue
		}
		builder.WriteString(firstCodeStringOrEmpty(item, "text"))
	}
	return builder.String()
}

func firstCodeStringOrEmpty(payload map[string]any, keys ...string) string {
	value, _ := firstCodeString(payload, keys...)
	return value
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
		params := codeEventMap(event["params"])
		if sessionID, ok := firstCodeString(params, "sessionId", "session_id"); ok {
			result.NativeSessionID = sessionID
		}
		update := codeEventMap(params["update"])
		applyCodeUsageMap(result, update)
		applyCodeUsageMap(result, codeEventMap(update["usage"]))
		text, replace := conversationAssistantUpdate("grok", event)
		if text != "" {
			if replace {
				messages = []string{text}
			} else {
				messages = append(messages, text)
			}
		}
		if eventType, _ := event["type"].(string); eventType == "end" {
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
		text, replace := conversationAssistantUpdate("codex", event)
		if text == "" {
			return
		}
		if eventType == "item.completed" || replace {
			result.Message = text
			return
		}
		if result.Message == "" {
			result.Message = text
			return
		}
		result.Message += text
	})
}

func parseClaudeOutput(rawOutput []byte, result *codeExecutorOutput) {
	assembled := ""
	foundResult := false
	scanJSONLines(rawOutput, func(event map[string]any) {
		applyCodeUsageMap(result, event)
		applyCodeUsageMap(result, codeEventMap(event["usage"]))
		if sessionID, ok := firstCodeString(event, "session_id", "sessionId"); ok {
			result.NativeSessionID = sessionID
		}
		if model, ok := firstCodeString(event, "model"); ok {
			result.Model = model
		}
		text, replace := conversationAssistantUpdate("claude", event)
		if eventType, _ := event["type"].(string); eventType == "result" && text != "" {
			result.Message = text
			foundResult = true
			return
		}
		if foundResult || text == "" {
			return
		}
		if replace {
			assembled = text
		} else {
			assembled += text
		}
	})
	if foundResult {
		return
	}
	if assembled != "" {
		result.Message = assembled
		return
	}
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
