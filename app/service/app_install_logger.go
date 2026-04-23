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

var (
	appInstallLoggers   = make(map[string]*AppInstallLogger)
	appInstallLoggersMu sync.RWMutex
)

type AppInstallLogger struct {
	Name      string
	logs      []string
	Listeners []chan string
	mu        sync.RWMutex
	file      *os.File
}

func getAppInstallLogFilePath(name string) string {
	logDir := filepath.Join(global.CONF.System.LogPath, "app_install")
	_ = os.MkdirAll(logDir, 0o755)
	safeName := strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(strings.TrimSpace(name))
	if safeName == "" {
		safeName = "unknown"
	}
	return filepath.Join(logDir, safeName+".log")
}

func loadAppInstallLogsFromFile(name string) []string {
	content, err := os.ReadFile(getAppInstallLogFilePath(name))
	if err != nil {
		return []string{}
	}
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return []string{}
	}
	return strings.Split(text, "\n")
}

func IsAppInstallLoggerActive(name string) bool {
	appInstallLoggersMu.RLock()
	defer appInstallLoggersMu.RUnlock()
	_, exists := appInstallLoggers[name]
	return exists
}

func GetAppInstallLogger(name string) *AppInstallLogger {
	appInstallLoggersMu.Lock()
	defer appInstallLoggersMu.Unlock()
	if logger, exists := appInstallLoggers[name]; exists {
		return logger
	}

	f, err := os.OpenFile(getAppInstallLogFilePath(name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil && global.LOG != nil {
		global.LOG.Errorf("failed to open app install log file: %v", err)
	}
	logger := &AppInstallLogger{
		Name:      name,
		logs:      loadAppInstallLogsFromFile(name),
		Listeners: make([]chan string, 0),
		file:      f,
	}
	appInstallLoggers[name] = logger
	return logger
}

func RemoveAppInstallLogger(name string) {
	appInstallLoggersMu.Lock()
	defer appInstallLoggersMu.Unlock()
	if logger, exists := appInstallLoggers[name]; exists {
		logger.mu.Lock()
		for _, listener := range logger.Listeners {
			close(listener)
		}
		logger.Listeners = nil
		if logger.file != nil {
			_ = logger.file.Close()
			logger.file = nil
		}
		logger.mu.Unlock()
		delete(appInstallLoggers, name)
	}
}

func (l *AppInstallLogger) Info(format string, a ...interface{}) {
	msg := fmt.Sprintf("[%s] INFO: %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
	l.appendLog(msg)
}

func (l *AppInstallLogger) Error(format string, a ...interface{}) {
	msg := fmt.Sprintf("[%s] ERROR: %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
	l.appendLog(msg)
}

func (l *AppInstallLogger) appendLog(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, msg)
	if l.file != nil {
		_, _ = l.file.WriteString(msg + "\n")
	}
	for _, listener := range l.Listeners {
		// Non-blocking send
		select {
		case listener <- msg:
		default:
		}
	}
}

func (l *AppInstallLogger) GetLogs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	logsCopy := make([]string, len(l.logs))
	copy(logsCopy, l.logs)
	return logsCopy
}

func (l *AppInstallLogger) Subscribe() chan string {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan string, 100)
	l.Listeners = append(l.Listeners, ch)
	return ch
}

func (l *AppInstallLogger) Unsubscribe(ch chan string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, listener := range l.Listeners {
		if listener == ch {
			l.Listeners = append(l.Listeners[:i], l.Listeners[i+1:]...)
			close(ch)
			break
		}
	}
}
