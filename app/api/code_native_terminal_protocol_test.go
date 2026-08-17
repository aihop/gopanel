package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/aihop/gopanel/app/model"
)

func newNativeTerminalProtocolTestSubject() *nativeCodeTerminal {
	return &nativeCodeTerminal{subscribers: make(map[string]*nativeTerminalSubscription)}
}

func TestNativeTerminalAttachOnlyDoesNotStartInactiveSession(t *testing.T) {
	manager := &nativeCodeTerminalManager{sessions: make(map[uint]*nativeCodeTerminal)}
	terminal, started, err := manager.connect(&model.AIDevSession{ID: 41}, 120, 40, true)
	if terminal != nil || started || !errors.Is(err, errNativeCodeTerminalInactive) {
		t.Fatalf("attach-only inactive result = %#v, %v, %v", terminal, started, err)
	}
	if len(manager.sessions) != 0 {
		t.Fatal("attach-only connection started a terminal")
	}
}

func TestNativeTerminalAttachOnlyReusesRunningSession(t *testing.T) {
	running := newNativeTerminalProtocolTestSubject()
	manager := &nativeCodeTerminalManager{sessions: map[uint]*nativeCodeTerminal{42: running}}
	terminal, started, err := manager.connect(&model.AIDevSession{ID: 42}, 120, 40, true)
	if err != nil || started || terminal != running {
		t.Fatalf("attach-only running result = %#v, %v, %v", terminal, started, err)
	}
}

func TestNativeTerminalConnectionErrorsAreStructured(t *testing.T) {
	for _, test := range []struct {
		err       error
		eventType string
		code      string
	}{
		{err: errNativeCodeTerminalInactive, eventType: "inactive", code: "terminal_inactive"},
		{err: errCodeExecutionBusy, eventType: "error", code: "workspace_busy"},
		{err: errors.New("boom"), eventType: "error", code: "start_failed"},
	} {
		var event struct {
			Type string `json:"type"`
			Code string `json:"code"`
		}
		if err := json.Unmarshal(nativeTerminalConnectionErrorPayload(test.err), &event); err != nil {
			t.Fatal(err)
		}
		if event.Type != test.eventType || event.Code != test.code {
			t.Fatalf("connection event = %#v, want %s/%s", event, test.eventType, test.code)
		}
	}
}

func TestNativeTerminalProtocolResyncsAfterSequenceGap(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.publish([]byte("one"))
	terminal.publish([]byte("two"))
	subscription, _ := terminal.subscribe(0, false)
	drainNativeTerminalEvents(subscription.Events)
	if !terminal.resync(subscription.ID, 1, "request-1") {
		t.Fatal("resync request was rejected")
	}
	baseline := <-subscription.Events
	if baseline.Type != "baseline" || baseline.RequestID != "request-1" || baseline.StartSequence != 2 || baseline.Sequence != 2 || string(baseline.Data) != "two" {
		t.Fatalf("unexpected resync baseline: %#v", baseline)
	}
	terminal.unsubscribe(subscription)
}

func TestNativeTerminalProtocolSignalsSubscriberOverflow(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	subscription, _ := terminal.subscribe(0, false)
	for index := 0; index < cap(subscription.Events); index++ {
		subscription.Events <- nativeTerminalEvent{Type: "output"}
	}
	terminal.publish([]byte("latest"))
	event := <-subscription.Events
	if event.Type != "resync_required" || !subscription.NeedsResync || event.Sequence != 1 {
		t.Fatalf("unexpected overflow event: %#v", event)
	}
	terminal.publish([]byte("paused"))
	select {
	case event := <-subscription.Events:
		t.Fatalf("output should pause until resync, got %#v", event)
	default:
	}
	terminal.unsubscribe(subscription)
}

func TestNativeTerminalProtocolMarksTruncatedBaseline(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.sequence = 5
	terminal.history = []nativeTerminalChunk{{Sequence: 4, Data: []byte("four")}, {Sequence: 5, Data: []byte("five")}}
	subscription, baseline := terminal.subscribe(1, false)
	if !baseline.Truncated || baseline.StartSequence != 4 || string(baseline.Data) != "fourfive" {
		t.Fatalf("unexpected truncated baseline: %#v", baseline)
	}
	terminal.unsubscribe(subscription)
}

