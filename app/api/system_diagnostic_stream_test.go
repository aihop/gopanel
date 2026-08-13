package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

type systemDiagnosticRoundTripFunc func(*http.Request) (*http.Response, error)

func (function systemDiagnosticRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestReadSystemDiagnosticLLMStreamAccumulatesContentAndToolCalls(t *testing.T) {
	input := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"正在","reasoning_content":"先检查"}}]}`,
		`data: {"choices":[{"delta":{"content":"分析","reasoning_content":"系统状态"}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"query_panel_","arguments":"{\"sql\":\"SELECT "}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"type":"function","function":{"name":"database","arguments":"id FROM backup_records\"}"}}]}}]}`,
		`data: {"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":4,"total_tokens":16}}`,
		`data: [DONE]`,
	}, "\n\n")
	var deltas strings.Builder
	message, usage, _, err := readSystemDiagnosticLLMStream(strings.NewReader(input), func(delta string) error {
		deltas.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if message.Content != "正在分析" || deltas.String() != message.Content {
		t.Fatalf("unexpected streamed content: message=%q deltas=%q", message.Content, deltas.String())
	}
	if message.ReasoningContent != "先检查系统状态" {
		t.Fatalf("unexpected reasoning content: %q", message.ReasoningContent)
	}
	if len(message.ToolCalls) != 1 || message.ToolCalls[0].Type != "function" || message.ToolCalls[0].Function.Name != "query_panel_database" || message.ToolCalls[0].Function.Arguments != `{"sql":"SELECT id FROM backup_records"}` {
		t.Fatalf("unexpected streamed tool call: %#v", message.ToolCalls)
	}
	if usage.TotalTokens != 16 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
}

func TestSystemDiagnosticAssistantMessagePreservesReasoningContent(t *testing.T) {
	message := systemDiagnosticLLMMessage{
		Role: "assistant", ReasoningContent: "必须回传的思考内容",
		ToolCalls: []systemDiagnosticToolCall{{ID: "call_1", Type: "function"}},
	}
	encoded, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"reasoning_content":"必须回传的思考内容"`) {
		t.Fatalf("reasoning content missing from next request: %s", encoded)
	}
}

func TestRunSystemDiagnosticLLMReturnsReasoningContentOnToolRound(t *testing.T) {
	requests := 0
	previousClient := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previousClient })
	http.DefaultClient = &http.Client{Transport: systemDiagnosticRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		var payload struct {
			Messages []systemDiagnosticLLMMessage `json:"messages"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		responseBody := "data: {\"choices\":[{\"delta\":{\"content\":\"诊断完成\"}}]}\n\ndata: [DONE]\n\n"
		if requests == 1 {
			responseBody = "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"先读取快照\"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"unknown_test_tool\",\"arguments\":\"{}\"}}]}}]}\n\n" +
				"data: [DONE]\n\n"
		}
		if requests == 2 && (len(payload.Messages) < 2 || payload.Messages[len(payload.Messages)-2].ReasoningContent != "先读取快照") {
			t.Fatalf("reasoning content was not returned: %#v", payload.Messages)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Request:    request,
		}, nil
	})}

	answer, _, _, _, err := runSystemDiagnosticLLM(context.Background(), &model.AIProviderAccount{
		BaseURL: "https://deepseek.test/v1", Model: "deepseek-test",
	}, "secret", []systemDiagnosticLLMMessage{{Role: "user", Content: "检查系统"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if answer != "诊断完成" || requests != 2 {
		t.Fatalf("unexpected diagnostic result: answer=%q requests=%d", answer, requests)
	}
}

func TestSystemDiagnosticProviderErrorExtractsMessage(t *testing.T) {
	raw := []byte(`{"error":{"message":"messages[2]: unknown variant functionfunction","type":"invalid_request_error"}}`)
	if message := systemDiagnosticProviderError(raw); message != "messages[2]: unknown variant functionfunction" {
		t.Fatalf("unexpected provider error: %q", message)
	}
}

func TestWriteSystemDiagnosticSSEWritesStructuredEvent(t *testing.T) {
	var output bytes.Buffer
	writer := bufio.NewWriter(&output)
	if err := writeSystemDiagnosticSSE(writer, "delta", map[string]string{"content": "第一行\n第二行"}); err != nil {
		t.Fatal(err)
	}
	want := "event: delta\ndata: {\"content\":\"第一行\\n第二行\"}\n\n"
	if output.String() != want {
		t.Fatalf("SSE output = %q, want %q", output.String(), want)
	}
}
