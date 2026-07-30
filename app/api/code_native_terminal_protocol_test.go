package api

import "testing"

func newNativeTerminalProtocolTestSubject() *nativeCodeTerminal {
	return &nativeCodeTerminal{subscribers: make(map[string]chan nativeTerminalEvent)}
}

func TestNativeTerminalProtocolReplaysOnlyMissingOutput(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.publish([]byte("one"))
	terminal.publish([]byte("two"))
	terminal.publish([]byte("three"))
	subscription, baseline := terminal.subscribe(1)
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

func TestNativeTerminalProtocolTransfersAndReleasesControl(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	first, firstBaseline := terminal.subscribe(0)
	second, secondBaseline := terminal.subscribe(0)
	if !firstBaseline.HasControl || secondBaseline.HasControl {
		t.Fatalf("unexpected initial control: first=%v second=%v", firstBaseline.HasControl, secondBaseline.HasControl)
	}
	if !terminal.takeControl(second.ID) {
		t.Fatal("second subscriber failed to take control")
	}
	firstControl := <-first.Events
	secondControl := <-second.Events
	if firstControl.HasControl || !secondControl.HasControl {
		t.Fatalf("unexpected transferred control: first=%#v second=%#v", firstControl, secondControl)
	}
	terminal.unsubscribe(second)
	firstControl = <-first.Events
	if firstControl.HasControl || terminal.controllerID != "" {
		t.Fatalf("control was not released: %#v", firstControl)
	}
	terminal.unsubscribe(first)
}

func TestNativeTerminalProtocolCanReleaseReadOnlySubscriberControl(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	subscription, baseline := terminal.subscribe(0)
	if !baseline.HasControl {
		t.Fatal("first subscriber should initially receive control")
	}
	if !terminal.releaseControl(subscription.ID) || terminal.controllerID != "" {
		t.Fatal("read-only subscriber should release terminal control")
	}
	control := <-subscription.Events
	if control.HasControl {
		t.Fatalf("unexpected control event after release: %#v", control)
	}
	terminal.unsubscribe(subscription)
}

func TestNativeTerminalProtocolResetsStaleSequence(t *testing.T) {
	terminal := newNativeTerminalProtocolTestSubject()
	terminal.publish([]byte("fresh"))
	subscription, baseline := terminal.subscribe(99)
	defer terminal.unsubscribe(subscription)
	if baseline.Sequence != 1 || string(baseline.Data) != "fresh" {
		t.Fatalf("unexpected reset baseline: %#v", baseline)
	}
}
