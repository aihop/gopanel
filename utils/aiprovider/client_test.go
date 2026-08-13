package aiprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallOpenAIChatCompletions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected authorization: %s", request.Header.Get("Authorization"))
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["max_tokens"] != float64(32) || payload["messages"] == nil {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"chat ok"}}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`))
	}))
	defer server.Close()

	response, err := Call(context.Background(), Config{
		Protocol: ProtocolOpenAIChat, BaseURL: server.URL, APIKey: "secret", Model: "chat-model",
	}, Request{Messages: []Message{{Role: "user", Content: "ping"}}, MaxTokens: 32})
	if err != nil {
		t.Fatal(err)
	}
	if response.Message.Content != "chat ok" || response.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestCallOpenAIResponsesWithToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/responses" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["input"] == nil || payload["max_output_tokens"] != float64(64) {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"response ok"}]},{"type":"function_call","call_id":"call_1","name":"inspect","arguments":"{\"id\":1}"}],"usage":{"input_tokens":4,"output_tokens":6,"total_tokens":10}}`))
	}))
	defer server.Close()

	response, err := Call(context.Background(), Config{
		Protocol: ProtocolOpenAIResponses, BaseURL: server.URL, APIKey: "secret", Model: "response-model",
	}, Request{Messages: []Message{{Role: "user", Content: "ping"}}, MaxTokens: 64})
	if err != nil {
		t.Fatal(err)
	}
	if response.Message.Content != "response ok" || len(response.Message.ToolCalls) != 1 ||
		response.Message.ToolCalls[0].Function.Name != "inspect" || response.Usage.TotalTokens != 10 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestCallAnthropicMessagesWithToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/messages" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("x-api-key") != "anthropic-secret" || request.Header.Get("anthropic-version") == "" {
			t.Fatalf("unexpected anthropic headers: %#v", request.Header)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["system"] != "system prompt" || payload["messages"] == nil {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"content":[{"type":"text","text":"anthropic ok"},{"type":"tool_use","id":"tool_1","name":"inspect","input":{"id":1}}],"usage":{"input_tokens":7,"output_tokens":8}}`))
	}))
	defer server.Close()

	response, err := Call(context.Background(), Config{
		Protocol: ProtocolAnthropic, BaseURL: server.URL, APIKey: "anthropic-secret", Model: "claude-model",
	}, Request{Messages: []Message{{Role: "system", Content: "system prompt"}, {Role: "user", Content: "ping"}}})
	if err != nil {
		t.Fatal(err)
	}
	if response.Message.Content != "anthropic ok" || len(response.Message.ToolCalls) != 1 ||
		response.Message.ToolCalls[0].Function.Name != "inspect" || response.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestNormalizeProtocolDefaultsLegacyValues(t *testing.T) {
	if NormalizeProtocol("") != ProtocolOpenAIChat {
		t.Fatal("legacy blank protocol should use Chat Completions")
	}
	if NormalizeProtocol("unknown") != "" {
		t.Fatal("unknown protocols must be rejected")
	}
}

func TestCallOpenAIResponsesReportsOutputLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"reasoning"}]}`))
	}))
	defer server.Close()

	_, err := Call(context.Background(), Config{
		Protocol: ProtocolOpenAIResponses, BaseURL: server.URL, Model: "reasoning-model",
	}, Request{Messages: []Message{{Role: "user", Content: "ping"}}, MaxTokens: 16})
	if err == nil || !strings.Contains(err.Error(), "token 上限") {
		t.Fatalf("应明确报告输出被截断，实际：%v", err)
	}
}

func TestCallReportsProtocolMismatch(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		body     string
	}{
		{name: "chat", protocol: ProtocolOpenAIChat, body: `{"output":[]}`},
		{name: "responses", protocol: ProtocolOpenAIResponses, body: `{"choices":[]}`},
		{name: "anthropic", protocol: ProtocolAnthropic, body: `{"choices":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			_, err := Call(context.Background(), Config{
				Protocol: test.protocol, BaseURL: server.URL, Model: "model",
			}, Request{Messages: []Message{{Role: "user", Content: "ping"}}})
			if err == nil || !strings.Contains(err.Error(), "协议不匹配") {
				t.Fatalf("应明确报告协议不匹配，实际：%v", err)
			}
		})
	}
}
