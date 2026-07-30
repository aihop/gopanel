package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/creack/pty"
)

const nativeTerminalHistoryLimit = 1024 * 1024

type nativeCodeTerminal struct {
	mu           sync.Mutex
	sessionID    uint
	command      *exec.Cmd
	ptmx         *os.File
	sequence     uint64
	history      []nativeTerminalChunk
	historySize  int
	subscribers  map[string]chan nativeTerminalEvent
	controllerID string
	done         chan struct{}
}

type nativeCodeTerminalManager struct {
	mu       sync.Mutex
	sessions map[uint]*nativeCodeTerminal
}

var codeNativeTerminals = &nativeCodeTerminalManager{sessions: make(map[uint]*nativeCodeTerminal)}

func supportsNativeCodeTerminal(executorID string) bool {
	return executorID == "codex"
}

func (manager *nativeCodeTerminalManager) attach(
	session *model.AIDevSession,
	cols, rows uint16,
) (*nativeCodeTerminal, bool, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if terminal := manager.sessions[session.ID]; terminal != nil {
		return terminal, false, nil
	}
	command, err := buildNativeCodexCommand(session)
	if err != nil {
		return nil, false, err
	}
	command.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := pty.StartWithSize(command, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, false, err
	}
	terminal := &nativeCodeTerminal{
		sessionID:   session.ID,
		command:     command,
		ptmx:        ptmx,
		subscribers: make(map[string]chan nativeTerminalEvent),
		done:        make(chan struct{}),
	}
	manager.sessions[session.ID] = terminal
	go terminal.readOutput()
	go terminal.wait(manager)
	go discoverNativeCodexSession(session, command.Process.Pid, time.Now())
	go watchNativeCodeNotifications(session.ID, terminal.done)
	return terminal, true, nil
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
	for _, subscriber := range terminal.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
	terminal.mu.Unlock()
}

func (terminal *nativeCodeTerminal) write(subscriptionID string, data []byte) error {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	if terminal.controllerID != subscriptionID {
		return errors.New("当前连接没有终端输入权")
	}
	_, err := terminal.ptmx.Write(data)
	return err
}

func (terminal *nativeCodeTerminal) resize(cols, rows uint16) error {
	return pty.Setsize(terminal.ptmx, &pty.Winsize{Cols: cols, Rows: rows})
}

func (terminal *nativeCodeTerminal) wait(manager *nativeCodeTerminalManager) {
	err := terminal.command.Wait()
	_ = terminal.ptmx.Close()
	terminal.publish([]byte(fmt.Sprintf("\r\n\x1b[33m[GoPanel] Codex 会话已退出: %v\x1b[0m\r\n", err)))
	terminal.mu.Lock()
	closedEvent := nativeTerminalEvent{Type: "closed", Sequence: terminal.sequence}
	for subscriptionID, subscriber := range terminal.subscribers {
		select {
		case subscriber <- closedEvent:
		default:
		}
		delete(terminal.subscribers, subscriptionID)
		close(subscriber)
	}
	terminal.mu.Unlock()
	close(terminal.done)
	manager.mu.Lock()
	delete(manager.sessions, terminal.sessionID)
	manager.mu.Unlock()
	if session, getErr := repo.NewAIDevSessionRepo().GetSessionByID(terminal.sessionID); getErr == nil {
		session.CurrentStage = "idle"
		_ = repo.NewAIDevSessionRepo().UpdateSession(session)
	}
}

func serveNativeCodeTerminal(
	wsConn *websocket.Conn,
	sessionRepo repo.IAIDevSessionRepo,
	session *model.AIDevSession,
	cols, rows uint16,
) {
	terminal, created, err := codeNativeTerminals.attach(session, cols, rows)
	if err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("启动原生 Codex 会话失败: %v", err)))
		return
	}
	if created {
		session.CurrentStage = "interactive"
		_ = sessionRepo.UpdateSession(session)
	}
	afterSequence, _ := strconv.ParseUint(wsConn.Query("after_sequence", "0"), 10, 64)
	subscription, baseline := terminal.subscribe(afterSequence)
	if wsConn.Query("read_only") == "1" && baseline.HasControl {
		terminal.releaseControl(subscription.ID)
		baseline.HasControl = false
	} else if wsConn.Query("take_control") == "1" {
		terminal.takeControl(subscription.ID)
		baseline.HasControl = true
	}
	defer terminal.unsubscribe(subscription)
	var writeMu sync.Mutex
	writeEvent := func(event nativeTerminalEvent) error {
		payload, _ := json.Marshal(struct {
			Type       string `json:"type"`
			Sequence   uint64 `json:"sequence"`
			Data       string `json:"data,omitempty"`
			HasControl bool   `json:"hasControl"`
		}{Type: event.Type, Sequence: event.Sequence, Data: string(event.Data), HasControl: event.HasControl})
		writeMu.Lock()
		defer writeMu.Unlock()
		return wsConn.WriteMessage(websocket.TextMessage, payload)
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
			terminal.takeControl(subscription.ID)
		case "release_control":
			terminal.releaseControl(subscription.ID)
		case "resize":
			var size struct {
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if json.Unmarshal([]byte(message.Data), &size) == nil && terminal.hasControl(subscription.ID) {
				_ = terminal.resize(size.Cols, size.Rows)
			}
		case "ping":
			writeMu.Lock()
			_ = wsConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
			writeMu.Unlock()
		}
	}
}
