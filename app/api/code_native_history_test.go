package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
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

// 只认旧格式会让升级后的会话历史整段读不出来。
func TestParseNativeCodexMessagesReadsResponseItemFormat(t *testing.T) {
	rollout := strings.Join([]string{
		`{"timestamp":"2026-08-10T07:17:05.221Z","type":"session_meta","payload":{"id":"s1"}}`,
		`{"timestamp":"2026-08-10T07:17:06.000Z","type":"response_item","payload":{"type":"message","id":"msg_1","role":"developer","content":[{"type":"text","text":"注入的系统提示"}]}}`,
		`{"timestamp":"2026-08-10T07:17:07.000Z","type":"response_item","payload":{"type":"message","id":"msg_2","role":"user","content":[{"type":"text","text":"帮我改一下交付"}]}}`,
		`{"timestamp":"2026-08-10T07:17:08.000Z","type":"response_item","payload":{"type":"message","id":"msg_3","role":"assistant","content":[{"type":"text","text":"已经改好了"}]}}`,
		`{"timestamp":"2026-08-10T07:17:09.000Z","type":"response_item","payload":{"type":"reasoning","id":"r1"}}`,
		`{"timestamp":"2026-08-10T07:17:10.000Z","type":"event_msg","payload":{"type":"token_count"}}`,
	}, "\n")

	messages, err := parseNativeCodexMessages(strings.NewReader(rollout), 7, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected the user and assistant messages only, got %d", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "帮我改一下交付" {
		t.Fatalf("unexpected user message: %#v", messages[0])
	}
	if messages[1].Role != "agent" || messages[1].Content != "已经改好了" {
		t.Fatalf("unexpected agent message: %#v", messages[1])
	}
	if messages[0].SessionID != 7 || messages[0].TaskID != 9 {
		t.Fatalf("message binding lost: %#v", messages[0])
	}
	if messages[0].CreatedAt.IsZero() {
		t.Fatal("timestamp was not parsed")
	}
}

