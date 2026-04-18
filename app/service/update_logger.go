package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/global"
)

type UpdateLogEvent struct {
	Type    string
	Message string
	Status  string
}

type UpdateLogger struct {
	Name      string
	logs      []string
	listeners []chan UpdateLogEvent
	status    string
	mu        sync.RWMutex
	file      *os.File
}

var (
	updateLoggers   = make(map[string]*UpdateLogger)
	updateLoggersMu sync.RWMutex
)

func getUpdateLogFilePath(name string) string {
	logDir := filepath.Join(global.CONF.System.TmpDir, "install_logs")
	_ = os.MkdirAll(logDir, 0o755)
	return filepath.Join(logDir, name)
}

func GetUpdateLogger(name string) *UpdateLogger {
	updateLoggersMu.Lock()
	defer updateLoggersMu.Unlock()
	if logger, ok := updateLoggers[name]; ok {
		return logger
	}

	f, err := os.OpenFile(getUpdateLogFilePath(name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		global.LOG.Errorf("failed to open update log file: %v", err)
	}

	logger := &UpdateLogger{
		Name:      name,
		logs:      make([]string, 0),
		listeners: make([]chan UpdateLogEvent, 0),
		status:    "running",
		file:      f,
	}
	updateLoggers[name] = logger
	return logger
}

func IsUpdateLoggerActive(name string) bool {
	updateLoggersMu.RLock()
	defer updateLoggersMu.RUnlock()
	_, ok := updateLoggers[name]
	return ok
}

func RemoveUpdateLogger(name string) {
	updateLoggersMu.Lock()
	logger, ok := updateLoggers[name]
	if ok {
		delete(updateLoggers, name)
	}
	updateLoggersMu.Unlock()
	if !ok {
		return
	}

	logger.mu.Lock()
	listeners := append([]chan UpdateLogEvent(nil), logger.listeners...)
	logger.listeners = nil
	if logger.file != nil {
		_ = logger.file.Close()
		logger.file = nil
	}
	status := logger.status
	logger.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- UpdateLogEvent{Type: "status", Status: status}:
		default:
		}
		select {
		case listener <- UpdateLogEvent{Type: "eof", Message: "EOF"}:
		default:
		}
		close(listener)
	}
}

func (l *UpdateLogger) Append(text string, param interface{}) {
	line := fmt.Sprintf("[%s] %s: %v", nowRFC3339(), text, param)
	l.mu.Lock()
	l.logs = append(l.logs, line)
	if l.file != nil {
		_, _ = l.file.WriteString(line + "\n")
	}
	listeners := append([]chan UpdateLogEvent(nil), l.listeners...)
	l.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- UpdateLogEvent{Type: "log", Message: line}:
		default:
		}
	}
}

func (l *UpdateLogger) SetStatus(status string) {
	l.mu.Lock()
	l.status = status
	listeners := append([]chan UpdateLogEvent(nil), l.listeners...)
	l.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- UpdateLogEvent{Type: "status", Status: status}:
		default:
		}
	}
}

func (l *UpdateLogger) GetLogs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, len(l.logs))
	copy(out, l.logs)
	return out
}

func (l *UpdateLogger) GetStatus() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.status
}

func (l *UpdateLogger) Subscribe() chan UpdateLogEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan UpdateLogEvent, 100)
	l.listeners = append(l.listeners, ch)
	return ch
}

func (l *UpdateLogger) Unsubscribe(ch chan UpdateLogEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, listener := range l.listeners {
		if listener == ch {
			l.listeners = append(l.listeners[:i], l.listeners[i+1:]...)
			break
		}
	}
}

func ReadUpdateLogFromFile(name string) ([]string, error) {
	content, err := os.ReadFile(getUpdateLogFilePath(name))
	if err != nil {
		return nil, err
	}
	raw := strings.Split(string(content), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func InferUpdateLogStatus(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if strings.Contains(line, "successful update to version_code") {
			return "success"
		}
		if strings.Contains(strings.ToLower(line), "upload error") ||
			strings.Contains(strings.ToLower(line), "restart error") ||
			strings.Contains(strings.ToLower(line), "failed") {
			return "failed"
		}
	}
	return "running"
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}
