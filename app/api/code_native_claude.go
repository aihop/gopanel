package api

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

type nativeClaudeEvent struct {
	Type      string              `json:"type"`
	UUID      string              `json:"uuid"`
	Timestamp string              `json:"timestamp"`
	SessionID string              `json:"sessionId"`
	Origin    json.RawMessage     `json:"origin"`
	Message   nativeClaudeMessage `json:"message"`
}

type nativeClaudeMessage struct {
	ID         string          `json:"id"`
	Role       string          `json:"role"`
	Model      string          `json:"model"`
	Content    json.RawMessage `json:"content"`
	StopReason string          `json:"stop_reason"`
	Usage      struct {
		InputTokens         int64 `json:"input_tokens"`
		OutputTokens        int64 `json:"output_tokens"`
		CacheReadTokens     int64 `json:"cache_read_input_tokens"`
		CacheCreationTokens int64 `json:"cache_creation_input_tokens"`
	} `json:"usage"`
}

type nativeClaudeContent struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	ID        string `json:"id"`
	ToolUseID string `json:"tool_use_id"`
	IsError   bool   `json:"is_error"`
}

func findNativeClaudeRuntimePath(session *model.AIDevSession) string {
	if session == nil || strings.TrimSpace(session.NativeSessionID) == "" {
		return ""
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	paths, err := filepath.Glob(filepath.Join(homeDir, ".claude", "projects", "*", session.NativeSessionID+".jsonl"))
	if err != nil || len(paths) == 0 {
		return ""
	}
	var latestPath string
	var latestTime time.Time
	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr == nil && info.ModTime().After(latestTime) {
			latestPath = path
			latestTime = info.ModTime()
		}
	}
	return latestPath
}

func getNativeClaudeMessages(session *model.AIDevSession) ([]*model.AIMessage, error) {
	path := findNativeClaudeRuntimePath(session)
	if path == "" {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseNativeClaudeMessages(file, session.ID, session.LastTaskID)
}

func parseNativeClaudeMessages(reader io.Reader, sessionID, taskID uint) ([]*model.AIMessage, error) {
	messages := make([]*model.AIMessage, 0)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var event nativeClaudeEvent
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		role, content := nativeClaudeVisibleMessage(&event)
		if role == "" || content == "" {
			continue
		}
		messages = append(messages, &model.AIMessage{
			CreatedAt: parseCodexEventTime(event.Timestamp),
			SessionID: sessionID,
			TaskID:    taskID,
			Role:      role,
			Content:   content,
			NativeID:  nativeCodexMessageID(event.UUID, role, event.Timestamp, content),
		})
	}
	return messages, scanner.Err()
}

func nativeClaudeVisibleMessage(event *nativeClaudeEvent) (string, string) {
	if event == nil {
		return "", ""
	}
	switch event.Type {
	case "user":
		if event.Message.Role != "user" || !nativeClaudeHumanInput(event) {
			return "", ""
		}
		return "user", nativeClaudeText(event.Message.Content)
	case "assistant":
		if event.Message.Role != "assistant" {
			return "", ""
		}
		return "agent", nativeClaudeText(event.Message.Content)
	default:
		return "", ""
	}
}

func nativeClaudeHumanInput(event *nativeClaudeEvent) bool {
	var origin struct {
		Kind string `json:"kind"`
	}
	if len(event.Origin) > 0 && json.Unmarshal(event.Origin, &origin) == nil && origin.Kind != "" {
		return origin.Kind == "human"
	}
	var items []nativeClaudeContent
	if json.Unmarshal(event.Message.Content, &items) == nil {
		for _, item := range items {
			if item.Type == "tool_result" {
				return false
			}
		}
	}
	return true
}

func nativeClaudeText(content json.RawMessage) string {
	var text string
	if json.Unmarshal(content, &text) == nil {
		return strings.TrimSpace(text)
	}
	var items []nativeClaudeContent
	if json.Unmarshal(content, &items) != nil {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item.Type == "text" && strings.TrimSpace(item.Text) != "" {
			parts = append(parts, strings.TrimSpace(item.Text))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func getNativeClaudeRuntimeState(session *model.AIDevSession) *codexRuntimeState {
	path := findNativeClaudeRuntimePath(session)
	if path == "" {
		return &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	}
	file, err := os.Open(path)
	if err != nil {
		return &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	}
	defer file.Close()
	state, err := parseNativeClaudeRuntime(file, time.Now(), codeNativeTerminals.running(session.ID), session.ApprovalPolicy)
	if err != nil {
		return &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	}
	return state
}

func parseNativeClaudeRuntime(reader io.Reader, now time.Time, terminalRunning bool, approvalPolicy string) (*codexRuntimeState, error) {
	state := &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	pendingTools := make(map[string]time.Time)
	usageByMessage := make(map[string]nativeClaudeMessage)
	activeTurn := false
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var event nativeClaudeEvent
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		eventAt := parseCodexEventTime(event.Timestamp)
		if eventAt.After(state.UpdatedAt) {
			state.UpdatedAt = eventAt
		}
		if event.Type == "user" && event.Message.Role == "user" {
			if nativeClaudeHumanInput(&event) {
				clear(pendingTools)
				activeTurn = true
				state.WasInterrupted = false
			} else {
				for _, item := range nativeClaudeContentItems(event.Message.Content) {
					if item.Type == "tool_result" {
						delete(pendingTools, item.ToolUseID)
					}
				}
			}
			continue
		}
		if event.Type != "assistant" || event.Message.Role != "assistant" {
			continue
		}
		activeTurn = event.Message.StopReason != "end_turn"
		if event.Message.Model != "" {
			state.Model = event.Message.Model
		}
		if event.Message.ID != "" {
			usageByMessage[event.Message.ID] = event.Message
		}
		for _, item := range nativeClaudeContentItems(event.Message.Content) {
			switch item.Type {
			case "text":
				if strings.TrimSpace(item.Text) != "" {
					state.LastAssistantPreview = buildTimelineContent(item.Text)
				}
			case "tool_use":
				if item.ID != "" {
					pendingTools[item.ID] = eventAt
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for _, message := range usageByMessage {
		state.InputTokens += message.Usage.InputTokens + message.Usage.CacheCreationTokens
		state.OutputTokens += message.Usage.OutputTokens
		state.CachedInputTokens += message.Usage.CacheReadTokens
	}
	state.TotalTokens = state.InputTokens + state.OutputTokens
	if activeTurn {
		state.ResponseState = "responding"
		state.NeedsInput = false
		if !terminalRunning {
			state.ResponseState = "failed"
			state.NeedsInput = true
			state.WasInterrupted = true
		} else if approvalPolicy != codeApprovalPolicyFullAuto {
			for _, toolAt := range pendingTools {
				if now.Sub(toolAt) >= codexNeedsInputIdleSeconds*time.Second {
					state.ResponseState = "needsInput"
					state.NeedsInput = true
					state.AwaitingApproval = true
					break
				}
			}
		}
	} else if !state.UpdatedAt.IsZero() {
		state.ResponseState = "completed"
	}
	return state, nil
}

func nativeClaudeContentItems(content json.RawMessage) []nativeClaudeContent {
	var items []nativeClaudeContent
	_ = json.Unmarshal(content, &items)
	return items
}
