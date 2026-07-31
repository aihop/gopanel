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
	Type          string
	Sequence      uint64
	StartSequence uint64
	RequestID     string
	Data          []byte
	HasControl    bool
	Truncated     bool
}

type nativeTerminalSubscription struct {
	ID          string
	Events      chan nativeTerminalEvent
	AckSequence uint64
	NeedsResync bool
}

type nativeTerminalResyncRequest struct {
	Sequence  uint64 `json:"sequence"`
	RequestID string `json:"requestId"`
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
	terminal.subscribers[subscription.ID] = subscription
	if terminal.controllerID == "" {
		terminal.controllerID = subscription.ID
	}
	baseline := terminal.baselineAfter(afterSequence, "")
	baseline.HasControl = terminal.controllerID == subscription.ID
	subscription.AckSequence = afterSequence
	terminal.mu.Unlock()
	return subscription, baseline
}

func (terminal *nativeCodeTerminal) baselineAfter(afterSequence uint64, requestID string) nativeTerminalEvent {
	startSequence, truncated := afterSequence+1, false
	if len(terminal.history) > 0 {
		oldest := terminal.history[0].Sequence
		truncated = terminal.historyTruncated && afterSequence < oldest || afterSequence > 0 && oldest > afterSequence+1
		if truncated || afterSequence == 0 {
			startSequence = oldest
		}
	}
	return nativeTerminalEvent{Type: "baseline", Sequence: terminal.sequence, StartSequence: startSequence, RequestID: requestID, Data: terminal.historyAfter(afterSequence), Truncated: truncated}
}

func splitNativeTerminalBaseline(event nativeTerminalEvent) []nativeTerminalEvent {
	if event.Type != "baseline" || len(event.Data) <= nativeTerminalBaselineChunkLimit {
		return []nativeTerminalEvent{event}
	}
	chunks := make([]nativeTerminalEvent, 0, (len(event.Data)+nativeTerminalBaselineChunkLimit-1)/nativeTerminalBaselineChunkLimit)
	for start := 0; start < len(event.Data); start += nativeTerminalBaselineChunkLimit {
		end := min(start+nativeTerminalBaselineChunkLimit, len(event.Data))
		chunk := event
		chunk.Data = event.Data[start:end]
		chunks = append(chunks, chunk)
	}
	return chunks
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
	registered, exists := terminal.subscribers[subscription.ID]
	if exists {
		delete(terminal.subscribers, subscription.ID)
		close(registered.Events)
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

func (terminal *nativeCodeTerminal) releaseControl(subscriptionID string) bool {
	terminal.mu.Lock()
	if terminal.controllerID != subscriptionID {
		terminal.mu.Unlock()
		return false
	}
	terminal.controllerID = ""
	terminal.mu.Unlock()
	terminal.broadcastControl()
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
	for subscriptionID, subscription := range terminal.subscribers {
		event := nativeTerminalEvent{Type: "control", Sequence: terminal.sequence, HasControl: terminal.controllerID == subscriptionID}
		select {
		case subscription.Events <- event:
		default:
		}
	}
}

func (terminal *nativeCodeTerminal) acknowledge(subscriptionID string, sequence uint64) {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	if subscription := terminal.subscribers[subscriptionID]; subscription != nil && sequence > subscription.AckSequence {
		subscription.AckSequence = min(sequence, terminal.sequence)
	}
}

func (terminal *nativeCodeTerminal) resync(subscriptionID string, afterSequence uint64, requestID string) bool {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	subscription := terminal.subscribers[subscriptionID]
	if subscription == nil {
		return false
	}
	if afterSequence > terminal.sequence {
		afterSequence = 0
	}
	drainNativeTerminalEvents(subscription.Events)
	subscription.NeedsResync = false
	subscription.AckSequence = afterSequence
	baseline := terminal.baselineAfter(afterSequence, requestID)
	baseline.HasControl = terminal.controllerID == subscriptionID
	subscription.Events <- baseline
	return true
}

func (terminal *nativeCodeTerminal) markSubscriptionForResync(subscription *nativeTerminalSubscription) {
	if subscription.NeedsResync {
		return
	}
	subscription.NeedsResync = true
	drainNativeTerminalEvents(subscription.Events)
	subscription.Events <- nativeTerminalEvent{Type: "resync_required", Sequence: terminal.sequence}
}

func drainNativeTerminalEvents(events chan nativeTerminalEvent) {
	for {
		select {
		case <-events:
		default:
			return
		}
	}
}
