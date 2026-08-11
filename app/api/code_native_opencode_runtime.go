package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	_ "github.com/glebarez/go-sqlite"
)

type nativeOpenCodeMessage struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Data      nativeOpenCodeMessageData
	Parts     []nativeOpenCodePart
}

type nativeOpenCodeMessageData struct {
	Role       string          `json:"role"`
	ModelID    string          `json:"modelID"`
	ProviderID string          `json:"providerID"`
	Finish     string          `json:"finish"`
	Error      json.RawMessage `json:"error"`
	Time       struct {
		Created   int64 `json:"created"`
		Completed int64 `json:"completed"`
	} `json:"time"`
	Tokens struct {
		Total     int64 `json:"total"`
		Input     int64 `json:"input"`
		Output    int64 `json:"output"`
		Reasoning int64 `json:"reasoning"`
		Cache     struct {
			Read  int64 `json:"read"`
			Write int64 `json:"write"`
		} `json:"cache"`
	} `json:"tokens"`
}

type nativeOpenCodePart struct {
	ID        string
	CreatedAt time.Time
	Type      string
	Text      string
	Tool      string
	State     struct {
		Status string `json:"status"`
	} `json:"state"`
}

func nativeOpenCodeDatabasePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".local", "share", "opencode", "opencode.db")
}

