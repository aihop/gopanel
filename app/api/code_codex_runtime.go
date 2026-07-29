package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

const (
	codexRuntimeTailBytes      = 128 * 1024
	codexNeedsInputIdleSeconds = 4
)

type codexRuntimeState struct {
	ResponseState        string    `json:"responseState"`
	NeedsInput           bool      `json:"needsInput"`
	AwaitingApproval     bool      `json:"awaitingApproval"`
	Model                string    `json:"model"`
	InputTokens          int64     `json:"inputTokens"`
	OutputTokens         int64     `json:"outputTokens"`
	CachedInputTokens    int64     `json:"cachedInputTokens"`
	ReasoningTokens      int64     `json:"reasoningTokens"`
	TotalTokens          int64     `json:"totalTokens"`
	LastAssistantPreview string    `json:"lastAssistantPreview"`
	UpdatedAt            time.Time `json:"updatedAt"`
	WasInterrupted       bool      `json:"wasInterrupted"`
}

type codexRuntimeEvent struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		Type           string          `json:"type"`
		Role           string          `json:"role"`
		Phase          string          `json:"phase"`
		Model          string          `json:"model"`
		ApprovalPolicy string          `json:"approval_policy"`
		Name           string          `json:"name"`
		CallID         string          `json:"call_id"`
		Message        json.RawMessage `json:"message"`
		Content        json.RawMessage `json:"content"`
		StartedAt      float64         `json:"started_at"`
		CompletedAt    float64         `json:"completed_at"`
		Info           struct {
			TotalTokenUsage codexTokenUsage `json:"total_token_usage"`
			LastTokenUsage  codexTokenUsage `json:"last_token_usage"`
		} `json:"info"`
	} `json:"payload"`
}

