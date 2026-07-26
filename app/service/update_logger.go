package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
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
	closed    bool // 已 Remove：listeners 里的 channel 都已 close，不能再往里发
}

// NewUpdateLogName 生成唯一的日志名。
//
// 以前用 "<前缀>_20060102150405.log"，只到秒 —— 同一秒内点两次按钮会拿到同一个
// UpdateLogger 实例，先跑完的那个 RemoveUpdateLogger 会把 channel 关掉，
// 另一个还在写日志的任务就会 "send on closed channel" panic，
// 而它跑在后台 goroutine 里，panic 会直接带走整个面板进程。
func NewUpdateLogName(prefix string) string {
	// 进程内用原子自增保证唯一；再拼一段随机串，避免面板重启后计数器归零
	// 撞上同一秒里已经存在的日志文件
	seq := updateLogSeq.Add(1)
	return fmt.Sprintf("%s_%s_%d%s.log", prefix, time.Now().Format("20060102150405"), seq, common.RandStr(4))
}

var updateLogSeq atomic.Uint64

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

	// 关 channel 和往 channel 里发消息必须在同一把锁内完成：
	// 否则 Append/SetStatus 拿到 listeners 快照后、发送前被这里 close，
	// 就是 "send on closed channel" panic（后台 goroutine 里 = 面板进程退出）。
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if logger.closed {
		return
	}
	logger.closed = true
	if logger.file != nil {
		_ = logger.file.Close()
		logger.file = nil
	}
	for _, listener := range logger.listeners {
		select {
		case listener <- UpdateLogEvent{Type: "status", Status: logger.status}:
		default:
		}
		select {
		case listener <- UpdateLogEvent{Type: "eof", Message: "EOF"}:
		default:
		}
		close(listener)
	}
	logger.listeners = nil
}

func (l *UpdateLogger) Append(text string, param interface{}) {
	line := fmt.Sprintf("[%s] %s: %v", nowRFC3339(), text, param)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, line)
	if l.file != nil {
		_, _ = l.file.WriteString(line + "\n")
	}
	l.broadcastLocked(UpdateLogEvent{Type: "log", Message: line})
}

func (l *UpdateLogger) SetStatus(status string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.status = status
	l.broadcastLocked(UpdateLogEvent{Type: "status", Status: status})
}

// broadcastLocked 必须在持有 l.mu 时调用。
// channel 都是带缓冲的（cap 100）且这里用非阻塞发送，所以持锁广播不会卡住写日志的一方。
func (l *UpdateLogger) broadcastLocked(event UpdateLogEvent) {
	if l.closed {
		return
	}
	for _, listener := range l.listeners {
		select {
		case listener <- event:
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
	if l.closed {
		// 任务已结束：直接给一个已关闭的 channel，订阅方读到 EOF 就退出，
		// 不能挂进 listeners（那个 channel 永远不会再被 close）
		close(ch)
		return ch
	}
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
