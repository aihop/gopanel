package api

import (
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func TestParseNativeCodexMessagesFiltersInternalEvents(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"2026-07-29T10:00:00Z","type":"response_item","payload":{"type":"message","role":"developer","content":[]}}`,
		`{"timestamp":"2026-07-29T10:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"检查项目"}}`,
		`{"timestamp":"2026-07-29T10:00:02Z","type":"event_msg","payload":{"type":"token_count","message":"secret"}}`,
		`{"timestamp":"2026-07-29T10:00:03Z","type":"event_msg","payload":{"type":"agent_message","message":"正在检查","phase":"commentary"}}`,
	}, "\n")
	messages, err := parseNativeCodexMessages(strings.NewReader(input), 7, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[1].Role != "agent" {
		t.Fatalf("unexpected messages: %#v", messages)
	}
	if messages[0].Content != "检查项目" || messages[1].Content != "正在检查" {
		t.Fatalf("unexpected content: %#v", messages)
	}
	if messages[0].SessionID != 7 || messages[0].TaskID != 9 {
		t.Fatalf("unexpected links: %#v", messages[0])
	}
}

func TestGetNativeCodexMessagesRequiresBoundNativeSession(t *testing.T) {
	messages, err := getNativeCodexMessages(&model.AIDevSession{WorkDir: t.TempDir()})
	if err != nil || len(messages) != 0 {
		t.Fatalf("unbound session should not guess native history: %#v, %v", messages, err)
	}
}

func TestMergeCodeHistoryMessagesDeduplicatesNearbyDatabaseMessage(t *testing.T) {
	now := time.Now()
	databaseMessages := []*model.AIMessage{{ID: 4, CreatedAt: now, Role: "user", Content: "继续"}}
	nativeMessages := []*model.AIMessage{
		{CreatedAt: now.Add(time.Second), Role: "user", Content: "继续"},
		{CreatedAt: now.Add(2 * time.Second), Role: "agent", Content: "已完成"},
		{CreatedAt: now.Add(10 * time.Minute), Role: "user", Content: "继续"},
	}
	merged := mergeCodeHistoryMessages(databaseMessages, nativeMessages)
	if len(merged) != 3 {
		t.Fatalf("unexpected merged messages: %#v", merged)
	}
	if merged[1].Content != "已完成" || merged[1].ID == 0 || merged[2].ID == 0 {
		t.Fatalf("unexpected native messages: %#v", merged)
	}
}
