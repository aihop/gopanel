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
	pipelineLoggers   = make(map[uint]*PipelineLogger)
	pipelineLoggersMu sync.RWMutex
)

func getLogFilePath(id uint) string {
	baseDir := strings.TrimSpace(global.CONF.System.BaseDir)
	if baseDir == "" || !filepath.IsAbs(baseDir) {
		baseDir = filepath.Join(os.TempDir(), "gopanel")
	}
	logDir := filepath.Join(baseDir, "pipelines", "logs")
	os.MkdirAll(logDir, 0755)
	return filepath.Join(logDir, fmt.Sprintf("task_%d.log", id))
}

type PipelineLogger struct {
	ID        uint
	logs      []string
	Listeners []chan string
	mu        sync.RWMutex
	file      *os.File
}

func GetPipelineLogger(id uint) *PipelineLogger {
	pipelineLoggersMu.Lock()
	defer pipelineLoggersMu.Unlock()
	if logger, exists := pipelineLoggers[id]; exists {
		return logger
	}

	logPath := getLogFilePath(id)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		global.LOG.Errorf("failed to open pipeline log file: %v", err)
	}

	logs, _ := ReadPipelineLogFromFile(id)
	if len(logs) > 3000 {
		logs = logs[len(logs)-3000:]
	}
	logger := &PipelineLogger{
		ID:        id,
		logs:      logs,
		Listeners: make([]chan string, 0),
		file:      f,
	}
	pipelineLoggers[id] = logger
	return logger
}

func SubscribePipelineLogger(id uint) (*PipelineLogger, []string, chan string, bool) {
	pipelineLoggersMu.RLock()
	defer pipelineLoggersMu.RUnlock()
	logger, exists := pipelineLoggers[id]
	if !exists {
		return nil, nil, nil, false
	}
	logger.mu.Lock()
	defer logger.mu.Unlock()
	logs := make([]string, len(logger.logs))
	copy(logs, logger.logs)
	ch := make(chan string, 100)
	logger.Listeners = append(logger.Listeners, ch)
	return logger, logs, ch, true
}

func IsPipelineLoggerActive(id uint) bool {
	pipelineLoggersMu.RLock()
	defer pipelineLoggersMu.RUnlock()
	_, exists := pipelineLoggers[id]
	return exists
}

func RemovePipelineLogger(id uint) {
	pipelineLoggersMu.Lock()
	defer pipelineLoggersMu.Unlock()
	if logger, exists := pipelineLoggers[id]; exists {
		logger.mu.Lock()
		listeners := append([]chan string(nil), logger.Listeners...)
		logger.Listeners = nil
		logger.mu.Unlock()

		for _, listener := range listeners {
			// 发送结束信号，让前端断开
			select {
			case listener <- "EOF":
			default:
			}
			close(listener)
		}

		logger.mu.Lock()
		if logger.file != nil {
			logger.file.Close()
		}
		logger.mu.Unlock()
		delete(pipelineLoggers, id)
	}
}

func (l *PipelineLogger) Info(format string, a ...interface{}) {
	msg := fmt.Sprintf("[%s] INFO: %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
	l.appendLog(msg)
}

func (l *PipelineLogger) Error(format string, a ...interface{}) {
	msg := fmt.Sprintf("[%s] ERROR: %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
	l.appendLog(msg)
}

func (l *PipelineLogger) appendLog(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, msg)
	if l.file != nil {
		l.file.WriteString(msg + "\n")
	}
	for _, listener := range l.Listeners {
		select {
		case listener <- msg:
		default:
		}
	}
}

func ReadPipelineLogFromFile(id uint) ([]string, error) {
	logPath := getLogFilePath(id)
	content, err := os.ReadFile(logPath)
	if err != nil {
		return nil, err
	}
	var lines []string
	// simple split
	raw := string(content)
	for _, line := range strings.Split(raw, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func (l *PipelineLogger) GetLogs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	logsCopy := make([]string, len(l.logs))
	copy(logsCopy, l.logs)
	return logsCopy
}

func (l *PipelineLogger) Subscribe() chan string {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan string, 100)
	l.Listeners = append(l.Listeners, ch)
	return ch
}

func (l *PipelineLogger) Unsubscribe(ch chan string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, listener := range l.Listeners {
		if listener == ch {
			l.Listeners = append(l.Listeners[:i], l.Listeners[i+1:]...)
			break
		}
	}
}