func readNativeOpenCodeMessages(nativeSessionID string) ([]nativeOpenCodeMessage, error) {
	databasePath := nativeOpenCodeDatabasePath()
	if strings.TrimSpace(nativeSessionID) == "" || databasePath == "" {
		return nil, nil
	}
	if _, err := os.Stat(databasePath); err != nil {
		return nil, nil
	}
	database, err := sql.Open("sqlite", nativeOpenCodeDatabaseDSN(databasePath))
	if err != nil {
		return nil, err
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rows, err := database.QueryContext(ctx, `
		SELECT m.id, m.time_created, m.time_updated, m.data,
		       COALESCE(p.id, ''), COALESCE(p.time_created, 0), COALESCE(p.data, '')
		FROM message m
		LEFT JOIN part p ON p.message_id = m.id
		WHERE m.session_id = ?
		ORDER BY m.time_created, m.id, p.time_created, p.id`, nativeSessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := make([]nativeOpenCodeMessage, 0)
	indexes := make(map[string]int)
	for rows.Next() {
		var messageID, messageData, partID, partData string
		var messageCreated, messageUpdated, partCreated int64
		if err := rows.Scan(&messageID, &messageCreated, &messageUpdated, &messageData, &partID, &partCreated, &partData); err != nil {
			return nil, err
		}
		index, exists := indexes[messageID]
		if !exists {
			var data nativeOpenCodeMessageData
			if json.Unmarshal([]byte(messageData), &data) != nil {
				continue
			}
			index = len(messages)
			indexes[messageID] = index
			messages = append(messages, nativeOpenCodeMessage{
				ID: messageID, CreatedAt: openCodeTime(messageCreated), UpdatedAt: openCodeTime(messageUpdated), Data: data,
			})
		}
		if partID == "" || partData == "" {
			continue
		}
		var part nativeOpenCodePart
		if json.Unmarshal([]byte(partData), &part) != nil {
			continue
		}
		part.ID = partID
		part.CreatedAt = openCodeTime(partCreated)
		messages[index].Parts = append(messages[index].Parts, part)
	}
	return messages, rows.Err()
}

func findNativeOpenCodeSessionInDatabase(workDir string, startedAt time.Time) (string, error) {
	databasePath := nativeOpenCodeDatabasePath()
	if databasePath == "" {
		return "", nil
	}
	if _, err := os.Stat(databasePath); err != nil {
		return "", nil
	}
	database, err := sql.Open("sqlite", nativeOpenCodeDatabaseDSN(databasePath))
	if err != nil {
		return "", err
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var nativeSessionID string
	err = database.QueryRowContext(ctx, `
		SELECT id FROM session
		WHERE directory = ? AND time_created >= ?
		ORDER BY time_created DESC LIMIT 1`, filepath.Clean(workDir), startedAt.Add(-2*time.Second).UnixMilli()).Scan(&nativeSessionID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return nativeSessionID, err
}

func getNativeOpenCodeMessages(session *model.AIDevSession) ([]*model.AIMessage, error) {
	if session == nil {
		return nil, nil
	}
	nativeMessages, err := readNativeOpenCodeMessages(session.NativeSessionID)
	if err != nil {
		return nil, err
	}
	messages := make([]*model.AIMessage, 0, len(nativeMessages))
	for _, nativeMessage := range nativeMessages {
		role := ""
		switch nativeMessage.Data.Role {
		case "user":
			role = "user"
		case "assistant":
			role = "agent"
		}
		content := nativeOpenCodeText(nativeMessage.Parts)
		if role == "" || content == "" {
			continue
		}
		messages = append(messages, &model.AIMessage{
			CreatedAt: nativeMessage.CreatedAt, SessionID: session.ID, TaskID: session.LastTaskID,
			Role: role, Content: content, NativeID: nativeMessage.ID,
		})
	}
	return messages, nil
}

func getNativeOpenCodeRuntimeState(session *model.AIDevSession) *codexRuntimeState {
	if session == nil || strings.TrimSpace(session.NativeSessionID) == "" {
		return &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	}
	messages, err := readNativeOpenCodeMessages(session.NativeSessionID)
	if err != nil || len(messages) == 0 {
		return &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	}
	return parseNativeOpenCodeRuntime(messages, codeNativeTerminals.running(session.ID))
}

func parseNativeOpenCodeRuntime(messages []nativeOpenCodeMessage, terminalRunning bool) *codexRuntimeState {
	state := &codexRuntimeState{ResponseState: "idle", NeedsInput: true}
	activeTurn := false
	totalTokens := int64(0)
	for _, message := range messages {
		updatedAt := message.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = message.CreatedAt
		}
		if updatedAt.After(state.UpdatedAt) {
			state.UpdatedAt = updatedAt
		}
		switch message.Data.Role {
		case "user":
			activeTurn = true
			state.WasInterrupted = false
		case "assistant":
			activeTurn = message.Data.Time.Completed == 0 && message.Data.Finish == "" && !nativeOpenCodeMessageHasError(message.Data)
			if message.Data.ModelID != "" {
				state.Model = message.Data.ModelID
			}
			state.InputTokens += message.Data.Tokens.Input
			state.OutputTokens += message.Data.Tokens.Output
			state.ReasoningTokens += message.Data.Tokens.Reasoning
			state.CachedInputTokens += message.Data.Tokens.Cache.Read
			if message.Data.Tokens.Total > 0 {
				totalTokens += message.Data.Tokens.Total
			} else {
				totalTokens += message.Data.Tokens.Input + message.Data.Tokens.Output + message.Data.Tokens.Reasoning
			}
			if text := nativeOpenCodeText(message.Parts); text != "" {
				state.LastAssistantPreview = buildTimelineContent(text)
			}
			if nativeOpenCodeMessageHasError(message.Data) {
				state.WasInterrupted = true
				activeTurn = false
			}
		}
	}
	state.TotalTokens = totalTokens
	if state.WasInterrupted {
		state.ResponseState = "failed"
	} else if activeTurn {
		state.ResponseState = "responding"
		state.NeedsInput = false
		if !terminalRunning {
			state.ResponseState = "failed"
			state.NeedsInput = true
			state.WasInterrupted = true
		}
	} else {
		state.ResponseState = "completed"
	}
	return state
}

func nativeOpenCodeText(parts []nativeOpenCodePart) string {
	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
			texts = append(texts, strings.TrimSpace(part.Text))
		}
	}
	return strings.TrimSpace(strings.Join(texts, "\n"))
}

func openCodeTime(milliseconds int64) time.Time {
	if milliseconds <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(milliseconds)
}

func nativeOpenCodeMessageHasError(message nativeOpenCodeMessageData) bool {
	value := strings.TrimSpace(string(message.Error))
	return value != "" && value != "null"
}

func nativeOpenCodeDatabaseDSN(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: "mode=ro"}).String()
}

func nativeOpenCodeDatabaseHasCredentials() bool {
	path := nativeOpenCodeDatabasePath()
	if path == "" {
		return false
	}
	database, err := sql.Open("sqlite", nativeOpenCodeDatabaseDSN(path))
	if err != nil {
		return false
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, table := range []string{"credential", "account"} {
		var count int
		if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err == nil && count > 0 {
			return true
		}
	}
	return false
}

func nativeOpenCodeAuthFileHasCredentials() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	content, err := os.ReadFile(filepath.Join(homeDir, ".local", "share", "opencode", "auth.json"))
	if err != nil {
		return false
	}
	var values any
	if json.Unmarshal(content, &values) != nil {
		return false
	}
	switch value := values.(type) {
	case []any:
		return len(value) > 0
	case map[string]any:
		return len(value) > 0
	default:
		return false
	}
}
