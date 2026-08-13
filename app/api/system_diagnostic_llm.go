package api

import (
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
)

const systemDiagnosticMaxToolRounds = 6

type systemDiagnosticLLMMessage struct {
	Role       string                     `json:"role"`
	Content    string                     `json:"content,omitempty"`
	ToolCallID string                     `json:"tool_call_id,omitempty"`
	ToolCalls  []systemDiagnosticToolCall `json:"tool_calls,omitempty"`
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

func runSystemDiagnosticLLM(ctx context.Context, account *model.AIProviderAccount, apiKey string, messages []systemDiagnosticLLMMessage) (string, string, systemDiagnosticUsage, []systemDiagnosticToolAudit, error) {
	var totalUsage systemDiagnosticUsage
	rawOutputs := make([]string, 0, systemDiagnosticMaxToolRounds)
	audits := make([]systemDiagnosticToolAudit, 0, systemDiagnosticMaxToolRounds)
	for round := 0; round < systemDiagnosticMaxToolRounds; round++ {
		assistant, usage, raw, err := callSystemDiagnosticLLM(ctx, account, apiKey, messages)
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
			return assistant.Content, strings.Join(rawOutputs, "\n"), totalUsage, audits, nil
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

func callSystemDiagnosticLLM(ctx context.Context, account *model.AIProviderAccount, apiKey string, messages []systemDiagnosticLLMMessage) (systemDiagnosticLLMMessage, systemDiagnosticUsage, string, error) {
	payload := map[string]any{
		"model": account.Model, "messages": messages, "tools": systemDiagnosticTools(),
		"tool_choice": "auto", "max_tokens": 2200,
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
	raw, err := io.ReadAll(io.LimitReader(response.Body, 3<<20))
	if err != nil {
		return systemDiagnosticLLMMessage{}, systemDiagnosticUsage{}, "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return systemDiagnosticLLMMessage{}, systemDiagnosticUsage{}, string(raw), fmt.Errorf("诊断模型返回 %d", response.StatusCode)
	}
	var decoded struct {
		Choices []struct {
			Message systemDiagnosticLLMMessage `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil || len(decoded.Choices) == 0 {
		return systemDiagnosticLLMMessage{}, systemDiagnosticUsage{}, string(raw), errors.New("诊断模型响应无法解析")
	}
	return decoded.Choices[0].Message, systemDiagnosticUsage{decoded.Usage.PromptTokens, decoded.Usage.CompletionTokens, decoded.Usage.TotalTokens}, string(raw), nil
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
