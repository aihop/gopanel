package api

import "testing"

func TestConversationAssistantUpdate(t *testing.T) {
	delta, replace := conversationAssistantUpdate("grok", map[string]any{"type": "text", "data": "hello"})
	if delta != "hello" || replace {
		t.Fatalf("grok delta = %q replace=%v", delta, replace)
	}
	delta, replace = conversationAssistantUpdate("grok", map[string]any{
		"method": "session/update",
		"params": map[string]any{
			"update": map[string]any{
				"sessionUpdate": "agent_message_chunk",
				"content":       map[string]any{"type": "text", "text": "先改这里"},
			},
		},
	})
	if delta != "先改这里" || replace {
		t.Fatalf("grok acp delta = %q replace=%v", delta, replace)
	}
	delta, replace = conversationAssistantUpdate("claude", map[string]any{
		"type":  "content_block_delta",
		"delta": map[string]any{"type": "text_delta", "text": "ok"},
	})
	if delta != "ok" || replace {
		t.Fatalf("claude delta = %q replace=%v", delta, replace)
	}
	text, replace := conversationAssistantUpdate("codex", map[string]any{
		"type": "item.completed",
		"item": map[string]any{"type": "agent_message", "text": "done"},
	})
	if text != "done" || !replace {
		t.Fatalf("codex snapshot = %q replace=%v", text, replace)
	}
}

func TestParseCodexOutput(t *testing.T) {
	raw := []byte("{\"type\":\"thread.started\",\"thread_id\":\"thread-1\"}\n" +
		"{\"type\":\"turn.completed\",\"model\":\"gpt-5\",\"usage\":{\"input_tokens\":120,\"cached_input_tokens\":80,\"output_tokens\":30,\"reasoning_output_tokens\":10}}\n" +
		"{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"done\"}}\n")
	result := parseCodeExecutorOutput("codex", raw, "")
	if result.NativeSessionID != "thread-1" || result.Message != "done" || result.RawOutput != string(raw) || result.Model != "gpt-5" || result.TotalTokens != 150 || result.CachedInputTokens != 80 || !result.TokenUsageReported {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseGrokOutput(t *testing.T) {
	raw := []byte("{\"type\":\"text\",\"data\":\"hello \"}\n" +
		"{\"type\":\"thought\",\"data\":\"working\"}\n" +
		"{\"type\":\"text\",\"data\":\"world\"}\n" +
		"{\"type\":\"end\",\"sessionId\":\"grok-session\"}\n")
	result := parseCodeExecutorOutput("grok", raw, "prepared-session")
	if result.NativeSessionID != "grok-session" || result.Message != "hello world" || result.RawOutput != string(raw) {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseGrokACPOutput(t *testing.T) {
	raw := []byte("{\"method\":\"session/update\",\"params\":{\"sessionId\":\"grok-session\",\"update\":{\"sessionUpdate\":\"agent_thought_chunk\",\"content\":{\"text\":\"thinking\"}}}}\n" +
		"{\"method\":\"session/update\",\"params\":{\"sessionId\":\"grok-session\",\"update\":{\"sessionUpdate\":\"agent_message_chunk\",\"content\":{\"type\":\"text\",\"text\":\"hello \"}}}}\n" +
		"{\"method\":\"session/update\",\"params\":{\"sessionId\":\"grok-session\",\"update\":{\"sessionUpdate\":\"agent_message_chunk\",\"content\":{\"type\":\"text\",\"text\":\"world\"}}}}\n")
	result := parseCodeExecutorOutput("grok", raw, "prepared-session")
	if result.NativeSessionID != "grok-session" || result.Message != "hello world" {
		t.Fatalf("unexpected grok acp result: %#v", result)
	}
}

func TestParseClaudeOutput(t *testing.T) {
	raw := []byte(`{"result":"finished","session_id":"session-1","model":"claude-sonnet","usage":{"input_tokens":20,"output_tokens":5,"cache_read_input_tokens":7}}`)
	result := parseCodeExecutorOutput("claude", raw, "")
	if result.NativeSessionID != "session-1" || result.Message != "finished" || result.Model != "claude-sonnet" || result.TotalTokens != 25 || result.CachedInputTokens != 7 || !result.TokenUsageReported {
		t.Fatalf("unexpected result: %#v", result)
	}
	stream := []byte("{\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"fin\"}}\n" +
		"{\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"ished\"}}\n" +
		"{\"type\":\"result\",\"result\":\"finished\",\"session_id\":\"session-2\",\"model\":\"claude-sonnet\",\"usage\":{\"input_tokens\":20,\"output_tokens\":5}}\n")
	streamed := parseCodeExecutorOutput("claude", stream, "")
	if streamed.NativeSessionID != "session-2" || streamed.Message != "finished" || streamed.Model != "claude-sonnet" {
		t.Fatalf("unexpected claude stream result: %#v", streamed)
	}
}

func TestParseOpenCodeOutput(t *testing.T) {
	raw := []byte("{\"type\":\"text\",\"sessionID\":\"session-2\",\"part\":{\"text\":\"hello \"}}\n" +
		"{\"type\":\"step_finish\",\"sessionID\":\"session-2\",\"part\":{\"tokens\":{\"input\":12,\"output\":4,\"reasoning\":2,\"cache\":{\"read\":3}}}}\n" +
		"{\"type\":\"text\",\"sessionID\":\"session-2\",\"part\":{\"text\":\"world\"}}\n")
	result := parseCodeExecutorOutput("opencode", raw, "")
	if result.NativeSessionID != "session-2" || result.Message != "hello world" || result.TotalTokens != 16 || result.ReasoningTokens != 2 || result.CachedInputTokens != 3 || !result.TokenUsageReported {
		t.Fatalf("unexpected result: %#v", result)
	}
}
