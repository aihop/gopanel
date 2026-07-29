package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	mu          sync.Mutex
	sessionID   uint
	command     *exec.Cmd
	ptmx        *os.File
	history     []byte
	subscribers map[chan []byte]struct{}
	done        chan struct{}
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
		subscribers: make(map[chan []byte]struct{}),
		done:        make(chan struct{}),
	}
	manager.sessions[session.ID] = terminal
	go terminal.readOutput()
	go terminal.wait(manager)
	go discoverNativeCodexSession(session, command.Process.Pid, time.Now())
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
	terminal.history = append(terminal.history, chunk...)
	if len(terminal.history) > nativeTerminalHistoryLimit {
		terminal.history = append([]byte(nil), terminal.history[len(terminal.history)-nativeTerminalHistoryLimit:]...)
	}
	for subscriber := range terminal.subscribers {
		select {
		case subscriber <- chunk:
		default:
		}
	}
	terminal.mu.Unlock()
}

func (terminal *nativeCodeTerminal) subscribe() (chan []byte, []byte) {
	subscriber := make(chan []byte, 128)
	terminal.mu.Lock()
	terminal.subscribers[subscriber] = struct{}{}
	history := append([]byte(nil), terminal.history...)
	terminal.mu.Unlock()
	return subscriber, history
}

func (terminal *nativeCodeTerminal) unsubscribe(subscriber chan []byte) {
	terminal.mu.Lock()
	if _, exists := terminal.subscribers[subscriber]; exists {
		delete(terminal.subscribers, subscriber)
		close(subscriber)
	}
	terminal.mu.Unlock()
}

func (terminal *nativeCodeTerminal) write(data []byte) error {
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
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
	for subscriber := range terminal.subscribers {
		delete(terminal.subscribers, subscriber)
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
	subscriber, history := terminal.subscribe()
	defer terminal.unsubscribe(subscriber)
	var writeMu sync.Mutex
	writeCommand := func(data []byte) error {
		payload, _ := json.Marshal(WsMsg{Type: "cmd", Data: string(data)})
		writeMu.Lock()
		defer writeMu.Unlock()
		return wsConn.WriteMessage(websocket.TextMessage, payload)
	}
	if len(history) > 0 {
		_ = writeCommand(history)
	}
	go func() {
		for output := range subscriber {
			if writeCommand(output) != nil {
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
			if terminal.write([]byte(message.Data)) != nil {
				return
			}
		case "resize":
			var size struct {
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if json.Unmarshal([]byte(message.Data), &size) == nil {
				_ = terminal.resize(size.Cols, size.Rows)
			}
		case "ping":
			writeMu.Lock()
			_ = wsConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
			writeMu.Unlock()
		}
	}
}
