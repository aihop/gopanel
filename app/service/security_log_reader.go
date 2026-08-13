package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

func readSecurityLogBatch(path string, cursor *model.SecurityLogCursor, maxBytes, maxLines int) ([]string, error) {
	if cursor == nil {
		return nil, errors.New("security log cursor is nil")
	}
	if maxBytes < 4096 {
		maxBytes = 4096
	}
	if maxLines < 1 {
		maxLines = 1000
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	identity := securityFileIdentity(info, path)
	if cursor.FileIdentity != "" && cursor.FileIdentity != identity {
		cursor.Offset = 0
	}
	if info.Size() < cursor.Offset {
		cursor.Offset = 0
	}
	cursor.Path, cursor.FileIdentity = path, identity
	if cursor.Offset == 0 && cursor.Processed == 0 && cursor.Malformed == 0 && info.Size() > int64(maxBytes) {
		cursor.Offset = info.Size() - int64(maxBytes)
		cursor.Dropped++
	}
	if info.Size() == cursor.Offset {
		return []string{}, nil
	}
	readSize := int64(maxBytes + 1)
	if remaining := info.Size() - cursor.Offset; remaining < readSize {
		readSize = remaining
	}
	buffer := make([]byte, readSize)
	count, readErr := file.ReadAt(buffer, cursor.Offset)
	if readErr != nil && readErr != io.EOF {
		return nil, readErr
	}
	buffer = buffer[:count]
	if cursor.Dropped > 0 && cursor.Processed == 0 && cursor.Malformed == 0 && cursor.Offset > 0 {
		if firstNewline := strings.IndexByte(string(buffer), '\n'); firstNewline >= 0 {
			cursor.Offset += int64(firstNewline + 1)
			buffer = buffer[firstNewline+1:]
			count = len(buffer)
		}
	}
	consume := len(buffer)
	if int64(count)+cursor.Offset < info.Size() {
		lastNewline := strings.LastIndexByte(string(buffer[:min(len(buffer), maxBytes)]), '\n')
		if lastNewline < 0 {
			cursor.Offset += int64(min(len(buffer), maxBytes))
			cursor.Dropped++
			return []string{}, nil
		}
		consume = lastNewline + 1
	}
	content := strings.TrimSuffix(string(buffer[:consume]), "\n")
	cursor.Offset += int64(consume)
	if content == "" {
		return []string{}, nil
	}
	lines := strings.Split(content, "\n")
	if len(lines) > maxLines {
		cursor.Dropped += int64(len(lines) - maxLines)
		lines = lines[:maxLines]
	}
	return lines, nil
}

func securityFileIdentity(info os.FileInfo, path string) string {
	value := reflect.ValueOf(info.Sys())
	if value.IsValid() && value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value.IsValid() && value.Kind() == reflect.Struct {
		inode, inodeOK := securityStatUint(value, "Ino")
		device, _ := securityStatUint(value, "Dev")
		if inodeOK {
			return fmt.Sprintf("%d:%d", device, inode)
		}
	}
	return path
}

func securityStatUint(value reflect.Value, name string) (uint64, bool) {
	field := value.FieldByName(name)
	if !field.IsValid() {
		return 0, false
	}
	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(field.Int()), true
	case reflect.String:
		parsed, err := strconv.ParseUint(field.String(), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
