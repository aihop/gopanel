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
	// 执行期间对话里原先只有一个转圈的「运行中」，用户不知道 AI 在做什么，
	// 也判断不了要不要等下去。这两个字段回答「此刻在干什么」：
	// Kind 是活动种类（command/file/tool/search/thinking），文案由前端 i18n 决定；
	// Activity 是细节，比如正在执行的命令或正在改的文件。
	ActivityKind string `json:"activityKind,omitempty"`
	Activity     string `json:"activity,omitempty"`
}

type conversationStreamSession struct {
	mu           sync.Mutex
	runID        uint
	text         string
	status       string
	activityKind string
	activity     string
	subs         map[chan conversationStreamPayload]struct{}
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
	return conversationStreamPayload{
		Type: "snapshot", RunID: current.runID, Content: current.text,
		Status: current.status, ActivityKind: current.activityKind, Activity: current.activity,
	}
}

func (hub *conversationStreamHub) Subscribe(sessionID uint) (conversationStreamPayload, <-chan conversationStreamPayload, func()) {
	current := hub.session(sessionID)
	events := make(chan conversationStreamPayload, 256)
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
	// 新一轮开始，上一轮残留的活动状态要清掉。
	current.activityKind = ""
	current.activity = ""
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
	activityKind := current.activityKind
	activity := current.activity
	current.mu.Unlock()
	if delta != "" {
		hub.broadcast(sessionID, conversationStreamPayload{
			Type: "delta", RunID: runID, Content: delta, Status: status,
			ActivityKind: activityKind, Activity: activity,
		})
		return
	}
	hub.broadcast(sessionID, conversationStreamPayload{
		Type: "snapshot", RunID: runID, Content: text, Status: status,
		ActivityKind: activityKind, Activity: activity,
	})
}

// SetActivity 更新「此刻在干什么」。用单独的事件类型推送，不动正文：
// 进度是过程信息，混进 text 会被当成对话内容一起落库。
func (hub *conversationStreamHub) SetActivity(sessionID uint, kind, activity string) {
	current := hub.session(sessionID)
	current.mu.Lock()
	if current.activityKind == kind && current.activity == activity {
		current.mu.Unlock()
		return
	}
	current.activityKind = kind
	current.activity = activity
	if current.status == "" || current.status == "idle" {
		current.status = "running"
	}
	runID := current.runID
	status := current.status
	current.mu.Unlock()
	hub.broadcast(sessionID, conversationStreamPayload{
		Type: "activity", RunID: runID, Status: status, ActivityKind: kind, Activity: activity,
	})
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
	// 收尾时清掉活动状态，否则界面会一直挂着最后那句「正在执行…」。
	current.activityKind = ""
	current.activity = ""
	payload := conversationStreamPayload{Type: "done", RunID: current.runID, Content: current.text, Status: status}
	current.mu.Unlock()
	hub.broadcast(sessionID, payload)
}

func (hub *conversationStreamHub) broadcast(sessionID uint, payload conversationStreamPayload) {
	current := hub.session(sessionID)
	current.mu.Lock()
	subscribers := make([]chan conversationStreamPayload, 0, len(current.subs))
	for subscriber := range current.subs {
		subscribers = append(subscribers, subscriber)
	}
	current.mu.Unlock()
	for _, subscriber := range subscribers {
		func() {
			defer func() { _ = recover() }()
			select {
			case subscriber <- payload:
			case <-time.After(2 * time.Second):
			}
		}()
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
	c.Status(fiber.StatusOK)
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
