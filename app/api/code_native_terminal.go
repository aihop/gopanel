package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/token"
)

const (
	nativeTerminalHistoryLimit       = 1024 * 1024
	nativeTerminalBaselineChunkLimit = 64 * 1024
)

type nativeCodeTerminal struct {
	mu               sync.Mutex
	sessionID        uint
	projectID        uint
	command          *exec.Cmd
	ptmx             nativeTerminal
	sequence         uint64
	history          []nativeTerminalChunk
	historySize      int
	historyTruncated bool
	subscribers      map[string]*nativeTerminalSubscription
	controllerID     string
	controlExpiresAt time.Time
	controlTimer     *time.Timer
	done             chan struct{}
	lease            *codeExecutionLease
	executorName     string
}

type nativeCodeTerminalManager struct {
	mu       sync.Mutex
	sessions map[uint]*nativeCodeTerminal
}

var codeNativeTerminals = &nativeCodeTerminalManager{sessions: make(map[uint]*nativeCodeTerminal)}

func supportsNativeCodeTerminal(executorID string) bool {
	definition, err := getCodeExecutorDefinition(executorID)
	return err == nil && definition.NativeTerminal && nativeTerminalPlatformSupported()
}

func (manager *nativeCodeTerminalManager) attach(
	session *model.AIDevSession,
	cols, rows uint16,
) (*nativeCodeTerminal, bool, error) {
	if err := validateCodeSessionDevelopmentOpen(session); err != nil {
		return nil, false, err
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if terminal := manager.sessions[session.ID]; terminal != nil {
		return terminal, false, nil
	}
	if err := validateCodeTokenBudget(session); err != nil {
		return nil, false, err
	}
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionInteractive, false)
	if err != nil {
		return nil, false, err
	}
	command, preparedSessionID, err := buildNativeCodeCommand(session)
	if err != nil {
		lease.Release()
		return nil, false, err
	}
	configureNativeTerminalEnvironment(command)
	ptmx, err := startNativeTerminal(command, cols, rows)
	if err != nil {
		lease.Release()
		return nil, false, err
	}
	terminal := &nativeCodeTerminal{
		sessionID:    session.ID,
		projectID:    session.ProjectID,
		command:      command,
		ptmx:         ptmx,
		subscribers:  make(map[string]*nativeTerminalSubscription),
		done:         make(chan struct{}),
		lease:        lease,
		executorName: session.AgentName,
	}
	if preparedSessionID != "" && session.NativeSessionID != preparedSessionID {
		session.NativeSessionID = preparedSessionID
		if err := repo.NewAIDevSessionRepo().UpdateSession(session); err != nil {
			_ = command.Process.Kill()
			_ = ptmx.Close()
			lease.Release()
			return nil, false, err
		}
	}
	lease.SetCancel(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
	})
	if err := global.DB.Model(&model.AIDevSession{}).Where("id = ?", session.ID).
		Updates(map[string]any{"status": "active", "current_stage": "interactive"}).Error; err != nil {
		_ = command.Process.Kill()
		_ = ptmx.Close()
		lease.Release()
		return nil, false, err
	}
	manager.sessions[session.ID] = terminal
	go terminal.readOutput()
	go terminal.wait(manager)
	if session.AgentName == "codex" {
		go discoverNativeCodexSession(session, command.Process.Pid, time.Now())
	} else if session.AgentName == "opencode" && session.NativeSessionID == "" {
		go discoverNativeOpenCodeSession(session, command.Path, command.Env, time.Now(), terminal.done)
	}
	go watchNativeCodeNotifications(session.ID, terminal.done)
	return terminal, true, nil
}

func configureNativeTerminalEnvironment(command *exec.Cmd) {
	commandEnv := command.Env
	if len(commandEnv) == 0 {
		commandEnv = os.Environ()
	}
	commandEnv = upsertEnvironment(commandEnv, "TERM", "xterm-256color")
	command.Env = upsertEnvironment(commandEnv, "COLORTERM", "truecolor")
}

func (terminal *nativeCodeTerminal) readOutput() {
	buffer := make([]byte, 32*1024)
	for {
		count, err := terminal.ptmx.Read(buffer)
		if count > 0 {
			terminal.publish(buffer[:count])
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
				global.LOG.Errorf("Read native Code terminal %d failed: %v", terminal.sessionID, err)
			}
			return
		}
	}
}

