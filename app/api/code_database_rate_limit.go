package api

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	codeDatabaseRateWindow   = time.Minute
	codeDatabaseUserLimit    = 30
	codeDatabaseProjectLimit = 60
)

type codeRateWindow struct {
	mu     sync.Mutex
	events map[string][]time.Time
	now    func() time.Time
}

var codeDatabaseQueries = &codeRateWindow{events: make(map[string][]time.Time), now: time.Now}

func (window *codeRateWindow) allow(key string, limit int) bool {
	window.mu.Lock()
	defer window.mu.Unlock()
	now := window.now()
	cutoff := now.Add(-codeDatabaseRateWindow)
	events := window.events[key]
	first := 0
	for first < len(events) && events[first].Before(cutoff) {
		first++
	}
	events = events[first:]
	if len(events) >= limit {
		window.events[key] = events
		return false
	}
	window.events[key] = append(events, now)
	return true
}

func allowCodeDatabaseQuery(userID, projectID uint) error {
	if !codeDatabaseQueries.allow(fmt.Sprintf("user:%d", userID), codeDatabaseUserLimit) {
		return errors.New("数据库查询过于频繁，请稍后重试")
	}
	if !codeDatabaseQueries.allow(fmt.Sprintf("project:%d", projectID), codeDatabaseProjectLimit) {
		return errors.New("当前项目数据库查询过于频繁，请稍后重试")
	}
	return nil
}
