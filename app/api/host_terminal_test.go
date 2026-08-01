package api

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/aihop/gopanel/app/model"
)

type fakeHostTerminalPTY struct {
	writes []byte
	cols   uint16
	rows   uint16
	closed bool
}

func (terminal *fakeHostTerminalPTY) Read([]byte) (int, error) { return 0, io.EOF }

func (terminal *fakeHostTerminalPTY) Write(data []byte) (int, error) {
	terminal.writes = append(terminal.writes, data...)
	return len(data), nil
}

func (terminal *fakeHostTerminalPTY) Resize(cols, rows uint16) error {
	terminal.cols, terminal.rows = cols, rows
	return nil
}

func (terminal *fakeHostTerminalPTY) Close() error {
	terminal.closed = true
	return nil
}

func TestResolveHostTerminalWorkDir(t *testing.T) {
	dir := t.TempDir()
	resolved, err := resolveHostTerminalWorkDir(dir)
	expected, resolveErr := filepath.EvalSymlinks(filepath.Clean(dir))
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	if err != nil || resolved != expected {
		t.Fatalf("unexpected work directory: %q, %v", resolved, err)
	}
	file, err := os.CreateTemp(dir, "file")
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	if _, err := resolveHostTerminalWorkDir(file.Name()); err == nil {
		t.Fatal("regular file should not be accepted as terminal work directory")
	}
}

func TestBuildHostTerminalCommandRejectsUnknownShell(t *testing.T) {
	if _, _, err := buildHostTerminalCommand("unknown-shell", t.TempDir()); err == nil {
		t.Fatal("unknown shell should be rejected")
	}
}

func TestHostTerminalControlAndHistory(t *testing.T) {
	pty := &fakeHostTerminalPTY{}
	session := &hostTerminal{
		record: &model.HostTerminalSession{ID: 1, UserID: 7}, pty: pty,
		subscribers: make(map[string]*hostTerminalSubscription), done: make(chan struct{}),
	}
	first, baseline := session.subscribe(7, "127.0.0.1", false)
	defer session.unsubscribe(first)
	if !baseline.HasControl {
		t.Fatal("first writable subscriber should receive control")
	}
	second, secondBaseline := session.subscribe(7, "127.0.0.2", false)
	defer session.unsubscribe(second)
	if secondBaseline.HasControl {
		t.Fatal("second subscriber should initially be read-only")
	}
	if granted, _ := session.takeControl(second.ID); granted {
		t.Fatal("control takeover should be denied while lease is active")
	}
	if err := session.write(first.ID, []byte("echo ok\r")); err != nil || string(pty.writes) != "echo ok\r" {
		t.Fatalf("unexpected terminal write: %q, %v", pty.writes, err)
	}
	if err := session.resize(first.ID, 100, 40); err != nil || pty.cols != 100 || pty.rows != 40 {
		t.Fatalf("unexpected resize: %dx%d, %v", pty.cols, pty.rows, err)
	}
	session.publish([]byte("output"))
	if session.sequence != 1 || string(session.history) != "output" {
		t.Fatalf("unexpected history: sequence=%d history=%q", session.sequence, session.history)
	}
	if !session.releaseControl(first.ID) {
		t.Fatal("controller should be able to release control")
	}
	if granted, reason := session.takeControl(second.ID); !granted || reason != "" {
		t.Fatalf("second subscriber should take released control: %v %q", granted, reason)
	}
	if err := session.write(first.ID, []byte("denied")); err == nil {
		t.Fatal("former controller should not retain input access")
	}
}

func TestHostTerminalHistoryIsCapped(t *testing.T) {
	session := &hostTerminal{record: &model.HostTerminalSession{}, subscribers: make(map[string]*hostTerminalSubscription)}
	data := make([]byte, hostTerminalHistoryLimit+128)
	for index := range data {
		data[index] = 'x'
	}
	session.publish(data)
	if len(session.history) != hostTerminalHistoryLimit || !session.historyTruncated {
		t.Fatalf("history cap not applied: size=%d truncated=%v", len(session.history), session.historyTruncated)
	}
}

func TestHostTerminalSplitsLargeBaseline(t *testing.T) {
	event := hostTerminalEvent{Type: "baseline", Data: string(make([]byte, hostTerminalBaselineChunkLimit+1)), Truncated: true}
	chunks := splitHostTerminalBaseline(event)
	if len(chunks) != 2 || len(chunks[0].Data) != hostTerminalBaselineChunkLimit || len(chunks[1].Data) != 1 {
		t.Fatalf("unexpected baseline chunks: %#v", chunks)
	}
	if !chunks[0].Truncated || chunks[1].Truncated || chunks[0].ChunkIndex != 0 || chunks[1].ChunkIndex != 1 || chunks[1].ChunkCount != 2 {
		t.Fatalf("unexpected chunk metadata: %#v", chunks)
	}
}

func TestHostTerminalBaselineChunksPreserveUTF8(t *testing.T) {
	data := strings.Repeat("x", hostTerminalBaselineChunkLimit-1) + "中文"
	chunks := splitHostTerminalBaseline(hostTerminalEvent{Type: "baseline", Data: data})
	var combined strings.Builder
	for _, chunk := range chunks {
		if !utf8.ValidString(chunk.Data) {
			t.Fatalf("invalid UTF-8 chunk: %q", chunk.Data)
		}
		combined.WriteString(chunk.Data)
		if chunk.ChunkCount != len(chunks) {
			t.Fatalf("unexpected chunk count: %#v", chunks)
		}
	}
	if combined.String() != data {
		t.Fatal("baseline data changed during UTF-8 chunking")
	}
}

func TestHostTerminalControlLeaseExpires(t *testing.T) {
	session := &hostTerminal{record: &model.HostTerminalSession{}, pty: &fakeHostTerminalPTY{}, subscribers: make(map[string]*hostTerminalSubscription)}
	subscriber, _ := session.subscribe(1, "127.0.0.1", false)
	defer session.unsubscribe(subscriber)
	session.mu.Lock()
	session.controlExpiresAt = time.Now().Add(-time.Second)
	session.mu.Unlock()
	if err := session.write(subscriber.ID, []byte("denied")); err == nil {
		t.Fatal("expired controller should not retain input access")
	}
}

func TestHostTerminalResumeReturnsSameSession(t *testing.T) {
	record := &model.HostTerminalSession{ID: 8, UserID: 1, Status: "running"}
	manager := &hostTerminalManager{sessions: map[uint]*hostTerminal{record.ID: {record: record}}}
	resumed, err := manager.resume(record.ID)
	if err != nil || resumed.ID != record.ID {
		t.Fatalf("existing terminal should resume with the same id: %#v, %v", resumed, err)
	}
	if _, err := manager.resume(99); err == nil {
		t.Fatal("missing terminal process should not be recreated")
	}
}
