package api

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync/atomic"
	"time"
)

func nativeTerminalConnectionErrorPayload(err error) []byte {
	eventType, code := "error", "start_failed"
	if errors.Is(err, errNativeCodeTerminalInactive) {
		eventType, code = "inactive", "terminal_inactive"
	} else if errors.Is(err, errCodeExecutionBusy) {
		code = "workspace_busy"
	}
	payload, _ := json.Marshal(struct {
		Type string `json:"type"`
		Code string `json:"code"`
	}{Type: eventType, Code: code})
	return payload
}

const nativeTerminalControlLease = 60 * time.Second

type nativeTerminalChunk struct {
	Sequence uint64
	Data     []byte
}

type nativeTerminalEvent struct {
	Type           string
	Sequence       uint64
	StartSequence  uint64
	RequestID      string
	Data           []byte
	HasControl     bool
	Truncated      bool
	ControlReason  string
	LeaseExpiresAt int64
	Cols           uint16
	Rows           uint16
}

type nativeTerminalControlRequest struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type nativeTerminalSubscription struct {
	ID              string
	Events          chan nativeTerminalEvent
	AckSequence     uint64
	NeedsResync     bool
	UserID          uint
	DeviceID        uint
	IP              string
	ReadOnly        bool
	DefaultReadOnly bool
	AllowControl    bool
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

func (terminal *nativeCodeTerminal) subscribe(afterSequence uint64, readOnly bool) (*nativeTerminalSubscription, nativeTerminalEvent) {
	subscription := &nativeTerminalSubscription{
		ID:              nextNativeTerminalConnectionID(),
		Events:          make(chan nativeTerminalEvent, 128),
		ReadOnly:        readOnly,
		DefaultReadOnly: readOnly,
	}
	terminal.mu.Lock()
	if afterSequence > terminal.sequence {
		afterSequence = 0
	}
	terminal.subscribers[subscription.ID] = subscription
	if !readOnly && (terminal.controllerID == "" || terminal.controlExpiredLocked(time.Now())) {
		terminal.controllerID = subscription.ID
		terminal.renewControlLeaseLocked(time.Now())
	}
	baseline := terminal.baselineAfter(afterSequence, "")
	baseline.HasControl = terminal.controllerID == subscription.ID
	baseline.LeaseExpiresAt = terminal.controlExpiresAt.UnixMilli()
	baseline.Cols = terminal.cols
	baseline.Rows = terminal.rows
	subscription.AckSequence = afterSequence
	terminal.mu.Unlock()
	return subscription, baseline
}

func (terminal *nativeCodeTerminal) baselineAfter(afterSequence uint64, requestID string) nativeTerminalEvent {
	startSequence := afterSequence + 1
	truncated := false
	if len(terminal.history) > 0 {
		oldestSequence := terminal.history[0].Sequence
		truncated = terminal.historyTruncated && afterSequence < oldestSequence
		if afterSequence > 0 && oldestSequence > afterSequence+1 {
			truncated = true
		}
		if truncated || afterSequence == 0 {
			startSequence = oldestSequence
		}
	}
	return nativeTerminalEvent{
		Type: "baseline", Sequence: terminal.sequence, StartSequence: startSequence,
		RequestID: requestID, Data: terminal.historyAfter(afterSequence), Truncated: truncated,
	}
}

func splitNativeTerminalBaseline(event nativeTerminalEvent) []nativeTerminalEvent {
	if event.Type != "baseline" || len(event.Data) <= nativeTerminalBaselineChunkLimit {
		return []nativeTerminalEvent{event}
	}
	chunks := make([]nativeTerminalEvent, 0, (len(event.Data)+nativeTerminalBaselineChunkLimit-1)/nativeTerminalBaselineChunkLimit)
	for start := 0; start < len(event.Data); {
		end := terminalChunkEnd(event.Data, start, nativeTerminalBaselineChunkLimit)
		chunk := event
		chunk.Data = event.Data[start:end]
		chunks = append(chunks, chunk)
		start = end
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
		terminal.controlExpiresAt = time.Time{}
		if terminal.controlTimer != nil {
			terminal.controlTimer.Stop()
			terminal.controlTimer = nil
		}
	}
	terminal.mu.Unlock()
	if controlChanged {
		terminal.broadcastControl()
	}
}

func (terminal *nativeCodeTerminal) takeControl(subscriptionID string, cols, rows uint16) (bool, string) {
	terminal.mu.Lock()
	subscription, exists := terminal.subscribers[subscriptionID]
	if !exists {
		terminal.mu.Unlock()
		return false, "连接不存在"
	}
	if subscription.ReadOnly && !subscription.AllowControl {
		terminal.mu.Unlock()
		return false, "只读连接不能接管终端输入"
	}
	now := time.Now()
	if terminal.controllerID != "" && terminal.controllerID != subscriptionID && !terminal.controlExpiredLocked(now) {
		terminal.mu.Unlock()
		return false, "其他设备正在控制终端"
	}
	if cols > 0 && rows > 0 {
		if err := terminal.resizeLocked(cols, rows); err != nil {
			terminal.mu.Unlock()
			return false, err.Error()
		}
	}
	if terminal.controllerID != "" && terminal.controllerID != subscriptionID {
		terminal.restoreSubscriptionReadOnlyLocked(terminal.controllerID)
	}
	changed := terminal.controllerID != subscriptionID
	terminal.controllerID = subscriptionID
	subscription.ReadOnly = false
	terminal.renewControlLeaseLocked(now)
	if changed || cols > 0 && rows > 0 {
		terminal.broadcastControlLocked()
	}
	terminal.mu.Unlock()
	return true, ""
}

func (terminal *nativeCodeTerminal) releaseControl(subscriptionID string) bool {
	terminal.mu.Lock()
	if terminal.controllerID != subscriptionID {
		terminal.mu.Unlock()
		return false
	}
	terminal.restoreSubscriptionReadOnlyLocked(subscriptionID)
	terminal.controllerID = ""
	terminal.controlExpiresAt = time.Time{}
	if terminal.controlTimer != nil {
		terminal.controlTimer.Stop()
		terminal.controlTimer = nil
	}
	terminal.mu.Unlock()
	terminal.broadcastControl()
	return true
}

func (terminal *nativeCodeTerminal) hasControl(subscriptionID string) bool {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	if terminal.controlExpiredLocked(time.Now()) {
		return false
	}
	return terminal.controllerID == subscriptionID
}

func (terminal *nativeCodeTerminal) broadcastControl() {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	terminal.broadcastControlLocked()
}

func (terminal *nativeCodeTerminal) broadcastControlLocked() {
	for subscriptionID, subscription := range terminal.subscribers {
		event := nativeTerminalEvent{Type: "control", Sequence: terminal.sequence, HasControl: terminal.controllerID == subscriptionID, LeaseExpiresAt: terminal.controlExpiresAt.UnixMilli(), Cols: terminal.cols, Rows: terminal.rows}
		select {
		case subscription.Events <- event:
		default:
		}
	}
}

func (terminal *nativeCodeTerminal) renewControlLease(subscriptionID string) bool {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	if terminal.controllerID != subscriptionID || terminal.controlExpiredLocked(time.Now()) {
		return false
	}
	terminal.renewControlLeaseLocked(time.Now())
	return true
}

func (terminal *nativeCodeTerminal) controlState(subscriptionID string) (bool, int64) {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	if terminal.controlExpiredLocked(time.Now()) {
		return false, terminal.controlExpiresAt.UnixMilli()
	}
	return terminal.controllerID == subscriptionID, terminal.controlExpiresAt.UnixMilli()
}

func (terminal *nativeCodeTerminal) renewControlLeaseLocked(now time.Time) {
	terminal.controlExpiresAt = now.Add(nativeTerminalControlLease)
	if terminal.controlTimer != nil {
		terminal.controlTimer.Stop()
	}
	expectedController := terminal.controllerID
	terminal.controlTimer = time.AfterFunc(nativeTerminalControlLease, func() {
		terminal.expireControl(expectedController)
	})
}

func (terminal *nativeCodeTerminal) controlExpiredLocked(now time.Time) bool {
	return terminal.controllerID != "" && !terminal.controlExpiresAt.IsZero() && !now.Before(terminal.controlExpiresAt)
}

func (terminal *nativeCodeTerminal) clearExpiredControlLocked() {
	terminal.restoreSubscriptionReadOnlyLocked(terminal.controllerID)
	terminal.controllerID = ""
	terminal.controlExpiresAt = time.Time{}
	terminal.controlTimer = nil
}

func (terminal *nativeCodeTerminal) restoreSubscriptionReadOnlyLocked(subscriptionID string) {
	if subscription := terminal.subscribers[subscriptionID]; subscription != nil {
		subscription.ReadOnly = subscription.DefaultReadOnly
	}
}

func (terminal *nativeCodeTerminal) expireControl(expectedController string) {
	terminal.mu.Lock()
	if terminal.controllerID != expectedController || !terminal.controlExpiredLocked(time.Now()) {
		terminal.mu.Unlock()
		return
	}
	subscription := terminal.subscribers[expectedController]
	terminal.clearExpiredControlLocked()
	terminal.mu.Unlock()
	if subscription != nil {
		recordCodeAudit(subscription.UserID, terminal.projectID, terminal.sessionID, "terminal_control_expire", "success", subscription.ID, "终端控制租约因空闲自动过期", subscription.IP, time.Now(), codeAuditMeta{"deviceId": subscription.DeviceID, "automatic": true})
	}
	terminal.broadcastControl()
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
	baseline.LeaseExpiresAt = terminal.controlExpiresAt.UnixMilli()
	baseline.Cols = terminal.cols
	baseline.Rows = terminal.rows
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
