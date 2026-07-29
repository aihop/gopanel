package api

import "testing"

func TestParseCodexOutput(t *testing.T) {
	raw := []byte("{\"type\":\"thread.started\",\"thread_id\":\"thread-1\"}\n" +
		"{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"done\"}}\n")
	result := parseCodeExecutorOutput("codex", raw, "")
	if result.NativeSessionID != "thread-1" || result.Message != "done" || result.RawOutput != string(raw) {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseClaudeOutput(t *testing.T) {
	raw := []byte(`{"result":"finished","session_id":"session-1"}`)
	result := parseCodeExecutorOutput("claude", raw, "")
	if result.NativeSessionID != "session-1" || result.Message != "finished" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseOpenCodeOutput(t *testing.T) {
	raw := []byte("{\"type\":\"text\",\"sessionID\":\"session-2\",\"part\":{\"text\":\"hello \"}}\n" +
		"{\"type\":\"text\",\"sessionID\":\"session-2\",\"part\":{\"text\":\"world\"}}\n")
	result := parseCodeExecutorOutput("opencode", raw, "")
	if result.NativeSessionID != "session-2" || result.Message != "hello world" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
