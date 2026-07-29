package api

import (
	"strconv"
	"sync/atomic"
)

type nativeTerminalChunk struct {
	Sequence uint64
	Data     []byte
}

type nativeTerminalEvent struct {
	Type       string
	Sequence   uint64
	Data       []byte
	HasControl bool
}

type nativeTerminalSubscription struct {
	ID     string
	Events chan nativeTerminalEvent
}

var nativeTerminalConnectionSequence uint64

func nextNativeTerminalConnectionID() string {
	sequence := atomic.AddUint64(&nativeTerminalConnectionSequence, 1)
	return strconv.FormatUint(sequence, 10)
}

func (terminal *nativeCodeTerminal) subscribe(afterSequence uint64) (*nativeTerminalSubscription, nativeTerminalEvent) {
	subscription := &nativeTerminalSubscription{
		ID:     nextNativeTerminalConnectionID(),
		Events: make(chan nativeTerminalEvent, 128),
	}
	terminal.mu.Lock()
	if afterSequence > terminal.sequence {
		afterSequence = 0
	}
	terminal.subscribers[subscription.ID] = subscription.Events
	if terminal.controllerID == "" {
		terminal.controllerID = subscription.ID
	}
	baseline := nativeTerminalEvent{
		Type:       "baseline",
		Sequence:   terminal.sequence,
		Data:       terminal.historyAfter(afterSequence),
		HasControl: terminal.controllerID == subscription.ID,
	}
	terminal.mu.Unlock()
	return subscription, baseline
}

func (terminal *nativeCodeTerminal) historyAfter(afterSequence uint64) []byte {
	var result []byte
	for _, chunk := range terminal.history {
		if chunk.Sequence > afterSequence {
			result = append(result, chunk.Data...)
		}
	}
	return result
}

func (terminal *nativeCodeTerminal) unsubscribe(subscription *nativeTerminalSubscription) {
	if subscription == nil {
		return
	}
	terminal.mu.Lock()
	events, exists := terminal.subscribers[subscription.ID]
	if exists {
		delete(terminal.subscribers, subscription.ID)
		close(events)
	}
	controlChanged := terminal.controllerID == subscription.ID
	if controlChanged {
		terminal.controllerID = ""
	}
	terminal.mu.Unlock()
	if controlChanged {
		terminal.broadcastControl()
	}
}

func (terminal *nativeCodeTerminal) takeControl(subscriptionID string) bool {
	terminal.mu.Lock()
	if _, exists := terminal.subscribers[subscriptionID]; !exists {
		terminal.mu.Unlock()
		return false
	}
	changed := terminal.controllerID != subscriptionID
	terminal.controllerID = subscriptionID
	terminal.mu.Unlock()
	if changed {
		terminal.broadcastControl()
	}
	return true
}

func (terminal *nativeCodeTerminal) hasControl(subscriptionID string) bool {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	return terminal.controllerID == subscriptionID
}

func (terminal *nativeCodeTerminal) broadcastControl() {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	for subscriptionID, events := range terminal.subscribers {
		event := nativeTerminalEvent{Type: "control", Sequence: terminal.sequence, HasControl: terminal.controllerID == subscriptionID}
		select {
		case events <- event:
		default:
		}
	}
}
