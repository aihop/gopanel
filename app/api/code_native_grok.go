package api

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

type nativeGrokUpdate struct {
	Timestamp int64  `json:"timestamp"`
	Method    string `json:"method"`
	Params    struct {
		SessionID string `json:"sessionId"`
		Update    struct {
			SessionUpdate string `json:"sessionUpdate"`
			StopReason    string `json:"stop_reason"`
			Content       struct {
				Text string `json:"text"`
			} `json:"content"`
			Usage map[string]any `json:"usage"`
		} `json:"update"`
	} `json:"params"`
}

type nativeGrokSummary struct {
	CurrentModelID string `json:"current_model_id"`
}

func getNativeGrokRuntimeState(session *model.AIDevSession) *codexRuntimeState {
	state := &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	dir := findNativeGrokSessionDir(session)
	if dir == "" {
		if session != nil && codeNativeTerminals.running(session.ID) {
			state.ResponseState = "responding"
			state.NeedsInput = false
		}
		return state
	}
	if content, err := os.ReadFile(filepath.Join(dir, "summary.json")); err == nil {
		var summary nativeGrokSummary
		if json.Unmarshal(content, &summary) == nil {
			state.Model = summary.CurrentModelID
		}
	}
	file, err := os.Open(filepath.Join(dir, "updates.jsonl"))
	if err != nil {
		return state
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var event nativeGrokUpdate
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Params.SessionID != strings.TrimSpace(session.NativeSessionID) {
			continue
		}
		if event.Timestamp > 0 {
			state.UpdatedAt = time.UnixMilli(event.Timestamp)
		}
		switch event.Params.Update.SessionUpdate {
		case "agent_message_chunk":
			state.ResponseState = "responding"
			state.NeedsInput = false
			if text := strings.TrimSpace(event.Params.Update.Content.Text); text != "" {
				state.LastAssistantPreview = truncateRunes(text, 160)
			}
		case "user_message_chunk", "agent_thought_chunk", "tool_call", "tool_call_update", "plan", "retry_state":
			state.ResponseState = "responding"
			state.NeedsInput = false
		case "turn_completed":
			state.InputTokens = firstCodeInt(event.Params.Update.Usage, "inputTokens", "input_tokens")
			state.OutputTokens = firstCodeInt(event.Params.Update.Usage, "outputTokens", "output_tokens")
			state.CachedInputTokens = firstCodeInt(event.Params.Update.Usage, "cachedReadTokens", "cached_input_tokens")
			state.ReasoningTokens = firstCodeInt(event.Params.Update.Usage, "reasoningTokens", "reasoning_tokens")
			state.TotalTokens = firstCodeInt(event.Params.Update.Usage, "totalTokens", "total_tokens")
			if event.Params.Update.StopReason == "end_turn" {
				state.ResponseState = "completed"
				state.NeedsInput = true
			} else {
				state.ResponseState = "failed"
				state.NeedsInput = true
				state.WasInterrupted = true
			}
		}
	}
	return state
}

func getNativeGrokMessages(session *model.AIDevSession) ([]*model.AIMessage, error) {
	dir := findNativeGrokSessionDir(session)
	if dir == "" {
		return nil, nil
	}
	file, err := os.Open(filepath.Join(dir, "updates.jsonl"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	messages := make([]*model.AIMessage, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var event nativeGrokUpdate
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Params.SessionID != strings.TrimSpace(session.NativeSessionID) {
			continue
		}
		role := ""
		switch event.Params.Update.SessionUpdate {
		case "user_message_chunk":
			role = "user"
		case "agent_message_chunk":
			role = "agent"
		}
		content := strings.TrimSpace(event.Params.Update.Content.Text)
		if role == "" || content == "" {
			continue
		}
		createdAt := time.UnixMilli(event.Timestamp)
		messages = append(messages, &model.AIMessage{
			CreatedAt: createdAt, SessionID: session.ID, TaskID: session.LastTaskID,
			Role: role, Content: content,
			NativeID: nativeCodexMessageID("", role, createdAt.UTC().Format(time.RFC3339Nano), content),
		})
	}
	return messages, scanner.Err()
}

func findNativeGrokSessionDir(session *model.AIDevSession) string {
	if session == nil || strings.TrimSpace(session.NativeSessionID) == "" {
		return ""
	}
	return findNativeGrokSessionDirByID(session.NativeSessionID)
}

func nativeGrokSessionExists(nativeSessionID string) bool {
	return findNativeGrokSessionDirByID(nativeSessionID) != ""
}

func findNativeGrokSessionDirByID(nativeSessionID string) string {
	nativeSessionID = strings.TrimSpace(nativeSessionID)
	if nativeSessionID == "" {
		return ""
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	paths, _ := filepath.Glob(filepath.Join(homeDir, ".grok", "sessions", "*", nativeSessionID))
	for _, path := range paths {
		if info, statErr := os.Stat(filepath.Join(path, "summary.json")); statErr == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
