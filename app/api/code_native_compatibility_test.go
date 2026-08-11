package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestParseNativeClaudeMessagesFiltersToolResults(t *testing.T) {
	transcript := strings.Join([]string{
		`{"type":"user","uuid":"user-1","timestamp":"2026-08-11T08:00:00Z","origin":{"kind":"human"},"message":{"role":"user","content":"帮我修复任务"}}`,
		`{"type":"assistant","uuid":"assistant-1","timestamp":"2026-08-11T08:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"正在处理"},{"type":"tool_use","id":"tool-1","name":"Bash"}]}}`,
		`{"type":"user","uuid":"tool-result","timestamp":"2026-08-11T08:00:02Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-1"}]}}`,
		`{"type":"assistant","uuid":"assistant-2","timestamp":"2026-08-11T08:00:03Z","message":{"role":"assistant","content":[{"type":"text","text":"已经完成"}],"stop_reason":"end_turn"}}`,
	}, "\n")
	messages, err := parseNativeClaudeMessages(strings.NewReader(transcript), 7, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Fatalf("messages = %#v", messages)
	}
	if messages[0].Role != "user" || messages[0].Content != "帮我修复任务" || messages[1].Role != "agent" || messages[2].Content != "已经完成" {
		t.Fatalf("unexpected Claude history: %#v", messages)
	}
}

func TestParseNativeClaudeRuntimeTracksStateAndDeduplicatesUsage(t *testing.T) {
	transcript := strings.Join([]string{
		`{"type":"user","uuid":"user-1","timestamp":"2026-08-11T08:00:00Z","origin":{"kind":"human"},"message":{"role":"user","content":"开始"}}`,
		`{"type":"assistant","uuid":"assistant-1","timestamp":"2026-08-11T08:00:01Z","message":{"id":"message-1","role":"assistant","model":"claude-test","stop_reason":"tool_use","usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":2,"cache_creation_input_tokens":3},"content":[{"type":"text","text":"执行中"}]}}`,
		`{"type":"assistant","uuid":"assistant-2","timestamp":"2026-08-11T08:00:02Z","message":{"id":"message-1","role":"assistant","model":"claude-test","stop_reason":"tool_use","usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":2,"cache_creation_input_tokens":3},"content":[{"type":"tool_use","id":"tool-1","name":"Bash"}]}}`,
		`{"type":"user","uuid":"tool-result","timestamp":"2026-08-11T08:00:03Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-1"}]}}`,
		`{"type":"assistant","uuid":"assistant-3","timestamp":"2026-08-11T08:00:04Z","message":{"id":"message-2","role":"assistant","model":"claude-test","stop_reason":"end_turn","usage":{"input_tokens":4,"output_tokens":6,"cache_read_input_tokens":1},"content":[{"type":"text","text":"完成"}]}}`,
	}, "\n")
	state, err := parseNativeClaudeRuntime(strings.NewReader(transcript), time.Date(2026, 8, 11, 8, 0, 10, 0, time.UTC), true, codeApprovalPolicyManual)
	if err != nil {
		t.Fatal(err)
	}
	if state.ResponseState != "completed" || state.Model != "claude-test" || state.LastAssistantPreview != "完成" {
		t.Fatalf("state = %#v", state)
	}
	if state.InputTokens != 17 || state.OutputTokens != 11 || state.CachedInputTokens != 3 || state.TotalTokens != 28 {
		t.Fatalf("usage was duplicated: %#v", state)
	}
}

func TestParseNativeClaudeRuntimeDetectsApprovalWait(t *testing.T) {
	transcript := strings.Join([]string{
		`{"type":"user","uuid":"user-1","timestamp":"2026-08-11T08:00:00Z","origin":{"kind":"human"},"message":{"role":"user","content":"开始"}}`,
		`{"type":"assistant","uuid":"assistant-1","timestamp":"2026-08-11T08:00:01Z","message":{"id":"message-1","role":"assistant","stop_reason":"tool_use","content":[{"type":"tool_use","id":"tool-1","name":"Bash"}]}}`,
	}, "\n")
	state, err := parseNativeClaudeRuntime(strings.NewReader(transcript), time.Date(2026, 8, 11, 8, 0, 10, 0, time.UTC), true, codeApprovalPolicyManual)
	if err != nil {
		t.Fatal(err)
	}
	if state.ResponseState != "needsInput" || !state.AwaitingApproval || !state.NeedsInput {
		t.Fatalf("manual approval state = %#v", state)
	}
	fullAuto, err := parseNativeClaudeRuntime(strings.NewReader(transcript), time.Date(2026, 8, 11, 8, 0, 10, 0, time.UTC), true, codeApprovalPolicyFullAuto)
	if err != nil || fullAuto.ResponseState != "responding" || fullAuto.AwaitingApproval {
		t.Fatalf("full auto state = %#v, %v", fullAuto, err)
	}
}

