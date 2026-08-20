package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type conversationStreamPayload struct {
	Type    string `json:"type"`
	RunID   uint   `json:"runId,omitempty"`
	Content string `json:"content"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type conversationStreamSession struct {
	mu     sync.Mutex
	runID  uint
	text   string
	status string
	subs   map[chan conversationStreamPayload]struct{}
}

type conversationStreamHub struct {
	mu       sync.Mutex
	sessions map[uint]*conversationStreamSession
}

var codeConversationStreams = newConversationStreamHub()

func newConversationStreamHub() *conversationStreamHub {
	return &conversationStreamHub{sessions: map[uint]*conversationStreamSession{}}
}

func (hub *conversationStreamHub) session(sessionID uint) *conversationStreamSession {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	current := hub.sessions[sessionID]
	if current == nil {
		current = &conversationStreamSession{
			status: "idle",
			subs:   map[chan conversationStreamPayload]struct{}{},
		}
		hub.sessions[sessionID] = current
	}
	return current
}

func (hub *conversationStreamHub) snapshot(sessionID uint) conversationStreamPayload {
	current := hub.session(sessionID)
	current.mu.Lock()
	defer current.mu.Unlock()
	return conversationStreamPayload{Type: "snapshot", RunID: current.runID, Content: current.text, Status: current.status}
}

func (hub *conversationStreamHub) Subscribe(sessionID uint) (conversationStreamPayload, <-chan conversationStreamPayload, func()) {
	current := hub.session(sessionID)
	events := make(chan conversationStreamPayload, 16)
	current.mu.Lock()
	current.subs[events] = struct{}{}
	current.mu.Unlock()
	return hub.snapshot(sessionID), events, func() {
		current.mu.Lock()
		delete(current.subs, events)
		current.mu.Unlock()
		close(events)
	}
}

func (hub *conversationStreamHub) Begin(sessionID, runID uint) {
	current := hub.session(sessionID)
	current.mu.Lock()
	current.runID = runID
	current.text = ""
	current.status = "running"
	current.mu.Unlock()
	hub.broadcast(sessionID, conversationStreamPayload{Type: "snapshot", RunID: runID, Status: "running"})
}

func (hub *conversationStreamHub) SetText(sessionID uint, text string, delta string) {
	current := hub.session(sessionID)
	current.mu.Lock()
	current.text = text
	if current.status == "" || current.status == "idle" {
		current.status = "running"
	}
	runID := current.runID
	status := current.status
	current.mu.Unlock()
	if delta != "" {
		hub.broadcast(sessionID, conversationStreamPayload{Type: "delta", RunID: runID, Content: delta, Status: status})
		return
	}
	hub.broadcast(sessionID, conversationStreamPayload{Type: "snapshot", RunID: runID, Content: text, Status: status})
}

func (hub *conversationStreamHub) Finish(sessionID uint, status, content string) {
	current := hub.session(sessionID)
	current.mu.Lock()
	if content != "" {
		current.text = content
	}
	if status == "" {
		status = "completed"
	}
	current.status = status
	payload := conversationStreamPayload{Type: "done", RunID: current.runID, Content: current.text, Status: status}
	current.mu.Unlock()
	hub.broadcast(sessionID, payload)
}

func (hub *conversationStreamHub) broadcast(sessionID uint, payload conversationStreamPayload) {
	current := hub.session(sessionID)
	current.mu.Lock()
	defer current.mu.Unlock()
	for subscriber := range current.subs {
		select {
		case subscriber <- payload:
		default:
		}
	}
}

type jsonLineSplitter struct {
	rest []byte
}

func (splitter *jsonLineSplitter) Push(data []byte) [][]byte {
	splitter.rest = append(splitter.rest, data...)
	var lines [][]byte
	for {
		index := bytes.IndexByte(splitter.rest, '\n')
		if index < 0 {
			break
		}
		line := bytes.TrimSpace(splitter.rest[:index])
		splitter.rest = splitter.rest[index+1:]
		if len(line) == 0 {
			continue
		}
		lines = append(lines, append([]byte(nil), line...))
	}
	return lines
}

type conversationOutputWriter struct {
	inner      *boundedCodeOutput
	executorID string
	sessionID  uint
	splitter   jsonLineSplitter
	snapshot   string
}

func (writer *conversationOutputWriter) Write(data []byte) (int, error) {
	written, err := writer.inner.Write(data)
	for _, line := range writer.splitter.Push(data[:written]) {
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			continue
		}
		text, replace := conversationAssistantUpdate(writer.executorID, event)
		if text == "" && !replace {
			continue
		}
		delta := text
		if replace {
			writer.snapshot = text
			delta = ""
		} else {
			writer.snapshot += text
		}
		codeConversationStreams.SetText(writer.sessionID, writer.snapshot, delta)
	}
	return written, err
}

func writeConversationSSE(writer *bufio.Writer, event string, payload conversationStreamPayload) error {
	payload.Type = event
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event, body); err != nil {
		return err
	}
	return writer.Flush()
}

func StreamCodeConversation(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")
	snapshot, events, cancel := codeConversationStreams.Subscribe(session.ID)
	c.RequestCtx().SetBodyStreamWriter(func(writer *bufio.Writer) {
		defer cancel()
		if writeConversationSSE(writer, "snapshot", snapshot) != nil {
			return
		}
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.Context().Done():
				return
			case <-ticker.C:
				if _, err := writer.WriteString(": ping\n\n"); err != nil {
					return
				}
				if err := writer.Flush(); err != nil {
					return
				}
			case event, ok := <-events:
				if !ok {
					return
				}
				if writeConversationSSE(writer, event.Type, event) != nil {
					return
				}
			}
		}
	})
	return nil
}