func (terminal *nativeCodeTerminal) publish(output []byte) {
	chunk := append([]byte(nil), output...)
	terminal.mu.Lock()
	terminal.sequence++
	terminal.history = append(terminal.history, nativeTerminalChunk{Sequence: terminal.sequence, Data: chunk})
	terminal.historySize += len(chunk)
	for terminal.historySize > nativeTerminalHistoryLimit {
		terminal.historyTruncated = true
		excess := terminal.historySize - nativeTerminalHistoryLimit
		if len(terminal.history) > 1 && len(terminal.history[0].Data) <= excess {
			terminal.historySize -= len(terminal.history[0].Data)
			terminal.history = terminal.history[1:]
			continue
		}
		terminal.history[0].Data = append([]byte(nil), terminal.history[0].Data[excess:]...)
		terminal.historySize -= excess
	}
	event := nativeTerminalEvent{Type: "output", Sequence: terminal.sequence, Data: chunk}
	for _, subscription := range terminal.subscribers {
		if subscription.NeedsResync {
			continue
		}
		select {
		case subscription.Events <- event:
		default:
			terminal.markSubscriptionForResync(subscription)
		}
	}
	terminal.mu.Unlock()
}

func (terminal *nativeCodeTerminal) write(subscriptionID string, data []byte) error {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	subscription := terminal.subscribers[subscriptionID]
	if subscription == nil || subscription.ReadOnly || terminal.controllerID != subscriptionID || terminal.controlExpiredLocked(time.Now()) {
		return errors.New("当前连接没有终端输入权")
	}
	terminal.renewControlLeaseLocked(time.Now())
	_, err := terminal.ptmx.Write(data)
	return err
}

func (terminal *nativeCodeTerminal) resize(cols, rows uint16) error {
	return terminal.ptmx.Resize(cols, rows)
}

func (terminal *nativeCodeTerminal) wait(manager *nativeCodeTerminalManager) {
	err := terminal.command.Wait()
	_ = terminal.ptmx.Close()
	terminal.publish([]byte(fmt.Sprintf("\r\n\x1b[33m[GoPanel] %s 会话已退出: %v\x1b[0m\r\n", terminal.executorName, err)))
	terminal.mu.Lock()
	if terminal.controlTimer != nil {
		terminal.controlTimer.Stop()
		terminal.controlTimer = nil
	}
	terminal.controllerID = ""
	terminal.controlExpiresAt = time.Time{}
	closedEvent := nativeTerminalEvent{Type: "closed", Sequence: terminal.sequence}
	for subscriptionID, subscription := range terminal.subscribers {
		select {
		case subscription.Events <- closedEvent:
		default:
		}
		delete(terminal.subscribers, subscriptionID)
		close(subscription.Events)
	}
	terminal.mu.Unlock()
	close(terminal.done)
	manager.mu.Lock()
	delete(manager.sessions, terminal.sessionID)
	manager.mu.Unlock()
	terminal.lease.Release()
	_ = global.DB.Model(&model.AIDevSession{}).
		Where("id = ? AND current_stage = ?", terminal.sessionID, "interactive").
		Update("current_stage", "idle").Error
}