func TestNativeOpenCodeSQLiteHistoryRuntimeAndBinding(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	databasePath := filepath.Join(homeDir, ".local", "share", "opencode", "opencode.db")
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		t.Fatal(err)
	}
	database, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE session (id TEXT PRIMARY KEY, directory TEXT, time_created INTEGER, time_updated INTEGER)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT, time_created INTEGER, time_updated INTEGER, data TEXT)`,
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT, session_id TEXT, time_created INTEGER, time_updated INTEGER, data TEXT)`,
	} {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	startedAt := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	workDir := t.TempDir()
	if err := database.Exec(`INSERT INTO session VALUES (?, ?, ?, ?)`, "session-new", workDir, startedAt.UnixMilli(), startedAt.Add(5*time.Second).UnixMilli()).Error; err != nil {
		t.Fatal(err)
	}
	userData := `{"role":"user","time":{"created":1786435200000}}`
	agentData := `{"role":"assistant","modelID":"opencode-test","providerID":"test","finish":"stop","time":{"created":1786435201000,"completed":1786435203000},"tokens":{"total":17,"input":10,"output":5,"reasoning":2,"cache":{"read":3,"write":0}}}`
	for _, values := range [][]any{
		{"message-user", "session-new", startedAt.UnixMilli(), startedAt.UnixMilli(), userData},
		{"message-agent", "session-new", startedAt.Add(time.Second).UnixMilli(), startedAt.Add(3 * time.Second).UnixMilli(), agentData},
	} {
		if err := database.Exec(`INSERT INTO message VALUES (?, ?, ?, ?, ?)`, values...).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, values := range [][]any{
		{"part-user", "message-user", "session-new", startedAt.UnixMilli(), startedAt.UnixMilli(), `{"type":"text","text":"检查项目"}`},
		{"part-reasoning", "message-agent", "session-new", startedAt.Add(time.Second).UnixMilli(), startedAt.Add(time.Second).UnixMilli(), `{"type":"reasoning","text":"内部推理"}`},
		{"part-agent", "message-agent", "session-new", startedAt.Add(2 * time.Second).UnixMilli(), startedAt.Add(2 * time.Second).UnixMilli(), `{"type":"text","text":"检查完成"}`},
	} {
		if err := database.Exec(`INSERT INTO part VALUES (?, ?, ?, ?, ?, ?)`, values...).Error; err != nil {
			t.Fatal(err)
		}
	}
	session := &model.AIDevSession{ID: 7, LastTaskID: 9, NativeSessionID: "session-new"}
	messages, err := getNativeOpenCodeMessages(session)
	if err != nil || len(messages) != 2 || messages[0].Content != "检查项目" || messages[1].Content != "检查完成" {
		t.Fatalf("messages = %#v, err = %v", messages, err)
	}
	nativeMessages, err := readNativeOpenCodeMessages("session-new")
	if err != nil {
		t.Fatal(err)
	}
	state := parseNativeOpenCodeRuntime(nativeMessages, true)
	if state.ResponseState != "completed" || state.Model != "opencode-test" || state.TotalTokens != 17 || state.LastAssistantPreview != "检查完成" {
		t.Fatalf("state = %#v", state)
	}
	nativeSessionID, err := findNativeOpenCodeSessionInDatabase(workDir, startedAt)
	if err != nil || nativeSessionID != "session-new" {
		t.Fatalf("binding = %q, err = %v", nativeSessionID, err)
	}
}

func TestOpenCodeConfigHasProvider(t *testing.T) {
	provider, _ := json.Marshal(map[string]any{"provider": map[string]any{"custom": map[string]any{"models": map[string]any{"test": map[string]any{}}}}})
	if !openCodeConfigHasProvider(provider) || openCodeConfigHasProvider([]byte(`{"provider":{}}`)) || openCodeConfigHasProvider([]byte(`invalid`)) {
		t.Fatal("OpenCode provider configuration was not detected")
	}
}
