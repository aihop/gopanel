package service

import (
	"fmt"
	"sync"
	"time"
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
}

func GetAppInstallLogger(name string) *AppInstallLogger {
	appInstallLoggersMu.Lock()
	defer appInstallLoggersMu.Unlock()
	if logger, exists := appInstallLoggers[name]; exists {
		return logger
	}
	logger := &AppInstallLogger{
		Name:      name,
		logs:      []string{},
		Listeners: make([]chan string, 0),
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