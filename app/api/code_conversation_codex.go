package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func followCodexConversationRollout(ctx context.Context, session *model.AIDevSession, sessionID uint, nativeSessionID string, startedAt time.Time) {
	if session == nil || sessionID == 0 {
		return
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	var file *os.File
	var offset int64
	snapshot := ""
	seen := map[string]struct{}{}
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if file == nil {
				path := findCodexRolloutForFollow(session, nativeSessionID, startedAt)
				if path == "" {
					continue
				}
				handle, err := os.Open(path)
				if err != nil {
					continue
				}
				info, statErr := handle.Stat()
				if statErr != nil {
					_ = handle.Close()
					continue
				}
				file = handle
				offset = info.Size()
			}
			next, updated := readCodexRolloutUpdates(file, offset, snapshot, seen)
			offset = updated.offset
			if next == snapshot {
				continue
			}
			snapshot = next
			codeConversationStreams.SetText(sessionID, snapshot, "")
		}
	}
}

func findCodexRolloutForFollow(session *model.AIDevSession, nativeSessionID string, startedAt time.Time) string {
	probe := *session
	if strings.TrimSpace(nativeSessionID) != "" {
		probe.NativeSessionID = nativeSessionID
	}
	path := findCodexRuntimePath(&probe)
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil || info.ModTime().Before(startedAt.Add(-3*time.Second)) {
		return ""
	}
	return path
}

type codexRolloutCursor struct {
	offset int64
}

func readCodexRolloutUpdates(file *os.File, offset int64, snapshot string, seen map[string]struct{}) (string, codexRolloutCursor) {
	info, err := file.Stat()
	if err != nil || info.Size() <= offset {
		return snapshot, codexRolloutCursor{offset: offset}
	}
	if _, err = file.Seek(offset, io.SeekStart); err != nil {
		return snapshot, codexRolloutCursor{offset: offset}
	}
	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			offset += int64(len(line))
			snapshot = applyCodexLiveEvent(snapshot, seen, bytes.TrimSpace(line))
		}
		if readErr != nil {
			break
		}
	}
	return snapshot, codexRolloutCursor{offset: offset}
}

func applyCodexLiveEvent(snapshot string, seen map[string]struct{}, line []byte) string {
	if len(line) == 0 {
		return snapshot
	}
	var event map[string]any
	if json.Unmarshal(line, &event) != nil {
		return snapshot
	}
	text, _ := conversationAssistantUpdate("codex", event)
	text = strings.TrimSpace(text)
	if text == "" {
		return snapshot
	}
	if _, exists := seen[text]; exists {
		return snapshot
	}
	seen[text] = struct{}{}
	if snapshot == "" || strings.Contains(text, snapshot) {
		return text
	}
	if strings.Contains(snapshot, text) {
		return snapshot
	}
	return snapshot + "\n\n" + text
}
