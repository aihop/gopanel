package aiprovider

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
)

const (
	ProtocolOpenAIChat      = "openai_chat_completions"
	ProtocolOpenAIResponses = "openai_responses"
	ProtocolAnthropic       = "anthropic_messages"
	defaultTimeout          = 120 * time.Second
)

type Config struct {
	Protocol string
	BaseURL  string
	APIKey   string
	Model    string
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Request struct {
	Messages        []Message
	Tools           []map[string]any
	ToolChoice      string
	Temperature     *float64
	Schema          map[string]any
	SchemaName      string
	ReasoningEffort string
	MaxTokens       int
}

type Usage struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
}

type Response struct {
	Message Message
	Usage   Usage
	Raw     string
}

func NormalizeProtocol(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", ProtocolOpenAIChat:
		return ProtocolOpenAIChat
	case ProtocolOpenAIResponses:
		return ProtocolOpenAIResponses
	case ProtocolAnthropic:
		return ProtocolAnthropic
	default:
		return ""
	}
}

func Call(ctx context.Context, config Config, input Request) (Response, error) {
	config.Protocol = NormalizeProtocol(config.Protocol)
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.Model = strings.TrimSpace(config.Model)
	if config.Protocol == "" {
		return Response{}, errors.New("不支持的 AI 接口协议")
	}
	if config.BaseURL == "" || config.Model == "" {
		return Response{}, errors.New("未配置可用的模型服务")
	}
	switch config.Protocol {
	case ProtocolOpenAIChat:
		return callOpenAIChat(ctx, config, input)
	case ProtocolOpenAIResponses:
		return callOpenAIResponses(ctx, config, input)
	case ProtocolAnthropic:
		return callAnthropic(ctx, config, input)
	default:
		return Response{}, errors.New("不支持的 AI 接口协议")
	}
}

func callOpenAIChat(ctx context.Context, config Config, input Request) (Response, error) {
	payload := map[string]any{"model": config.Model, "messages": input.Messages}
	applyOpenAIOptions(payload, input, "max_tokens")
	if len(input.Tools) > 0 {
		payload["tools"] = input.Tools
		if input.ToolChoice != "" {
			payload["tool_choice"] = input.ToolChoice
		}
	}
	raw, err := send(ctx, config.BaseURL+"/chat/completions", config.APIKey, false, payload)
	if err != nil {
		return Response{Raw: string(raw)}, err
	}
	var decoded struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &decoded) != nil || len(decoded.Choices) == 0 {
		return Response{Raw: string(raw)}, errors.New("响应结构与 OpenAI Chat Completions 协议不匹配，请检查接口协议和 Base URL")
	}
	return Response{
		Message: decoded.Choices[0].Message,
		Usage:   Usage{decoded.Usage.PromptTokens, decoded.Usage.CompletionTokens, decoded.Usage.TotalTokens},
		Raw:     string(raw),
	}, nil
}

