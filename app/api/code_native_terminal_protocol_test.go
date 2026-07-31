package api

import (
	"testing"
	"time"
)

func newNativeTerminalProtocolTestSubject() *nativeCodeTerminal {
	return &nativeCodeTerminal{subscribers: make(map[string]*nativeTerminalSubscription)}
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
	if granted, _ := terminal.takeControl(second.ID); granted {
		t.Fatal("second subscriber should not preempt active control lease")
	}
	if !terminal.releaseControl(first.ID) {
		t.Fatal("first subscriber failed to release control")
	}
	<-first.Events
	<-second.Events
	if granted, reason := terminal.takeControl(second.ID); !granted || reason != "" {
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

func TestNativeTerminalProtocolReadOnlySubscriberNeverGetsControl(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	subscription, baseline := terminal.subscribe(0, true)
	if baseline.HasControl || terminal.controllerID != "" {
		t.Fatal("read-only subscriber should not receive terminal control")
	}
	if granted, reason := terminal.takeControl(subscription.ID); granted || reason != "只读连接不能接管终端输入" {
		t.Fatalf("read-only subscriber took control: granted=%v reason=%q", granted, reason)
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
	if granted, reason := terminal.takeControl(second.ID); !granted || reason != "" {
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
