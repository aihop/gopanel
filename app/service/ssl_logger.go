package service

import (
	"fmt"
	"sync"
	"time"
)

var (
	sslLoggers   = make(map[uint]*SSLLogger)
	sslLoggersMu sync.RWMutex
)

type SSLLogger struct {
	ID        uint
	logs      []string
	Listeners []chan string
	mu        sync.RWMutex
}

func GetSSLLogger(id uint) *SSLLogger {
	sslLoggersMu.Lock()
	defer sslLoggersMu.Unlock()
	if logger, exists := sslLoggers[id]; exists {
		return logger
	}
	logger := &SSLLogger{
		ID:        id,
		logs:      []string{},
		Listeners: make([]chan string, 0),
	}
	sslLoggers[id] = logger
	return logger
}

func RemoveSSLLogger(id uint) {
	sslLoggersMu.Lock()
	defer sslLoggersMu.Unlock()
	if logger, exists := sslLoggers[id]; exists {
		logger.mu.Lock()
		for _, listener := range logger.Listeners {
			close(listener)
		}
		logger.mu.Unlock()
		delete(sslLoggers, id)
	}
}

func (l *SSLLogger) Info(format string, a ...interface{}) {
	msg := fmt.Sprintf("[%s] INFO: %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
	l.appendLog(msg)
}

func (l *SSLLogger) Error(format string, a ...interface{}) {
	msg := fmt.Sprintf("[%s] ERROR: %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
	l.appendLog(msg)
}

func (l *SSLLogger) appendLog(msg string) {
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

func (l *SSLLogger) GetLogs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	logsCopy := make([]string, len(l.logs))
	copy(logsCopy, l.logs)
	return logsCopy
}

func (l *SSLLogger) Subscribe() chan string {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan string, 100)
	l.Listeners = append(l.Listeners, ch)
	return ch
}

func (l *SSLLogger) Unsubscribe(ch chan string) {
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
