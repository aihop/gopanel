package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func TestNativeGrokRuntimeAndHistory(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	session := &model.AIDevSession{ID: 7, LastTaskID: 9, NativeSessionID: "grok-session"}
	sessionDir := filepath.Join(homeDir, ".grok", "sessions", "%2Fworkspace", session.NativeSessionID)
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "summary.json"), []byte(`{"current_model_id":"grok-4.6"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	updates := "{\"timestamp\":1787146000000,\"method\":\"session/update\",\"params\":{\"sessionId\":\"grok-session\",\"update\":{\"sessionUpdate\":\"user_message_chunk\",\"content\":{\"text\":\"fix it\"}}}}\n" +
		"{\"timestamp\":1787146001000,\"method\":\"session/update\",\"params\":{\"sessionId\":\"grok-session\",\"update\":{\"sessionUpdate\":\"agent_message_chunk\",\"content\":{\"text\":\"done\"}}}}\n" +
		"{\"timestamp\":1787146002000,\"method\":\"_x.ai/session/update\",\"params\":{\"sessionId\":\"grok-session\",\"update\":{\"sessionUpdate\":\"turn_completed\",\"stop_reason\":\"end_turn\",\"usage\":{\"inputTokens\":20,\"outputTokens\":5,\"cachedReadTokens\":3,\"reasoningTokens\":2,\"totalTokens\":25}}}}\n"
	if err := os.WriteFile(filepath.Join(sessionDir, "updates.jsonl"), []byte(updates), 0o600); err != nil {
		t.Fatal(err)
	}
	state := getNativeGrokRuntimeState(session)
	if state.ResponseState != "completed" || state.Model != "grok-4.6" || state.TotalTokens != 25 || state.LastAssistantPreview != "done" || state.UpdatedAt != time.UnixMilli(1787146002000) {
		t.Fatalf("unexpected Grok runtime: %#v", state)
	}
	messages, err := getNativeGrokMessages(session)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[0].Content != "fix it" || messages[1].Role != "agent" || messages[1].Content != "done" {
		t.Fatalf("unexpected Grok history: %#v", messages)
	}
}
