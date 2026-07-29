package api

import (
	"fmt"
	"testing"
	"time"
)

func TestMobilePairingAttemptsAreRateLimited(t *testing.T) {
	resetMobilePairingAttempts(t)
	ip := fmt.Sprintf("test-%d", time.Now().UnixNano())
	now := time.Now()
	for attempt := 0; attempt < 20; attempt++ {
		if !allowMobilePairingAttempt(ip, now) {
			t.Fatalf("attempt %d should be allowed", attempt+1)
		}
	}
	if allowMobilePairingAttempt(ip, now) {
		t.Fatal("expected the twenty-first attempt to be rejected")
	}
	if !allowMobilePairingAttempt(ip, now.Add(time.Minute)) {
		t.Fatal("expected a new window to allow pairing")
	}
}

func TestMobilePairingAttemptCacheIsBounded(t *testing.T) {
	resetMobilePairingAttempts(t)
	now := time.Now()
	for index := 0; index < maxMobilePairingAttemptEntries; index++ {
		if !allowMobilePairingAttempt(fmt.Sprintf("bounded-%d", index), now) {
			t.Fatalf("entry %d should be allowed", index)
		}
	}
	if allowMobilePairingAttempt("bounded-overflow", now) {
		t.Fatal("expected a new address to be rejected when the cache is full")
	}
	if !allowMobilePairingAttempt("bounded-overflow", now.Add(time.Minute)) {
		t.Fatal("expected expired entries to be cleaned")
	}
}

func resetMobilePairingAttempts(t *testing.T) {
	t.Helper()
	mobilePairingAttempts.Lock()
	oldItems := mobilePairingAttempts.items
	oldCleanup := mobilePairingAttempts.lastCleanup
	mobilePairingAttempts.items = make(map[string]mobilePairingAttempt)
	mobilePairingAttempts.lastCleanup = time.Time{}
	mobilePairingAttempts.Unlock()
	t.Cleanup(func() {
		mobilePairingAttempts.Lock()
		mobilePairingAttempts.items = oldItems
		mobilePairingAttempts.lastCleanup = oldCleanup
		mobilePairingAttempts.Unlock()
	})
}
