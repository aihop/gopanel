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

type BackupLogEvent struct {
	Type    string
	Message string
	Status  string
}

type BackupLogger struct {
	Key       string
	logs      []string
	listeners []chan BackupLogEvent
	status    string
	mu        sync.RWMutex
	file      *os.File
}

var (
	backupLoggers   = make(map[string]*BackupLogger)
	backupLoggersMu sync.RWMutex
)

func getBackupLogFilePath(key string) string {
	logDir := filepath.Join(global.CONF.System.TmpDir, "backup_logs")
	_ = os.MkdirAll(logDir, 0o755)
	return filepath.Join(logDir, key)
}

func GetBackupLogger(key string) *BackupLogger {
	backupLoggersMu.Lock()
	defer backupLoggersMu.Unlock()
	if logger, ok := backupLoggers[key]; ok {
		return logger
	}

	f, err := os.OpenFile(getBackupLogFilePath(key), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		global.LOG.Errorf("failed to open backup log file: %v", err)
	}

	logger := &BackupLogger{
		Key:       key,
		logs:      make([]string, 0),
		listeners: make([]chan BackupLogEvent, 0),
		status:    "running",
		file:      f,
	}
	backupLoggers[key] = logger
	return logger
}

func IsBackupLoggerActive(key string) bool {
	backupLoggersMu.RLock()
	defer backupLoggersMu.RUnlock()
	_, ok := backupLoggers[key]
	return ok
}

func RemoveBackupLogger(key string) {
	backupLoggersMu.Lock()
	logger, ok := backupLoggers[key]
	if ok {
		delete(backupLoggers, key)
	}
	backupLoggersMu.Unlock()
	if !ok {
		return
	}

	logger.mu.Lock()
	listeners := append([]chan BackupLogEvent(nil), logger.listeners...)
	logger.listeners = nil
	if logger.file != nil {
		_ = logger.file.Close()
		logger.file = nil
	}
	status := logger.status
	logger.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- BackupLogEvent{Type: "status", Status: status}:
		default:
		}
		select {
		case listener <- BackupLogEvent{Type: "eof", Message: "EOF"}:
		default:
		}
		close(listener)
	}
}

func (l *BackupLogger) AppendLine(line string) {
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
	listeners := append([]chan BackupLogEvent(nil), l.listeners...)
	l.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- BackupLogEvent{Type: "log", Message: text}:
		default:
		}
	}
}

func (l *BackupLogger) Appendf(format string, args ...interface{}) {
	l.AppendLine(fmt.Sprintf(format, args...))
}

func (l *BackupLogger) SetStatus(status string) {
	l.mu.Lock()
	l.status = status
	listeners := append([]chan BackupLogEvent(nil), l.listeners...)
	l.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- BackupLogEvent{Type: "status", Status: status}:
		default:
		}
	}
}

func (l *BackupLogger) GetLogs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, len(l.logs))
	copy(out, l.logs)
	return out
}

func (l *BackupLogger) GetStatus() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.status
}

func (l *BackupLogger) Subscribe() chan BackupLogEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan BackupLogEvent, 200)
	l.listeners = append(l.listeners, ch)
	return ch
}

func (l *BackupLogger) Unsubscribe(ch chan BackupLogEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, listener := range l.listeners {
		if listener == ch {
			l.listeners = append(l.listeners[:i], l.listeners[i+1:]...)
			break
		}
	}
}

func ReadBackupLogFromFile(key string) ([]string, error) {
	f, err := os.Open(getBackupLogFilePath(key))
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

