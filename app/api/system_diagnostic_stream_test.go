package api

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestReadSystemDiagnosticLLMStreamAccumulatesContentAndToolCalls(t *testing.T) {
	input := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"正在"}}]}`,
		`data: {"choices":[{"delta":{"content":"分析"}}]}`,
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
	if len(message.ToolCalls) != 1 || message.ToolCalls[0].Type != "function" || message.ToolCalls[0].Function.Name != "query_panel_database" || message.ToolCalls[0].Function.Arguments != `{"sql":"SELECT id FROM backup_records"}` {
		t.Fatalf("unexpected streamed tool call: %#v", message.ToolCalls)
	}
	if usage.TotalTokens != 16 {
		t.Fatalf("unexpected usage: %#v", usage)
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
