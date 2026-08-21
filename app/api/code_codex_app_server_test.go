package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"testing"
)

type bufferWriteCloser struct {
	bytes.Buffer
}

func (writer *bufferWriteCloser) Close() error { return nil }

func TestCodexAppServerStreamsAgentMessageDeltas(t *testing.T) {
	hub := newConversationStreamHub()
	codeConversationStreams = hub
	_, events, cancel := hub.Subscribe(17)
	defer cancel()
	client := &codexAppServerClient{sessionID: 17}
	for _, line := range []string{
		`{"method":"item/agentMessage/delta","params":{"itemId":"first","delta":"hello"}}`,
		`{"method":"item/agentMessage/delta","params":{"itemId":"second","delta":"world"}}`,
	} {
		var message codexAppServerMessage
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			t.Fatal(err)
		}
		if err := client.handleNotification(message.Method, message.Params); err != nil {
			t.Fatal(err)
		}
	}
	if client.text != "hello\n\nworld" {
		t.Fatalf("streamed text = %q", client.text)
	}
	for _, expected := range []string{"hello", "\n\nworld"} {
		select {
		case event := <-events:
			if event.Type != "delta" || event.Content != expected {
				t.Fatalf("unexpected SSE event: %#v", event)
			}
		default:
			t.Fatalf("missing SSE delta %q", expected)
		}
	}
}

func TestCodexAppServerRequestHandlesNotificationsBeforeResponse(t *testing.T) {
	responses := strings.Join([]string{
		`{"method":"item/agentMessage/delta","params":{"itemId":"message","delta":"live"}}`,
		`{"id":1,"result":{"turn":{"id":"turn-1"}}}`,
	}, "\n")
	written := &bufferWriteCloser{}
	client := &codexAppServerClient{
		stdin:     written,
		reader:    bufio.NewScanner(strings.NewReader(responses)),
		output:    &boundedCodeOutput{},
		requestID: 1,
	}
	result, err := client.request("turn/start", map[string]string{"threadId": "thread-1"})
	if err != nil || !bytes.Contains(result, []byte(`"turn-1"`)) {
		t.Fatalf("unexpected response: %s, %v", result, err)
	}
	if client.text != "live" || !strings.Contains(written.String(), `"method":"turn/start"`) {
		t.Fatalf("request did not preserve streamed delta: text=%q request=%q", client.text, written.String())
	}
}

func TestCodexAppServerTurnCompletionAndUsage(t *testing.T) {
	client := &codexAppServerClient{}
	usage := json.RawMessage(`{"tokenUsage":{"total":{"inputTokens":12,"cachedInputTokens":4,"outputTokens":5,"reasoningOutputTokens":2,"totalTokens":17}}}`)
	if err := client.handleNotification("thread/tokenUsage/updated", usage); err != nil {
		t.Fatal(err)
	}
	completed := json.RawMessage(`{"turn":{"status":"completed"}}`)
	if err := client.handleNotification("turn/completed", completed); err != nil {
		t.Fatal(err)
	}
	if client.status != "completed" || client.usage.TotalTokens != 17 || client.usage.CachedInputTokens != 4 {
		t.Fatalf("unexpected completion state: status=%q usage=%#v", client.status, client.usage)
	}
}

func TestCodexAppServerFallsBackToCompletedAgentMessage(t *testing.T) {
	client := &codexAppServerClient{text: "streamed", itemID: "first"}
	completed := json.RawMessage(`{"item":{"id":"second","type":"agentMessage","text":"final"}}`)
	if err := client.handleNotification("item/completed", completed); err != nil {
		t.Fatal(err)
	}
	if client.text != "streamed\n\nfinal" || client.itemID != "second" {
		t.Fatalf("completed message fallback = %q (%q)", client.text, client.itemID)
	}
}

func TestCodexAppServerDeclinesInteractiveRequests(t *testing.T) {
	for _, test := range []struct {
		method string
		want   string
	}{
		{method: "item/commandExecution/requestApproval", want: `"decision":"decline"`},
		{method: "item/fileChange/requestApproval", want: `"decision":"decline"`},
		{method: "item/tool/requestUserInput", want: `"answers":{}`},
		{method: "mcpServer/elicitation/request", want: `"action":"decline"`},
	} {
		written := &bufferWriteCloser{}
		client := &codexAppServerClient{stdin: written}
		if err := client.respondToApproval(codexAppServerMessage{ID: json.RawMessage(`42`), Method: test.method}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(written.String(), test.want) {
			t.Fatalf("%s response = %s", test.method, written.String())
		}
	}
}

func TestCodexAppServerCommandKeepsNativeTerminalIndependent(t *testing.T) {
	command := exec.CommandContext(context.Background(), "codex", codexSandboxArgs(codeApprovalPolicySafeAuto)...)
	command.Args = append(command.Args, "app-server")
	command.Args = addCodexWritableDirArgs(command.Args, []string{"/tmp/repository.git"})
	joined := strings.Join(command.Args, " ")
	if !strings.Contains(joined, "--add-dir /tmp/repository.git app-server") {
		t.Fatalf("app-server writable args are misplaced: %s", joined)
	}
	native := &codexAppServerClient{ctx: context.Background(), stdin: &bufferWriteCloser{}, reader: bufio.NewScanner(strings.NewReader("")), output: &boundedCodeOutput{}}
	if _, err := native.request("initialize", map[string]any{}); err != io.EOF {
		t.Fatalf("empty app-server stream error = %v", err)
	}
}
