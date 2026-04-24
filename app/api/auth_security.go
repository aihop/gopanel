package api

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/dto"
)

const (
	loginAttemptWindow        = 10 * time.Minute
	loginAttemptLockDuration  = 15 * time.Minute
	loginAttemptDelay         = 800 * time.Millisecond
	loginAttemptMaxPerAccount = 8
	loginAttemptMaxPerIP      = 20
)

type loginAttemptEntry struct {
	Failed      int
	LastFailed  time.Time
	LockedUntil time.Time
}

type loginAttemptGuard struct {
	mu      sync.Mutex
	entries map[string]*loginAttemptEntry
}

var defaultLoginAttemptGuard = &loginAttemptGuard{
	entries: make(map[string]*loginAttemptEntry),
}

func (g *loginAttemptGuard) Check(ip string, req *dto.AuthSignin) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	g.cleanupExpired(now)

	if msg, ok := g.checkKeyLocked(now, g.ipKey(ip), "当前 IP 登录尝试过于频繁，请 15 分钟后再试"); ok {
		return msg, true
	}

	account := normalizeLoginAccount(req)
	if account == "" {
		return "", false
	}
	return g.checkKeyLocked(now, g.accountKey(ip, account), "该账号登录尝试过于频繁，请 15 分钟后再试")
}

func (g *loginAttemptGuard) RequiresCaptcha(ip string, req *dto.AuthSignin) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	g.cleanupExpired(now)

	if g.failedCount(g.ipKey(ip), now) > 0 {
		return true
	}

	account := normalizeLoginAccount(req)
	if account == "" {
		return false
	}
	return g.failedCount(g.accountKey(ip, account), now) > 0
}

func (g *loginAttemptGuard) RegisterFailure(ip string, req *dto.AuthSignin) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	g.cleanupExpired(now)
	g.bumpKey(now, g.ipKey(ip), loginAttemptMaxPerIP)

	account := normalizeLoginAccount(req)
	if account != "" {
		g.bumpKey(now, g.accountKey(ip, account), loginAttemptMaxPerAccount)
	}
}

func (g *loginAttemptGuard) RegisterSuccess(ip string, req *dto.AuthSignin) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.entries, g.ipKey(ip))

	account := normalizeLoginAccount(req)
	if account != "" {
		delete(g.entries, g.accountKey(ip, account))
	}
}

func (g *loginAttemptGuard) checkKeyLocked(now time.Time, key string, message string) (string, bool) {
	entry, ok := g.entries[key]
	if !ok {
		return "", false
	}
	if entry.LockedUntil.After(now) {
		waitMinutes := int(time.Until(entry.LockedUntil).Minutes())
		if waitMinutes < 1 {
			waitMinutes = 1
		}
		return fmt.Sprintf("%s（剩余约 %d 分钟）", message, waitMinutes), true
	}
	return "", false
}

func (g *loginAttemptGuard) bumpKey(now time.Time, key string, limit int) {
	entry, ok := g.entries[key]
	if !ok || now.Sub(entry.LastFailed) > loginAttemptWindow {
		entry = &loginAttemptEntry{}
		g.entries[key] = entry
	}
	entry.Failed++
	entry.LastFailed = now
	if entry.Failed >= limit {
		entry.LockedUntil = now.Add(loginAttemptLockDuration)
	}
}

func (g *loginAttemptGuard) cleanupExpired(now time.Time) {
	for key, entry := range g.entries {
		if entry == nil {
			delete(g.entries, key)
			continue
		}
		if entry.LockedUntil.After(now) {
			continue
		}
		if now.Sub(entry.LastFailed) > loginAttemptWindow {
			delete(g.entries, key)
		}
	}
}

func (g *loginAttemptGuard) failedCount(key string, now time.Time) int {
	entry, ok := g.entries[key]
	if !ok || entry == nil {
		return 0
	}
	if now.Sub(entry.LastFailed) > loginAttemptWindow {
		return 0
	}
	return entry.Failed
}

func (g *loginAttemptGuard) ipKey(ip string) string {
	return "ip:" + strings.TrimSpace(ip)
}

func (g *loginAttemptGuard) accountKey(ip string, account string) string {
	return "account:" + strings.TrimSpace(ip) + ":" + account
}

func normalizeLoginAccount(req *dto.AuthSignin) string {
	if req == nil {
		return ""
	}
	if value := strings.TrimSpace(strings.ToLower(req.Email)); value != "" {
		return value
	}
	return strings.TrimSpace(req.Mobile)
}