func TestNativeTerminalProtocolCapsAcknowledgement(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.publish([]byte("one"))
	subscription, _ := terminal.subscribe(0, false)
	terminal.acknowledge(subscription.ID, 99)
	if subscription.AckSequence != 1 {
		t.Fatalf("acknowledgement exceeded server sequence: %d", subscription.AckSequence)
	}
	terminal.unsubscribe(subscription)
}

func TestNativeTerminalProtocolSplitsLargeBaseline(t *testing.T) {
	event := nativeTerminalEvent{Type: "baseline", Data: make([]byte, nativeTerminalBaselineChunkLimit+1)}
	chunks := splitNativeTerminalBaseline(event)
	if len(chunks) != 2 || len(chunks[0].Data) != nativeTerminalBaselineChunkLimit || len(chunks[1].Data) != 1 {
		t.Fatalf("unexpected baseline chunks: %d, %d, %d", len(chunks), len(chunks[0].Data), len(chunks[1].Data))
	}
}

func TestNativeTerminalBaselineChunksPreserveUTF8(t *testing.T) {
	data := append(bytes.Repeat([]byte{'x'}, nativeTerminalBaselineChunkLimit-1), []byte("中文")...)
	chunks := splitNativeTerminalBaseline(nativeTerminalEvent{Type: "baseline", Data: data})
	var combined []byte
	for _, chunk := range chunks {
		if !utf8.Valid(chunk.Data) {
			t.Fatalf("invalid UTF-8 chunk: %q", chunk.Data)
		}
		combined = append(combined, chunk.Data...)
	}
	if !bytes.Equal(combined, data) {
		t.Fatal("baseline data changed during UTF-8 chunking")
	}
}

func TestNativeTerminalProtocolReplaysOnlyMissingOutput(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.publish([]byte("one"))
	terminal.publish([]byte("two"))
	terminal.publish([]byte("three"))
	subscription, baseline := terminal.subscribe(1, false)
	defer terminal.unsubscribe(subscription)
	if baseline.Type != "baseline" || baseline.Sequence != 3 || string(baseline.Data) != "twothree" {
		t.Fatalf("unexpected baseline: %#v", baseline)
	}
	if !baseline.HasControl {
		t.Fatal("first subscriber should receive input control")
	}
	terminal.publish([]byte("four"))
	event := <-subscription.Events
	if event.Type != "output" || event.Sequence != 4 || string(event.Data) != "four" {
		t.Fatalf("unexpected output event: %#v", event)
	}
}

func TestNativeTerminalProtocolPreventsControlLeasePreemption(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	first, firstBaseline := terminal.subscribe(0, false)
	second, secondBaseline := terminal.subscribe(0, false)
	if !firstBaseline.HasControl || secondBaseline.HasControl {
		t.Fatalf("unexpected initial control: first=%v second=%v", firstBaseline.HasControl, secondBaseline.HasControl)
	}
	if granted, _ := terminal.takeControl(second.ID, 0, 0); granted {
		t.Fatal("second subscriber should not preempt active control lease")
	}
	if !terminal.releaseControl(first.ID) {
		t.Fatal("first subscriber failed to release control")
	}
	<-first.Events
	<-second.Events
	if granted, reason := terminal.takeControl(second.ID, 0, 0); !granted || reason != "" {
		t.Fatalf("second subscriber failed to acquire released control: %q", reason)
	}
	firstControl := <-first.Events
	secondControl := <-second.Events
	if firstControl.HasControl || !secondControl.HasControl {
		t.Fatalf("unexpected transferred control: first=%#v second=%#v", firstControl, secondControl)
	}
	terminal.unsubscribe(second)
	terminal.unsubscribe(first)
}

func TestNativeTerminalProtocolReadOnlySubscriberRequiresExplicitControl(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	subscription, baseline := terminal.subscribe(0, true)
	if baseline.HasControl || terminal.controllerID != "" {
		t.Fatal("read-only subscriber should not receive terminal control")
	}
	if granted, reason := terminal.takeControl(subscription.ID, 0, 0); granted || reason != "只读连接不能接管终端输入" {
		t.Fatalf("untrusted read-only subscriber took control: granted=%v reason=%q", granted, reason)
	}
	subscription.AllowControl = true
	if granted, reason := terminal.takeControl(subscription.ID, 0, 0); !granted || reason != "" {
		t.Fatalf("read-only subscriber failed to explicitly take control: granted=%v reason=%q", granted, reason)
	}
	if subscription.ReadOnly {
		t.Fatal("controlled subscriber should accept terminal input")
	}
	if !terminal.releaseControl(subscription.ID) || !subscription.ReadOnly {
		t.Fatal("released subscriber should return to its default read-only state")
	}
	<-subscription.Events
	<-subscription.Events
	terminal.unsubscribe(subscription)
}

