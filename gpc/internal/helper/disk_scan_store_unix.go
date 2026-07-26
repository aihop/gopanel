//go:build !windows

package helper

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	errPathNotAbsolute = errors.New("path must be absolute")
	errPathProtected   = errors.New("path is protected and can never be removed")
	errPathNotInScan   = errors.New("path is not in the referenced scan result")
	errNotRegularFile  = errors.New("only regular files can be operated")
	errFileTooSmall    = errors.New("file is smaller than the minimum size allowed for scan-granted operations")
	errScanNotFound    = errors.New("scan result not found or expired")
)

const (
	// diskGrantMinSize 授权删除的硬门槛：无论请求里 minSize 传什么，
	// 小于这个体积的文件永远不能通过 scanId 授权删除。
	// 没有这条，攻击者只要用 minSize=1 扫一遍就能把任意文件纳入“可删”集合。
	diskGrantMinSize int64 = 10 << 20 // 10MB
	// diskScanTTL 扫描结果保留时长，过期即失效，逼迫重新扫描
	diskScanTTL = 30 * time.Minute
	// diskScanMaxEntries 单次扫描最多记住多少条路径，防止内存被撑爆
	diskScanMaxEntries = 5000
)

type diskScanEntry struct {
	paths     map[string]struct{}
	expiresAt time.Time
}

type diskScanStoreT struct {
	mu   sync.Mutex
	data map[string]*diskScanEntry
}

var diskScanStore = &diskScanStoreT{data: make(map[string]*diskScanEntry)}

func (s *diskScanStoreT) save(paths []string) (string, error) {
	id, err := randomScanID()
	if err != nil {
		return "", err
	}
	set := make(map[string]struct{}, len(paths))
	for i, p := range paths {
		if i >= diskScanMaxEntries {
			break
		}
		set[p] = struct{}{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	s.data[id] = &diskScanEntry{paths: set, expiresAt: time.Now().Add(diskScanTTL)}
	return id, nil
}

func (s *diskScanStoreT) contains(scanID string, path string) bool {
	if scanID == "" || path == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[scanID]
	if !ok {
		return false
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.data, scanID)
		return false
	}
	_, ok = entry.paths[path]
	return ok
}

func (s *diskScanStoreT) exists(scanID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[scanID]
	if !ok {
		return false
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.data, scanID)
		return false
	}
	return true
}

// forget 删除成功后把路径移出集合，避免同一个 scanId 被反复使用
func (s *diskScanStoreT) forget(scanID string, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, ok := s.data[scanID]; ok {
		delete(entry.paths, path)
	}
}

func (s *diskScanStoreT) gcLocked() {
	now := time.Now()
	for id, entry := range s.data {
		if now.After(entry.expiresAt) {
			delete(s.data, id)
		}
	}
}

func randomScanID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
