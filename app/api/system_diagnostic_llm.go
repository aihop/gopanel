package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/utils/aiprovider"
)

const systemDiagnosticMaxToolRounds = 6

type systemDiagnosticLLMMessage struct {
	Role             string                     `json:"role"`
	Content          string                     `json:"content,omitempty"`
	ReasoningContent string                     `json:"reasoning_content,omitempty"`
	ToolCallID       string                     `json:"tool_call_id,omitempty"`
	ToolCalls        []systemDiagnosticToolCall `json:"tool_calls,omitempty"`
}

type systemDiagnosticToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type systemDiagnosticUsage struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
}

type systemDiagnosticToolAudit struct {
	Tool           string `json:"tool"`
	SQLFingerprint string `json:"sqlFingerprint,omitempty"`
	Succeeded      bool   `json:"succeeded"`
}

func runSystemDiagnosticLLM(ctx context.Context, account *model.AIProviderAccount, apiKey string, messages []systemDiagnosticLLMMessage, onDelta func(string) error) (string, string, systemDiagnosticUsage, []systemDiagnosticToolAudit, error) {
	var totalUsage systemDiagnosticUsage
	rawOutputs := make([]string, 0, systemDiagnosticMaxToolRounds)
	audits := make([]systemDiagnosticToolAudit, 0, systemDiagnosticMaxToolRounds)
	var streamedAnswer strings.Builder
	for round := 0; round < systemDiagnosticMaxToolRounds; round++ {
		assistant, usage, raw, err := callSystemDiagnosticLLM(ctx, account, apiKey, messages, func(delta string) error {
			streamedAnswer.WriteString(delta)
			if onDelta != nil {
				return onDelta(delta)
			}
			return nil
		})
		totalUsage.InputTokens += usage.InputTokens
		totalUsage.OutputTokens += usage.OutputTokens
		totalUsage.TotalTokens += usage.TotalTokens
		if raw != "" {
			rawOutputs = append(rawOutputs, raw)
		}
		if err != nil {
			return "", strings.Join(rawOutputs, "\n"), totalUsage, audits, err
		}
		messages = append(messages, assistant)
		if len(assistant.ToolCalls) == 0 {
			if strings.TrimSpace(assistant.Content) == "" {
				return "", strings.Join(rawOutputs, "\n"), totalUsage, audits, errors.New("诊断模型返回了空结果")
			}
			return streamedAnswer.String(), strings.Join(rawOutputs, "\n"), totalUsage, audits, nil
		}
		for _, toolCall := range assistant.ToolCalls {
			result, audit := executeSystemDiagnosticTool(toolCall)
			audits = append(audits, audit)
			encoded, _ := json.Marshal(result)
			messages = append(messages, systemDiagnosticLLMMessage{Role: "tool", ToolCallID: toolCall.ID, Content: string(encoded)})
		}
	}
	return "", strings.Join(rawOutputs, "\n"), totalUsage, audits, errors.New("诊断模型查询次数过多，请缩小问题范围后重试")
}

func callSystemDiagnosticLLM(ctx context.Context, account *model.AIProviderAccount, apiKey string, messages []systemDiagnosticLLMMessage, onDelta func(string) error) (systemDiagnosticLLMMessage, systemDiagnosticUsage, string, error) {
	if aiprovider.NormalizeProtocol(account.Protocol) != aiprovider.ProtocolOpenAIChat {
		return callSystemDiagnosticProvider(ctx, account, apiKey, messages, onDelta)
	}
	payload := map[string]any{
		"model": account.Model, "messages": messages, "tools": systemDiagnosticTools(),
		"tool_choice": "auto", "max_tokens": 2200, "stream": true,
	}
	if account.SupportsTemperature {
		payload["temperature"] = 0
	}
	if account.SupportsReasoningEffort && strings.TrimSpace(account.DefaultReasoningEffort) != "" {
		payload["reasoning_effort"] = account.DefaultReasoningEffort
	}
	body, _ := json.Marshal(payload)
	requestCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, strings.TrimRight(account.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return systemDiagnosticLLMMessage{}, systemDiagnosticUsage{}, "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+apiKey)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return systemDiagnosticLLMMessage{}, systemDiagnosticUsage{}, "", fmt.Errorf("调用诊断模型失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(response.Body, 3<<20))
		return systemDiagnosticLLMMessage{}, systemDiagnosticUsage{}, string(raw), fmt.Errorf("诊断模型返回 %d：%s", response.StatusCode, systemDiagnosticProviderError(raw))
	}
	return readSystemDiagnosticLLMStream(response.Body, onDelta)
}

