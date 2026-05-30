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

// FileCompressEvent 压缩任务日志事件
type FileCompressEvent struct {
	Type    string // log | status | eof
	Message string
	Status  string // running | success | failed
}

// FileCompressLogger 文件压缩任务的 Logger，参考 BackupLogger 模式
type FileCompressLogger struct {
	Key       string
	logs      []string
	listeners []chan FileCompressEvent
	status    string
	mu        sync.RWMutex
	file      *os.File
}

var (
	fileCompressLoggers   = make(map[string]*FileCompressLogger)
	fileCompressLoggersMu sync.RWMutex
)

func getFileCompressLogPath(key string) string {
	logDir := filepath.Join(global.CONF.System.TmpDir, "file_compress_logs")
	_ = os.MkdirAll(logDir, 0o755)
	return filepath.Join(logDir, key)
}

// GetFileCompressLogger 获取或创建指定 key 的压缩任务 Logger
func GetFileCompressLogger(key string) *FileCompressLogger {
	fileCompressLoggersMu.Lock()
	defer fileCompressLoggersMu.Unlock()
	if logger, ok := fileCompressLoggers[key]; ok {
		return logger
	}

	f, err := os.OpenFile(getFileCompressLogPath(key), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		global.LOG.Errorf("failed to open file compress log file: %v", err)
	}

	logger := &FileCompressLogger{
		Key:       key,
		logs:      make([]string, 0),
		listeners: make([]chan FileCompressEvent, 0),
		status:    "running",
		file:      f,
	}
	fileCompressLoggers[key] = logger
	return logger
}

// IsFileCompressLoggerActive 检查指定 key 的压缩任务是否仍活跃
func IsFileCompressLoggerActive(key string) bool {
	fileCompressLoggersMu.RLock()
	defer fileCompressLoggersMu.RUnlock()
	_, ok := fileCompressLoggers[key]
	return ok
}

// RemoveFileCompressLogger 移除并清理指定 key 的压缩任务 Logger
func RemoveFileCompressLogger(key string) {
	fileCompressLoggersMu.Lock()
	logger, ok := fileCompressLoggers[key]
	if ok {
		delete(fileCompressLoggers, key)
	}
	fileCompressLoggersMu.Unlock()
	if !ok {
		return
	}

	logger.mu.Lock()
	listeners := append([]chan FileCompressEvent(nil), logger.listeners...)
	logger.listeners = nil
	if logger.file != nil {
		_ = logger.file.Close()
		logger.file = nil
	}
	status := logger.status
	logger.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- FileCompressEvent{Type: "status", Status: status}:
		default:
		}
		select {
		case listener <- FileCompressEvent{Type: "eof", Message: "EOF"}:
		default:
		}
		close(listener)
	}
}

// AppendLine 追加一行日志
func (l *FileCompressLogger) AppendLine(line string) {
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
	listeners := append([]chan FileCompressEvent(nil), l.listeners...)
	l.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- FileCompressEvent{Type: "log", Message: text}:
		default:
		}
	}
}

// Appendf 格式化追加日志
func (l *FileCompressLogger) Appendf(format string, args ...interface{}) {
	l.AppendLine(fmt.Sprintf(format, args...))
}

// SetStatus 设置状态（running / success / failed）
func (l *FileCompressLogger) SetStatus(status string) {
	l.mu.Lock()
	l.status = status
	listeners := append([]chan FileCompressEvent(nil), l.listeners...)
	l.mu.Unlock()

	for _, listener := range listeners {
		select {
		case listener <- FileCompressEvent{Type: "status", Status: status}:
		default:
		}
	}
}

// GetLogs 返回当前所有日志
func (l *FileCompressLogger) GetLogs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, len(l.logs))
	copy(out, l.logs)
	return out
}

// GetStatus 返回当前状态
func (l *FileCompressLogger) GetStatus() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.status
}

// Subscribe 注册一个事件接收 channel
func (l *FileCompressLogger) Subscribe() chan FileCompressEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan FileCompressEvent, 200)
	l.listeners = append(l.listeners, ch)
	return ch
}

// Unsubscribe 取消注册事件接收 channel
func (l *FileCompressLogger) Unsubscribe(ch chan FileCompressEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, listener := range l.listeners {
		if listener == ch {
			l.listeners = append(l.listeners[:i], l.listeners[i+1:]...)
			break
		}
	}
}

// ReadFileCompressLogFromFile 从日志文件读取历史日志
func ReadFileCompressLogFromFile(key string) ([]string, error) {
	f, err := os.Open(getFileCompressLogPath(key))
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