type codexTokenUsage struct {
	InputTokens           int64 `json:"input_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
}

func getCodexRuntimeState(session *model.AIDevSession) *codexRuntimeState {
	if session.AgentName != "codex" {
		return nil
	}
	path := findCodexRuntimePath(session)
	if path == "" {
		return &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	}
	state, err := parseCodexRuntimeFile(path, time.Now())
	if err != nil {
		return &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	}
	return state
}

func findCodexRuntimePath(session *model.AIDevSession) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	pattern := filepath.Join(homeDir, ".codex", "sessions", "*", "*", "*", "*.jsonl")
	if nativeID := strings.TrimSpace(session.NativeSessionID); nativeID != "" {
		pattern = filepath.Join(homeDir, ".codex", "sessions", "*", "*", "*", "*"+nativeID+"*.jsonl")
	}
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return ""
	}
	cleanWorkDir := filepath.Clean(session.WorkDir)
	var latestPath string
	var latestTime time.Time
	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr != nil || info.ModTime().Before(session.CreatedAt.Add(-5*time.Second)) || info.ModTime().Before(latestTime) {
			continue
		}
		nativeID, cwd, _ := readCodexSessionMeta(path)
		if session.NativeSessionID != "" && nativeID != session.NativeSessionID {
			continue
		}
		if session.NativeSessionID == "" && filepath.Clean(cwd) != cleanWorkDir {
			continue
		}
		latestPath = path
		latestTime = info.ModTime()
	}
	return latestPath
}

func parseCodexRuntimeFile(path string, now time.Time) (*codexRuntimeState, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	start := info.Size() - codexRuntimeTailBytes
	if start > 0 {
		if _, err = file.Seek(start, io.SeekStart); err != nil {
			return nil, err
		}
		reader := bufio.NewReader(file)
		if _, err = reader.ReadString('\n'); err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		return parseCodexRuntime(reader, now)
	}
	return parseCodexRuntime(file, now)
}

func parseCodexRuntime(reader io.Reader, now time.Time) (*codexRuntimeState, error) {
	state := &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	pendingCalls := make(map[string]time.Time)
	var startedAt, completedAt time.Time
	var approvalPolicy string
	activeTurn := false
	turnClosed := false
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var event codexRuntimeEvent
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		eventAt := parseCodexEventTime(event.Timestamp)
		if eventAt.After(state.UpdatedAt) {
			state.UpdatedAt = eventAt
		}
		switch event.Type {
		case "turn_context":
			if event.Payload.Model != "" {
				state.Model = event.Payload.Model
			}
			if event.Payload.ApprovalPolicy != "" {
				approvalPolicy = event.Payload.ApprovalPolicy
			}
		case "response_item":
			activeTurn = !turnClosed
			trackCodexCall(event, eventAt, pendingCalls)
			if event.Payload.Type == "message" && event.Payload.Role == "assistant" {
				if preview := codexContentText(event.Payload.Content); preview != "" {
					state.LastAssistantPreview = preview
				}
			}
		case "event_msg":
			switch event.Payload.Type {
			case "task_started":
				startedAt = codexUnixTime(event.Payload.StartedAt, eventAt)
				completedAt = time.Time{}
				activeTurn = true
				turnClosed = false
				state.WasInterrupted = false
			case "task_complete":
				completedAt = codexUnixTime(event.Payload.CompletedAt, eventAt)
				activeTurn = false
				turnClosed = true
				state.WasInterrupted = false
			case "turn_aborted":
				completedAt = codexUnixTime(event.Payload.CompletedAt, eventAt)
				activeTurn = false
				turnClosed = true
				state.WasInterrupted = true
			case "agent_message":
				activeTurn = !turnClosed
				if preview := codexRawString(event.Payload.Message); preview != "" {
					state.LastAssistantPreview = preview
				}
			case "token_count":
				activeTurn = !turnClosed
				applyCodexTokenUsage(state, event.Payload.Info.TotalTokenUsage, event.Payload.Info.LastTokenUsage)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if activeTurn || (!startedAt.IsZero() && (completedAt.IsZero() || completedAt.Before(startedAt))) {
		state.ResponseState = "responding"
		state.NeedsInput = false
	}
	if state.WasInterrupted && !completedAt.Before(startedAt) {
		state.ResponseState = "failed"
		state.NeedsInput = true
	} else if !completedAt.IsZero() && !completedAt.Before(startedAt) {
		state.ResponseState = "completed"
		state.NeedsInput = true
	}
	for _, callAt := range pendingCalls {
		if state.ResponseState == "responding" && approvalPolicy != "never" && now.Sub(callAt) >= codexNeedsInputIdleSeconds*time.Second {
			state.ResponseState = "needsInput"
			state.NeedsInput = true
			state.AwaitingApproval = true
			break
		}
	}
	return state, nil
}

func trackCodexCall(event codexRuntimeEvent, eventAt time.Time, pendingCalls map[string]time.Time) {
	callID := strings.TrimSpace(event.Payload.CallID)
	if callID == "" {
		return
	}
	switch event.Payload.Type {
	case "function_call", "custom_tool_call":
		if event.Payload.Name != "update_plan" {
			pendingCalls[callID] = eventAt
		}
	case "function_call_output", "custom_tool_call_output":
		delete(pendingCalls, callID)
	}
}

func applyCodexTokenUsage(state *codexRuntimeState, total, last codexTokenUsage) {
	usage := total
	if usage.TotalTokens == 0 && usage.InputTokens == 0 && usage.OutputTokens == 0 {
		usage = last
	}
	state.InputTokens = usage.InputTokens
	state.CachedInputTokens = usage.CachedInputTokens
	state.OutputTokens = usage.OutputTokens
	state.ReasoningTokens = usage.ReasoningOutputTokens
	state.TotalTokens = usage.TotalTokens
}

func parseCodexEventTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}

func codexUnixTime(value float64, fallback time.Time) time.Time {
	if value <= 0 {
		return fallback
	}
	seconds := int64(value)
	return time.Unix(seconds, int64((value-float64(seconds))*float64(time.Second)))
}

func codexRawString(value json.RawMessage) string {
	var text string
	_ = json.Unmarshal(value, &text)
	return strings.TrimSpace(text)
}

func codexContentText(value json.RawMessage) string {
	var content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(value, &content) != nil {
		return ""
	}
	for index := len(content) - 1; index >= 0; index-- {
		if content[index].Type == "output_text" && strings.TrimSpace(content[index].Text) != "" {
			return strings.TrimSpace(content[index].Text)
		}
	}
	return ""
}
