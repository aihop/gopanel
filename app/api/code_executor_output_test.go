package api

import "testing"

func TestParseCodexOutput(t *testing.T) {
	raw := []byte("{\"type\":\"thread.started\",\"thread_id\":\"thread-1\"}\n" +
		"{\"type\":\"turn.completed\",\"model\":\"gpt-5\",\"usage\":{\"input_tokens\":120,\"cached_input_tokens\":80,\"output_tokens\":30,\"reasoning_output_tokens\":10}}\n" +
		"{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"done\"}}\n")
	result := parseCodeExecutorOutput("codex", raw, "")
	if result.NativeSessionID != "thread-1" || result.Message != "done" || result.RawOutput != string(raw) || result.Model != "gpt-5" || result.TotalTokens != 150 || result.CachedInputTokens != 80 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseClaudeOutput(t *testing.T) {
	raw := []byte(`{"result":"finished","session_id":"session-1","model":"claude-sonnet","usage":{"input_tokens":20,"output_tokens":5,"cache_read_input_tokens":7}}`)
	result := parseCodeExecutorOutput("claude", raw, "")
	if result.NativeSessionID != "session-1" || result.Message != "finished" || result.Model != "claude-sonnet" || result.TotalTokens != 25 || result.CachedInputTokens != 7 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseOpenCodeOutput(t *testing.T) {
	raw := []byte("{\"type\":\"text\",\"sessionID\":\"session-2\",\"part\":{\"text\":\"hello \"}}\n" +
		"{\"type\":\"step_finish\",\"sessionID\":\"session-2\",\"part\":{\"tokens\":{\"input\":12,\"output\":4,\"reasoning\":2,\"cache\":{\"read\":3}}}}\n" +
		"{\"type\":\"text\",\"sessionID\":\"session-2\",\"part\":{\"text\":\"world\"}}\n")
	result := parseCodeExecutorOutput("opencode", raw, "")
	if result.NativeSessionID != "session-2" || result.Message != "hello world" || result.TotalTokens != 16 || result.ReasoningTokens != 2 || result.CachedInputTokens != 3 {
		t.Fatalf("unexpected result: %#v", result)
	}
}
