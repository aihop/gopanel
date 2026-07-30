package api

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

const nativeHistoryDuplicateWindow = 5 * time.Minute

func getNativeCodexMessages(session *model.AIDevSession) ([]*model.AIMessage, error) {
	if session == nil || strings.TrimSpace(session.NativeSessionID) == "" {
		return nil, nil
	}
	path := findCodexRuntimePath(session)
	if path == "" {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseNativeCodexMessages(file, session.ID, session.LastTaskID)
}

func parseNativeCodexMessages(reader io.Reader, sessionID, taskID uint) ([]*model.AIMessage, error) {
	messages := make([]*model.AIMessage, 0)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var event codexRuntimeEvent
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Type != "event_msg" {
			continue
		}
		role := ""
		switch event.Payload.Type {
		case "user_message":
			role = "user"
		case "agent_message":
			role = "agent"
		default:
			continue
		}
		content := nativeCodexMessageText(event.Payload.Message)
		if content == "" {
			continue
		}
		messages = append(messages, &model.AIMessage{
			CreatedAt: parseCodexEventTime(event.Timestamp),
			SessionID: sessionID,
			TaskID:    taskID,
			Role:      role,
			Content:   content,
		})
	}
	return messages, scanner.Err()
}

func nativeCodexMessageText(value json.RawMessage) string {
	var text string
	if json.Unmarshal(value, &text) != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

func mergeCodeHistoryMessages(databaseMessages, nativeMessages []*model.AIMessage) []*model.AIMessage {
	merged := append([]*model.AIMessage{}, databaseMessages...)
	for _, nativeMessage := range nativeMessages {
		if nativeMessage == nil || isDuplicateHistoryMessage(databaseMessages, nativeMessage) {
			continue
		}
		merged = append(merged, nativeMessage)
	}
	sort.SliceStable(merged, func(left, right int) bool {
		return merged[left].CreatedAt.Before(merged[right].CreatedAt)
	})
	var nextID uint
	for _, message := range merged {
		if message.ID > nextID {
			nextID = message.ID
		}
	}
	for _, message := range merged {
		if message.ID == 0 {
			nextID++
			message.ID = nextID
		}
	}
	return merged
}

func isDuplicateHistoryMessage(databaseMessages []*model.AIMessage, candidate *model.AIMessage) bool {
	for _, message := range databaseMessages {
		if message == nil || message.Role != candidate.Role || strings.TrimSpace(message.Content) != candidate.Content {
			continue
		}
		if message.CreatedAt.IsZero() || candidate.CreatedAt.IsZero() {
			return true
		}
		difference := message.CreatedAt.Sub(candidate.CreatedAt)
		if difference < 0 {
			difference = -difference
		}
		if difference <= nativeHistoryDuplicateWindow {
			return true
		}
	}
	return false
}