func callOpenAIResponses(ctx context.Context, config Config, input Request) (Response, error) {
	payload := map[string]any{"model": config.Model, "input": responsesInput(input.Messages)}
	if input.Temperature != nil {
		payload["temperature"] = *input.Temperature
	}
	if input.MaxTokens > 0 {
		payload["max_output_tokens"] = input.MaxTokens
	}
	if effort := strings.TrimSpace(input.ReasoningEffort); effort != "" {
		payload["reasoning"] = map[string]string{"effort": effort}
	}
	if input.Schema != nil {
		payload["text"] = map[string]any{"format": map[string]any{
			"type": "json_schema", "name": schemaName(input), "strict": false, "schema": input.Schema,
		}}
	}
	if len(input.Tools) > 0 {
		payload["tools"] = responsesTools(input.Tools)
		if input.ToolChoice != "" {
			payload["tool_choice"] = input.ToolChoice
		}
	}
	raw, err := send(ctx, config.BaseURL+"/responses", config.APIKey, false, payload)
	if err != nil {
		return Response{Raw: string(raw)}, err
	}
	var decoded struct {
		Status            string `json:"status"`
		IncompleteDetails struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Output []struct {
			Type      string `json:"type"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
			Content   []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
			TotalTokens  int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &decoded) != nil {
		return Response{Raw: string(raw)}, errors.New("模型服务响应无法解析")
	}
	if decoded.Output == nil {
		return Response{Raw: string(raw)}, errors.New("响应结构与 OpenAI Responses 协议不匹配，请检查接口协议和 Base URL")
	}
	message := Message{Role: "assistant"}
	texts := make([]string, 0)
	for _, item := range decoded.Output {
		switch item.Type {
		case "message":
			for _, content := range item.Content {
				if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
					texts = append(texts, strings.TrimSpace(content.Text))
				}
			}
		case "function_call":
			call := ToolCall{ID: item.CallID, Type: "function"}
			call.Function.Name, call.Function.Arguments = item.Name, item.Arguments
			message.ToolCalls = append(message.ToolCalls, call)
		}
	}
	message.Content = strings.Join(texts, "\n")
	if message.Content == "" && len(message.ToolCalls) == 0 {
		if decoded.Status == "incomplete" && decoded.IncompleteDetails.Reason == "max_output_tokens" {
			return Response{Raw: string(raw)}, errors.New("Responses 模型输出达到 token 上限，尚未生成正文")
		}
		return Response{Raw: string(raw)}, errors.New("Responses 服务未返回文本或工具调用，请检查模型名称和接口兼容性")
	}
	return Response{
		Message: message,
		Usage: Usage{
			InputTokens:  decoded.Usage.InputTokens,
			OutputTokens: decoded.Usage.OutputTokens,
			TotalTokens:  decoded.Usage.TotalTokens,
		},
		Raw: string(raw),
	}, nil
}

func callAnthropic(ctx context.Context, config Config, input Request) (Response, error) {
	if input.Schema != nil {
		return Response{}, errors.New("Anthropic Messages 不支持当前结构化输出格式")
	}
	if strings.TrimSpace(input.ReasoningEffort) != "" {
		return Response{}, errors.New("Anthropic Messages 不支持 reasoning_effort")
	}
	system, messages := anthropicMessages(input.Messages)
	payload := map[string]any{"model": config.Model, "messages": messages, "max_tokens": max(input.MaxTokens, 16)}
	if system != "" {
		payload["system"] = system
	}
	if input.Temperature != nil {
		payload["temperature"] = *input.Temperature
	}
	if len(input.Tools) > 0 {
		payload["tools"] = anthropicTools(input.Tools)
	}
	raw, err := send(ctx, config.BaseURL+"/messages", config.APIKey, true, payload)
	if err != nil {
		return Response{Raw: string(raw)}, err
	}
	var decoded struct {
		Content []struct {
			Type  string         `json:"type"`
			Text  string         `json:"text"`
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"content"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &decoded) != nil {
		return Response{Raw: string(raw)}, errors.New("模型服务响应无法解析")
	}
	if decoded.Content == nil {
		return Response{Raw: string(raw)}, errors.New("响应结构与 Anthropic Messages 协议不匹配，请检查接口协议和 Base URL")
	}
	message := Message{Role: "assistant"}
	texts := make([]string, 0)
	for _, content := range decoded.Content {
		if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
			texts = append(texts, strings.TrimSpace(content.Text))
		}
		if content.Type == "tool_use" {
			arguments, _ := json.Marshal(content.Input)
			call := ToolCall{ID: content.ID, Type: "function"}
			call.Function.Name, call.Function.Arguments = content.Name, string(arguments)
			message.ToolCalls = append(message.ToolCalls, call)
		}
	}
	message.Content = strings.Join(texts, "\n")
	if message.Content == "" && len(message.ToolCalls) == 0 {
		return Response{Raw: string(raw)}, errors.New("Anthropic Messages 服务未返回文本或工具调用，请检查模型名称和接口兼容性")
	}
	usage := Usage{InputTokens: decoded.Usage.InputTokens, OutputTokens: decoded.Usage.OutputTokens}
	usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	return Response{Message: message, Usage: usage, Raw: string(raw)}, nil
}

func applyOpenAIOptions(payload map[string]any, input Request, maxTokensKey string) {
	if input.Temperature != nil {
		payload["temperature"] = *input.Temperature
	}
	if effort := strings.TrimSpace(input.ReasoningEffort); effort != "" {
		payload["reasoning_effort"] = effort
	}
	if input.MaxTokens > 0 {
		payload[maxTokensKey] = input.MaxTokens
	}
	if input.Schema != nil {
		payload["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{
			"name": schemaName(input), "strict": false, "schema": input.Schema,
		}}
	}
}

func send(ctx context.Context, endpoint, apiKey string, anthropic bool, payload map[string]any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if anthropic {
		request.Header.Set("x-api-key", strings.TrimSpace(apiKey))
		request.Header.Set("anthropic-version", "2023-06-01")
	} else if strings.TrimSpace(apiKey) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("调用模型服务失败：%w", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return raw, fmt.Errorf("模型服务返回 %d：%s", response.StatusCode, truncate(string(raw), 200))
	}
	return raw, nil
}

func schemaName(input Request) string {
	if name := strings.TrimSpace(input.SchemaName); name != "" {
		return name
	}
	return "gopanel_output"
}

func truncate(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}