func callSystemDiagnosticProvider(ctx context.Context, account *model.AIProviderAccount, apiKey string, messages []systemDiagnosticLLMMessage, onDelta func(string) error) (systemDiagnosticLLMMessage, systemDiagnosticUsage, string, error) {
	providerMessages := make([]aiprovider.Message, 0, len(messages))
	for _, message := range messages {
		converted := aiprovider.Message{Role: message.Role, Content: message.Content, ToolCallID: message.ToolCallID}
		for _, toolCall := range message.ToolCalls {
			call := aiprovider.ToolCall{ID: toolCall.ID, Type: toolCall.Type}
			call.Function.Name = toolCall.Function.Name
			call.Function.Arguments = toolCall.Function.Arguments
			converted.ToolCalls = append(converted.ToolCalls, call)
		}
		providerMessages = append(providerMessages, converted)
	}
	request := aiprovider.Request{
		Messages: providerMessages, Tools: systemDiagnosticTools(), ToolChoice: "auto", MaxTokens: 2200,
	}
	if account.SupportsTemperature {
		temperature := 0.0
		request.Temperature = &temperature
	}
	if account.SupportsReasoningEffort {
		request.ReasoningEffort = account.DefaultReasoningEffort
	}
	response, err := aiprovider.Call(ctx, aiprovider.Config{
		Protocol: account.Protocol, BaseURL: account.BaseURL, APIKey: apiKey, Model: account.Model,
	}, request)
	if err != nil {
		return systemDiagnosticLLMMessage{}, systemDiagnosticUsage{}, response.Raw, fmt.Errorf("调用诊断模型失败: %w", err)
	}
	assistant := systemDiagnosticLLMMessage{Role: "assistant", Content: response.Message.Content}
	for _, toolCall := range response.Message.ToolCalls {
		call := systemDiagnosticToolCall{ID: toolCall.ID, Type: toolCall.Type}
		call.Function.Name = toolCall.Function.Name
		call.Function.Arguments = toolCall.Function.Arguments
		assistant.ToolCalls = append(assistant.ToolCalls, call)
	}
	if onDelta != nil && assistant.Content != "" {
		if err := onDelta(assistant.Content); err != nil {
			return assistant, systemDiagnosticUsage{}, response.Raw, err
		}
	}
	usage := systemDiagnosticUsage{
		response.Usage.InputTokens, response.Usage.OutputTokens, response.Usage.TotalTokens,
	}
	return assistant, usage, response.Raw, nil
}