func TestNativeTerminalProtocolExpiredControlRestoresReadOnlyState(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	subscription, _ := terminal.subscribe(0, true)
	subscription.AllowControl = true
	if granted, reason := terminal.takeControl(subscription.ID, 0, 0); !granted || reason != "" {
		t.Fatalf("take control failed: %q", reason)
	}
	terminal.mu.Lock()
	terminal.controlExpiresAt = time.Now().Add(-time.Second)
	terminal.mu.Unlock()
	terminal.expireControl(subscription.ID)
	if !subscription.ReadOnly || terminal.controllerID != "" {
		t.Fatal("expired control should restore the subscriber's read-only state")
	}
	terminal.unsubscribe(subscription)
}

func TestNativeTerminalProtocolAllowsExpiredLeaseTakeover(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	first, _ := terminal.subscribe(0, false)
	second, _ := terminal.subscribe(0, false)
	terminal.mu.Lock()
	terminal.controlExpiresAt = time.Now().Add(-time.Second)
	terminal.mu.Unlock()
	if granted, reason := terminal.takeControl(second.ID, 0, 0); !granted || reason != "" {
		t.Fatalf("expired lease should be acquirable: %q", reason)
	}
	terminal.unsubscribe(second)
	terminal.unsubscribe(first)
}

func TestNativeTerminalProtocolResetsStaleSequence(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.publish([]byte("fresh"))
	subscription, baseline := terminal.subscribe(99, false)
	defer terminal.unsubscribe(subscription)
	if baseline.Sequence != 1 || string(baseline.Data) != "fresh" {
		t.Fatalf("unexpected reset baseline: %#v", baseline)
	}
}

func TestNativeTerminalProtocolTakeoverAppliesAndBroadcastsSize(t *testing.T) {
	pty := &fakeHostTerminalPTY{}
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.ptmx = pty
	terminal.cols = 42
	terminal.rows = 18
	first, _ := terminal.subscribe(0, false)
	second, baseline := terminal.subscribe(0, true)
	second.AllowControl = true
	if baseline.Cols != 42 || baseline.Rows != 18 {
		t.Fatalf("baseline omitted authoritative size: %#v", baseline)
	}
	if !terminal.releaseControl(first.ID) {
		t.Fatal("first subscriber failed to release control")
	}
	<-first.Events
	<-second.Events
	if granted, reason := terminal.takeControl(second.ID, 56, 21); !granted || reason != "" {
		t.Fatalf("takeover failed: granted=%v reason=%q", granted, reason)
	}
	if pty.cols != 56 || pty.rows != 21 || terminal.cols != 56 || terminal.rows != 21 {
		t.Fatalf("takeover did not apply size: pty=%dx%d terminal=%dx%d", pty.cols, pty.rows, terminal.cols, terminal.rows)
	}
	firstControl := <-first.Events
	secondControl := <-second.Events
	if firstControl.Cols != 56 || firstControl.Rows != 21 || secondControl.Cols != 56 || secondControl.Rows != 21 {
		t.Fatalf("authoritative size was not broadcast: first=%#v second=%#v", firstControl, secondControl)
	}
	terminal.unsubscribe(second)
	terminal.unsubscribe(first)
}

func TestNativeTerminalProtocolDeniedTakeoverPreservesSize(t *testing.T) {
	pty := &fakeHostTerminalPTY{cols: 100, rows: 40}
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.ptmx = pty
	terminal.cols = 100
	terminal.rows = 40
	first, _ := terminal.subscribe(0, false)
	second, _ := terminal.subscribe(0, false)
	if granted, _ := terminal.takeControl(second.ID, 48, 20); granted {
		t.Fatal("active control lease should reject takeover")
	}
	if pty.cols != 100 || pty.rows != 40 || terminal.cols != 100 || terminal.rows != 40 {
		t.Fatalf("denied takeover changed size: pty=%dx%d terminal=%dx%d", pty.cols, pty.rows, terminal.cols, terminal.rows)
	}
	terminal.unsubscribe(second)
	terminal.unsubscribe(first)
}