// 旧版 rollout 仍然要能读出来，否则历史会话会反过来失联。
func TestParseNativeCodexMessagesStillReadsLegacyEventFormat(t *testing.T) {
	rollout := strings.Join([]string{
		`{"timestamp":"2026-07-27T22:55:27.000Z","type":"event_msg","payload":{"type":"user_message","message":"老格式的提问"}}`,
		`{"timestamp":"2026-07-27T22:55:40.000Z","type":"event_msg","payload":{"type":"agent_message","message":"老格式的回答"}}`,
		`{"timestamp":"2026-07-27T22:55:41.000Z","type":"event_msg","payload":{"type":"task_complete"}}`,
	}, "\n")

	messages, err := parseNativeCodexMessages(strings.NewReader(rollout), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[1].Role != "agent" {
		t.Fatalf("legacy rollout parsing regressed: %#v", messages)
	}
	if messages[0].Content != "老格式的提问" || messages[1].Content != "老格式的回答" {
		t.Fatalf("unexpected legacy content: %#v", messages)
	}
}

// 同一个会话跨过 codex 升级时，一个文件里可能新旧事件并存。
func TestParseNativeCodexMessagesHandlesMixedFormats(t *testing.T) {
	rollout := strings.Join([]string{
		`{"timestamp":"2026-08-01T10:00:00.000Z","type":"event_msg","payload":{"type":"user_message","message":"旧提问"}}`,
		`{"timestamp":"2026-08-01T10:00:05.000Z","type":"response_item","payload":{"type":"message","id":"m1","role":"assistant","content":[{"type":"text","text":"新回答"}]}}`,
	}, "\n")

	messages, err := parseNativeCodexMessages(strings.NewReader(rollout), 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Content != "旧提问" || messages[1].Content != "新回答" {
		t.Fatalf("mixed rollout parsing failed: %#v", messages)
	}
}

// 多段 content 要拼成一条完整正文，空片段不产生空消息。
func TestNativeCodexContentTextJoinsSegments(t *testing.T) {
	if text := nativeCodexContentText([]byte(`[{"type":"text","text":"第一段"},{"type":"text","text":"  "},{"type":"text","text":"第二段"}]`)); text != "第一段\n第二段" {
		t.Fatalf("unexpected joined content: %q", text)
	}
	if text := nativeCodexContentText([]byte(`[]`)); text != "" {
		t.Fatalf("empty content should stay empty: %q", text)
	}
	if text := nativeCodexContentText([]byte(`"not-an-array"`)); text != "" {
		t.Fatalf("malformed content should stay empty: %q", text)
	}
}

// 原生历史必须能增量固化进库，且反复读取不产生重复。
func TestPersistNativeCodexMessagesIsIdempotent(t *testing.T) {
	database := withCodeGovernanceDB(t)
	rollout := strings.Join([]string{
		`{"timestamp":"2026-08-10T07:17:07.000Z","type":"response_item","payload":{"type":"message","id":"msg_a","role":"user","content":[{"type":"text","text":"第一句"}]}}`,
		`{"timestamp":"2026-08-10T07:17:08.000Z","type":"response_item","payload":{"type":"message","id":"msg_b","role":"assistant","content":[{"type":"text","text":"第一答"}]}}`,
	}, "\n")

	first, err := parseNativeCodexMessages(strings.NewReader(rollout), 51, 61)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistNativeCodexMessages(51, first); err != nil {
		t.Fatal(err)
	}
	// 再读一次同一个文件，不应重复入库。
	second, err := parseNativeCodexMessages(strings.NewReader(rollout), 51, 61)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistNativeCodexMessages(51, second); err != nil {
		t.Fatal(err)
	}

	var stored []model.AIMessage
	if err := database.Where("session_id = ?", 51).Order("created_at asc").Find(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 {
		t.Fatalf("expected exactly 2 persisted messages, got %d", len(stored))
	}
	if stored[0].Content != "第一句" || stored[1].Content != "第一答" || stored[0].NativeID != "msg_a" {
		t.Fatalf("unexpected persisted rows: %#v", stored)
	}

	// 新增一条后只应插入增量。
	extended := rollout + "\n" + `{"timestamp":"2026-08-10T07:17:09.000Z","type":"response_item","payload":{"type":"message","id":"msg_c","role":"user","content":[{"type":"text","text":"第二句"}]}}`
	third, err := parseNativeCodexMessages(strings.NewReader(extended), 51, 61)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistNativeCodexMessages(51, third); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.Model(&model.AIMessage{}).Where("session_id = ?", 51).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("incremental persist produced %d rows, want 3", count)
	}
}

// 旧格式没有 payload.id，兜底 ID 也必须稳定，否则每次读取都会重复入库。
func TestNativeCodexMessageIDIsStableWithoutPayloadID(t *testing.T) {
	first := nativeCodexMessageID("", "user", "2026-07-27T22:55:27.000Z", "老格式的提问")
	second := nativeCodexMessageID("", "user", "2026-07-27T22:55:27.000Z", "老格式的提问")
	if first == "" || first != second {
		t.Fatalf("fallback id must be stable: %q vs %q", first, second)
	}
	if other := nativeCodexMessageID("", "agent", "2026-07-27T22:55:27.000Z", "老格式的提问"); other == first {
		t.Fatal("different roles must not collide")
	}
	if explicit := nativeCodexMessageID("msg_x", "user", "t", "c"); explicit != "msg_x" {
		t.Fatalf("payload id should win: %q", explicit)
	}
}

// 交付会把 session.WorkDir 改写成源仓路径，rollout 里记的却是当初的隔离 Worktree。
// 绑定修复必须能按隔离目录回溯，否则交付过的会话历史永久失联。
func TestRepairNativeCodexSessionBindingFallsBackToWorktreeDir(t *testing.T) {
	database := withCodeGovernanceDB(t)
	withAIProjectBaseDir(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	startedAt := time.Now().Add(-time.Second)
	session := &model.AIDevSession{UserID: 5, Title: "s", WorkDir: t.TempDir(), CreatedAt: startedAt}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	// 交付后 WorkDir 已经指向源仓，rollout 里记录的是隔离 Worktree 目录。
	worktreeDir := aiSessionWorktreeDir(session.UserID, session.ID)
	if err := os.MkdirAll(worktreeDir, 0o750); err != nil {
		t.Fatal(err)
	}
	sessionDir := filepath.Join(homeDir, ".codex", "sessions", "2026", "08", "10")
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]any{
		"timestamp": startedAt.Add(time.Minute),
		"type":      "session_meta",
		"payload":   map[string]any{"session_id": "native-xyz", "cwd": worktreeDir, "timestamp": startedAt},
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sessionDir, "rollout-native-xyz.jsonl")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	delayedWrite := startedAt.Add(time.Minute)
	if err := os.Chtimes(path, delayedWrite, delayedWrite); err != nil {
		t.Fatal(err)
	}

	if err := repairNativeCodexSessionBinding(session); err != nil {
		t.Fatal(err)
	}
	if session.NativeSessionID != "native-xyz" {
		t.Fatalf("binding was not recovered from the worktree dir, got %q", session.NativeSessionID)
	}
}

// 同一会话下存在多个任务时，历史必须按任务隔离，不能把别的任务的对话混进来。
func TestGetMessagesBySessionAndTaskIDIsolatesTasks(t *testing.T) {
	database := withCodeGovernanceDB(t)
	rows := []*model.AIMessage{
		{SessionID: 90, TaskID: 1, Role: "user", Content: "任务一的提问"},
		{SessionID: 90, TaskID: 1, Role: "agent", Content: "任务一的回答"},
		{SessionID: 90, TaskID: 2, Role: "user", Content: "任务二的提问"},
		{SessionID: 91, TaskID: 3, Role: "user", Content: "别的会话"},
	}
	if err := database.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	taskRepo := repo.NewAITaskRepo()

	scoped, err := taskRepo.GetMessagesBySessionAndTaskID(90, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(scoped) != 2 || scoped[0].Content != "任务一的提问" {
		t.Fatalf("task scoped history leaked other tasks: %#v", scoped)
	}

	// 不指定任务时仍返回整个会话，保持既有行为。
	whole, err := taskRepo.GetMessagesBySessionID(90)
	if err != nil {
		t.Fatal(err)
	}
	if len(whole) != 3 {
		t.Fatalf("session scoped history changed: %d", len(whole))
	}

	// 任务号相同但会话不同的消息不能被读到。
	crossed, err := taskRepo.GetMessagesBySessionAndTaskID(90, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(crossed) != 0 {
		t.Fatalf("history crossed the session boundary: %#v", crossed)
	}
}