func serveNativeCodeTerminal(
	wsConn *websocket.Conn,
	session *model.AIDevSession,
	cols, rows uint16,
	claims *token.CustomClaims,
) {
	terminal, _, err := codeNativeTerminals.attach(session, cols, rows)
	if err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("启动原生 %s 会话失败: %v", session.AgentName, err)))
		return
	}
	afterSequence, _ := strconv.ParseUint(wsConn.Query("after_sequence", "0"), 10, 64)
	readOnly := wsConn.Query("read_only") == "1"
	subscription, baseline := terminal.subscribe(afterSequence, readOnly)
	subscription.UserID = claims.UserId
	subscription.IP = wsConn.IP()
	subscription.DeviceID, _ = wsConn.Locals(middleware.MobileDeviceIDKey).(uint)
	subscription.AllowControl = subscription.DeviceID > 0
	if baseline.HasControl && !readOnly && wsConn.Query("take_control") != "1" {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "terminal_control_acquire", "success", subscription.ID, "连接自动获得控制权", wsConn.IP(), time.Now(), codeAuditMeta{"deviceId": subscription.DeviceID, "automatic": true})
	}
	if wsConn.Query("take_control") == "1" {
		startedAt := time.Now()
		granted, reason := terminal.takeControl(subscription.ID)
		baseline.HasControl = granted
		baseline.ControlReason = reason
		status := "success"
		if !granted {
			status = "denied"
		}
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "terminal_control_acquire", status, subscription.ID, reason, wsConn.IP(), startedAt, codeAuditMeta{"deviceId": subscription.DeviceID, "automatic": true})
	}
	defer func() {
		controlled, _ := terminal.controlState(subscription.ID)
		terminal.unsubscribe(subscription)
		if controlled {
			recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "terminal_control_release", "success", subscription.ID, "连接断开并释放控制权", wsConn.IP(), time.Now(), codeAuditMeta{"deviceId": subscription.DeviceID, "automatic": true})
		}
	}()
	var writeMu sync.Mutex
	writeEvent := func(event nativeTerminalEvent) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		chunks := splitNativeTerminalBaseline(event)
		for index, chunk := range chunks {
			payload, _ := json.Marshal(struct {
				Type           string `json:"type"`
				Sequence       uint64 `json:"sequence"`
				StartSequence  uint64 `json:"startSequence,omitempty"`
				RequestID      string `json:"requestId,omitempty"`
				Data           string `json:"data,omitempty"`
				HasControl     bool   `json:"hasControl"`
				ControlReason  string `json:"controlReason,omitempty"`
				LeaseExpiresAt int64  `json:"leaseExpiresAt,omitempty"`
				Truncated      bool   `json:"truncated,omitempty"`
				ChunkIndex     int    `json:"chunkIndex,omitempty"`
				ChunkCount     int    `json:"chunkCount,omitempty"`
			}{
				Type: chunk.Type, Sequence: chunk.Sequence, StartSequence: chunk.StartSequence,
				RequestID: chunk.RequestID, Data: string(chunk.Data), HasControl: chunk.HasControl,
				ControlReason: chunk.ControlReason, LeaseExpiresAt: chunk.LeaseExpiresAt,
				Truncated: chunk.Truncated && index == 0, ChunkIndex: index, ChunkCount: len(chunks),
			})
			if err := wsConn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return err
			}
		}
		return nil
	}
	_ = writeEvent(baseline)
	go func() {
		for event := range subscription.Events {
			if writeEvent(event) != nil {
				_ = wsConn.Close()
				return
			}
		}
		_ = wsConn.Close()
	}()
	for {
		messageType, payload, readErr := wsConn.ReadMessage()
		if readErr != nil {
			return
		}
		if messageType != websocket.TextMessage {
			continue
		}
		var message WsMsg
		if json.Unmarshal(payload, &message) != nil {
			continue
		}
		switch message.Type {
		case "cmd":
			if writeErr := terminal.write(subscription.ID, []byte(message.Data)); writeErr != nil {
				_ = writeEvent(nativeTerminalEvent{Type: "control", HasControl: false})
			}
		case "take_control":
			startedAt := time.Now()
			granted, reason := terminal.takeControl(subscription.ID)
			status := "success"
			if !granted {
				status = "denied"
			}
			recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "terminal_control_acquire", status, subscription.ID, reason, wsConn.IP(), startedAt, codeAuditMeta{"deviceId": subscription.DeviceID, "readOnly": subscription.DefaultReadOnly})
			if !granted {
				_, expiresAt := terminal.controlState(subscription.ID)
				_ = writeEvent(nativeTerminalEvent{Type: "control", HasControl: false, ControlReason: reason, LeaseExpiresAt: expiresAt})
			}
		case "release_control":
			startedAt := time.Now()
			released := terminal.releaseControl(subscription.ID)
			status, detail := "success", "控制权已释放"
			if !released {
				status, detail = "denied", "当前连接没有终端控制权"
			}
			recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "terminal_control_release", status, subscription.ID, detail, wsConn.IP(), startedAt, codeAuditMeta{"deviceId": subscription.DeviceID})
		case "ack":
			if sequence, parseErr := strconv.ParseUint(message.Data, 10, 64); parseErr == nil {
				terminal.acknowledge(subscription.ID, sequence)
			}
		case "resync":
			var request nativeTerminalResyncRequest
			if json.Unmarshal([]byte(message.Data), &request) == nil {
				terminal.resync(subscription.ID, request.Sequence, request.RequestID)
			}
		case "resize":
			var size struct {
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if json.Unmarshal([]byte(message.Data), &size) == nil && terminal.hasControl(subscription.ID) {
				terminal.renewControlLease(subscription.ID)
				_ = terminal.resize(size.Cols, size.Rows)
			}
		case "ping":
			writeMu.Lock()
			_ = wsConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
			writeMu.Unlock()
		}
	}
}
