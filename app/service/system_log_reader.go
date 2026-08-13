package service

import (
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
)

const (
	defaultSystemLogBytes int64 = 512 << 10
	maxSystemLogBytes     int64 = 2 << 20
)

var systemLogNamePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(?:[-._]\d+)?$`)

func readSystemLogTail(logDir, name string, maxBytes int64) (*dto.SystemLogContent, error) {
	path, err := resolveSystemLogPath(logDir, name)
	if err != nil {
		return nil, err
	}
	limit := normalizeSystemLogLimit(maxBytes)
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var content []byte
	var truncated bool
	if strings.HasSuffix(path, ".gz") {
		content, truncated, err = readGzipTail(path, limit)
	} else {
		content, truncated, err = readFileTail(path, limit)
	}
	if err != nil {
		return nil, err
	}
	return &dto.SystemLogContent{
		Content: string(content), FileName: filepath.Base(path), Size: info.Size(),
		ReturnedBytes: len(content), Truncated: truncated,
	}, nil
}

func resolveSystemLogPath(logDir, name string) (string, error) {
	name = strings.TrimSpace(name)
	if !systemLogNamePattern.MatchString(name) {
		return "", errors.New("系统日志文件名无效")
	}
	candidates := []string{"gopanel-" + name + ".log", "gopanel-" + name + ".log.gz", "gopanel-" + name + ".gz"}
	if name == time.Now().Format("2006-01-02") {
		candidates = append([]string{"gopanel.log"}, candidates...)
	}
	for _, candidate := range candidates {
		path := filepath.Join(logDir, candidate)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}

func normalizeSystemLogLimit(value int64) int64 {
	if value <= 0 {
		return defaultSystemLogBytes
	}
	if value > maxSystemLogBytes {
		return maxSystemLogBytes
	}
	return value
}

func readFileTail(path string, limit int64) ([]byte, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, false, err
	}
	start := info.Size() - limit
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, false, err
	}
	content, err := io.ReadAll(io.LimitReader(file, limit))
	return content, start > 0, err
}

func readGzipTail(path string, limit int64) ([]byte, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, false, err
	}
	defer reader.Close()
	buffer := make([]byte, int(limit))
	total := int64(0)
	for {
		chunk := make([]byte, 32<<10)
		read, readErr := reader.Read(chunk)
		if read > 0 {
			for _, value := range chunk[:read] {
				buffer[total%limit] = value
				total++
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, false, readErr
		}
	}
	if total <= limit {
		return buffer[:total], false, nil
	}
	start := total % limit
	result := append([]byte{}, buffer[start:]...)
	result = append(result, buffer[:start]...)
	return result, true, nil
}
