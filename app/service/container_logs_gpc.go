package service

import (
	"bytes"
	"context"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/docker"
	"os"
	"strings"
	"time"
)

func (u *ContainerService) streamContainerLogsViaGPC(wsConn *websocket.Conn, containerID, since, tail, runtimeHost string, follow bool) (bool, error) {
	ctx := context.Background()
	if !docker.IsPodmanRuntime(ctx) {
		return false, nil
	}
	meta, err := inspectPodmanContainerLogMeta(containerID, runtimeHost)
	if err != nil {
		return false, nil
	}
	if !strings.EqualFold(meta.LogDriver, "journald") {
		return false, nil
	}
	if !follow {
		batch, err := fetchPodmanJournalLogsViaGPC(meta, since, tail, runtimeHost, "")
		if err != nil {
			return true, err
		}
		if len(batch.Lines) > 0 {
			payload := bytes.ToValidUTF8([]byte(strings.Join(batch.Lines, "\n")), []byte("?"))
			if err := wsConn.WriteMessage(websocket.TextMessage, payload); err != nil {
				global.LOG.Errorf("send gpc journal logs to ws failed, err: %v", err)
			}
		}
		return true, nil
	}
	stopCh := make(chan struct{})
	go func() {
		defer close(stopCh)
		_, wsData, _ := wsConn.ReadMessage()
		if string(wsData) == "close conn" {
			return
		}
	}()
	cursor := ""
	firstRound := true
	for {
		select {
		case <-stopCh:
			return true, nil
		default:
		}
		currentTail := tail
		currentSince := since
		if !firstRound {
			currentTail = "0"
			currentSince = "all"
		}
		batch, err := fetchPodmanJournalLogsViaGPC(meta, currentSince, currentTail, runtimeHost, cursor)
		if err != nil {
			return true, err
		}
		if batch.Cursor != "" {
			cursor = batch.Cursor
		}
		for _, line := range batch.Lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			payload := bytes.ToValidUTF8([]byte(line), []byte("?"))
			if err := wsConn.WriteMessage(websocket.TextMessage, payload); err != nil {
				global.LOG.Errorf("send gpc journal log line to ws failed, err: %v", err)
				return true, nil
			}
		}
		firstRound = false
		select {
		case <-stopCh:
			return true, nil
		case <-time.After(1500 * time.Millisecond):
		}
	}
}
func (u *ContainerService) downloadContainerLogsViaGPC(containerID, since, tail, runtimeHost string) (string, bool, error) {
	ctx := context.Background()
	if !docker.IsPodmanRuntime(ctx) {
		return "", false, nil
	}
	meta, err := inspectPodmanContainerLogMeta(containerID, runtimeHost)
	if err != nil {
		return "", false, nil
	}
	if !strings.EqualFold(meta.LogDriver, "journald") {
		return "", false, nil
	}
	batch, err := fetchPodmanJournalLogsViaGPC(meta, since, tail, runtimeHost, "")
	if err != nil {
		return "", true, err
	}
	tempFile, err := os.CreateTemp("", "container_journal_*.txt")
	if err != nil {
		return "", true, err
	}
	defer tempFile.Close()
	if len(batch.Lines) > 0 {
		if _, err := tempFile.WriteString(strings.Join(batch.Lines, "\n") + "\n"); err != nil {
			return "", true, err
		}
	}
	return tempFile.Name(), true, nil
}