func readSystemDiagnosticLLMStream(reader io.Reader, onDelta func(string) error) (systemDiagnosticLLMMessage, systemDiagnosticUsage, string, error) {
	assistant := systemDiagnosticLLMMessage{Role: "assistant"}
	var usage systemDiagnosticUsage
	var raw strings.Builder
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 3<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		if raw.Len() < 3<<20 {
			raw.WriteString(data)
			raw.WriteByte('\n')
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					ToolCalls        []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int64 `json:"prompt_tokens"`
				CompletionTokens int64 `json:"completion_tokens"`
				TotalTokens      int64 `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return assistant, usage, raw.String(), errors.New("诊断模型流式响应无法解析")
		}
		if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 || chunk.Usage.TotalTokens > 0 {
			usage = systemDiagnosticUsage{chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens, chunk.Usage.TotalTokens}
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		assistant.ReasoningContent += delta.ReasoningContent
		if delta.Content != "" {
			assistant.Content += delta.Content
			if onDelta != nil {
				if err := onDelta(delta.Content); err != nil {
					return assistant, usage, raw.String(), err
				}
			}
		}
		for _, toolDelta := range delta.ToolCalls {
			for len(assistant.ToolCalls) <= toolDelta.Index {
				assistant.ToolCalls = append(assistant.ToolCalls, systemDiagnosticToolCall{})
			}
			toolCall := &assistant.ToolCalls[toolDelta.Index]
			toolCall.ID += toolDelta.ID
			if toolDelta.Type != "" {
				toolCall.Type = toolDelta.Type
			}
			toolCall.Function.Name += toolDelta.Function.Name
			toolCall.Function.Arguments += toolDelta.Function.Arguments
		}
	}
	if err := scanner.Err(); err != nil {
		return assistant, usage, raw.String(), fmt.Errorf("读取诊断模型流失败: %w", err)
	}
	if assistant.Content == "" && len(assistant.ToolCalls) == 0 {
		return assistant, usage, raw.String(), errors.New("诊断模型返回了空结果")
	}
	return assistant, usage, raw.String(), nil
}

func systemDiagnosticProviderError(raw []byte) string {
	var response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &response) == nil && strings.TrimSpace(response.Error.Message) != "" {
		return truncateSystemDiagnosticError(response.Error.Message, 300)
	}
	return truncateSystemDiagnosticError(string(raw), 300)
}

func truncateSystemDiagnosticError(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func executeSystemDiagnosticTool(call systemDiagnosticToolCall) (any, systemDiagnosticToolAudit) {
	audit := systemDiagnosticToolAudit{Tool: call.Function.Name}
	switch call.Function.Name {
	case "get_system_snapshot":
		audit.Succeeded = true
		return map[string]any{"ok": true, "data": buildSystemDiagnosticSnapshot()}, audit
	case "list_panel_tables":
		var input struct {
			Keyword string `json:"keyword"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &input); err != nil {
			return map[string]any{"ok": false, "error": "工具参数无效"}, audit
		}
		tables, err := listSystemDiagnosticTables(input.Keyword)
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}, audit
		}
		audit.Succeeded = true
		return map[string]any{"ok": true, "tables": tables}, audit
	case "describe_panel_table":
		var input struct {
			Table string `json:"table"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &input); err != nil {
			return map[string]any{"ok": false, "error": "工具参数无效"}, audit
		}
		columns, err := describeSystemDiagnosticTable(input.Table)
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}, audit
		}
		audit.Succeeded = true
		return map[string]any{"ok": true, "table": input.Table, "columns": columns}, audit
	case "query_panel_database":
		var input struct {
			SQL string `json:"sql"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &input); err != nil {
			return map[string]any{"ok": false, "error": "工具参数无效"}, audit
		}
		audit.SQLFingerprint = systemDiagnosticSQLFingerprint(input.SQL)
		result, err := querySystemDiagnosticDatabase(input.SQL)
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}, audit
		}
		audit.Succeeded = true
		return map[string]any{"ok": true, "data": result}, audit
	default:
		return map[string]any{"ok": false, "error": "未知诊断工具"}, audit
	}
}

func systemDiagnosticTools() []map[string]any {
	return []map[string]any{
		{"type": "function", "function": map[string]any{"name": "get_system_snapshot", "description": "读取 GoPanel 控制平面状态、资源数量和最近失败摘要", "parameters": map[string]any{"type": "object", "properties": map[string]any{}}}},
		{"type": "function", "function": map[string]any{"name": "list_panel_tables", "description": "列出诊断中心允许读取的 GoPanel 数据表", "parameters": map[string]any{"type": "object", "properties": map[string]any{"keyword": map[string]any{"type": "string"}}}}},
		{"type": "function", "function": map[string]any{"name": "describe_panel_table", "description": "查看一张允许诊断的 GoPanel 表所包含的安全字段和数据库类型", "parameters": map[string]any{"type": "object", "required": []string{"table"}, "properties": map[string]any{"table": map[string]any{"type": "string"}}}}},
		{"type": "function", "function": map[string]any{"name": "query_panel_database", "description": "对 GoPanel 主数据库执行单条受控只读 SQL，最多返回 100 行", "parameters": map[string]any{"type": "object", "required": []string{"sql"}, "properties": map[string]any{"sql": map[string]any{"type": "string"}}}}},
	}
}
