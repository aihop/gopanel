package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/global"
)

// DownloadFileEvent 远程下载任务日志事件
type DownloadFileEvent struct {
	Type    string // log | progress | status | eof
	Message string
	Status  string // running | success | failed
	Percent float64
}

// DownloadFileLogger 远程下载任务的 Logger
type DownloadFileLogger struct {
	Key       string
	logs      []string
	listeners []chan DownloadFileEvent
	status    string
	mu        sync.RWMutex
	file      *os.File
}

var (
	downloadLoggers   = make(map[string]*DownloadFileLogger)
	downloadLoggersMu sync.RWMutex
)

func getDownloadLogPath(key string) string {
	logDir := filepath.Join(global.CONF.System.TmpDir, "download_logs")
	_ = os.MkdirAll(logDir, 0o755)
	return filepath.Join(logDir, key)
}

func GetDownloadLogger(key string) *DownloadFileLogger {
	downloadLoggersMu.Lock()
	defer downloadLoggersMu.Unlock()
	if logger, ok := downloadLoggers[key]; ok {
		return logger
	}

	f, err := os.OpenFile(getDownloadLogPath(key), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		global.LOG.Errorf("failed to open download log file: %v", err)
	}

	logger := &DownloadFileLogger{
		Key:       key,
		logs:      make([]string, 0),
		listeners: make([]chan DownloadFileEvent, 0),
		status:    "running",
		file:      f,
	}
	downloadLoggers[key] = logger
	return logger
}

func IsDownloadLoggerActive(key string) bool {
	downloadLoggersMu.RLock()
	defer downloadLoggersMu.RUnlock()
	_, ok := downloadLoggers[key]
	return ok
}

func RemoveDownloadLogger(key string) {
	downloadLoggersMu.Lock()
	logger, ok := downloadLoggers[key]
	if ok {
		delete(downloadLoggers, key)
	}
	downloadLoggersMu.Unlock()
	if !ok {
		return
	}

	logger.mu.Lock()
	listeners := append([]chan DownloadFileEvent(nil), logger.listeners...)
	logger.listeners = nil
	if logger.file != nil {
		_ = logger.file.Close()
		logger.file = nil
	}
	status := logger.status
	logger.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- DownloadFileEvent{Type: "status", Status: status}:
		default:
		}
		select {
		case listener <- DownloadFileEvent{Type: "eof", Message: "EOF"}:
		default:
		}
		close(listener)
	}
}

func (l *DownloadFileLogger) AppendLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	text := fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), line)
	l.mu.Lock()
	l.logs = append(l.logs, text)
	if l.file != nil {
		_, _ = l.file.WriteString(text + "\n")
	}
	listeners := append([]chan DownloadFileEvent(nil), l.listeners...)
	l.mu.Unlock()
	for _, listener := range listeners {
		select {
		case listener <- DownloadFileEvent{Type: "log", Message: text}:
		default:
		}
	}
}

func (l *DownloadFileLogger) Appendf(format string, args ...interface{}) {
	l.AppendLine(fmt.Sprintf(format, args...))
}

// SetProgress 推送下载进度
func (l *DownloadFileLogger) SetProgress(written, total uint64) {
	percentVal := 0.0
	if total > 0 {
		percentVal = float64(written) / float64(total) * 100
	}
	l.mu.Lock()
	listeners := append([]chan DownloadFileEvent(nil), l.listeners...)
	l.mu.Unlock()
	for _, listener := range listeners {
		select {
		case listener <- DownloadFileEvent{Type: "progress", Percent: percentVal, Message: fmt.Sprintf("%.2f%% (%d/%d bytes)", percentVal, written, total)}:
		default:
		}
	}
}

func (l *DownloadFileLogger) SetStatus(status string) {
	l.mu.Lock()
	l.status = status
	listeners := append([]chan DownloadFileEvent(nil), l.listeners...)
	l.mu.Unlock()
	for _, listener := range listeners {
		select {
		case listener <- DownloadFileEvent{Type: "status", Status: status}:
		default:
		}
	}
}

func (l *DownloadFileLogger) GetLogs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, len(l.logs))
	copy(out, l.logs)
	return out
}

func (l *DownloadFileLogger) GetStatus() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.status
}

func (l *DownloadFileLogger) Subscribe() chan DownloadFileEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan DownloadFileEvent, 200)
	l.listeners = append(l.listeners, ch)
	return ch
}

func (l *DownloadFileLogger) Unsubscribe(ch chan DownloadFileEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, listener := range l.listeners {
		if listener == ch {
			l.listeners = append(l.listeners[:i], l.listeners[i+1:]...)
			break
		}
	}
}

func ReadDownloadLogFromFile(key string) ([]string, error) {
	f, err := os.Open(getDownloadLogPath(key))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lines := make([]string, 0, 256)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return lines, err
	}
	return lines, nil
}
