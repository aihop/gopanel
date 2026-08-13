package repo

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSecurityEventDebounceDedupeRecovery(t *testing.T) {
	previous := global.DB
	t.Cleanup(func() { global.DB = previous })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "security.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = database
	repository := NewSecurityMonitoring()
	if err := repository.MigrateTable(); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	candidate := &model.SecurityEvent{
		SourceType: "website", SourceID: 7, SourceName: "example.com", EventType: "not_found_scan",
		Level: "medium", Fingerprint: "stable", Summary: "scan", FirstSeenAt: now, LastSeenAt: now,
	}
	first, fired, err := repository.UpsertEvent(candidate, 2)
	if err != nil || fired || first.Status != model.SecurityEventPending {
		t.Fatalf("first upsert: event=%#v fired=%v err=%v", first, fired, err)
	}
	second, fired, err := repository.UpsertEvent(candidate, 2)
	if err != nil || !fired || second.ID != first.ID || second.Status != model.SecurityEventFiring {
		t.Fatalf("second upsert: event=%#v fired=%v err=%v", second, fired, err)
	}
	second.AIConclusion, second.SuggestedActions, second.NotifyStatus = "old conclusion", `[{"action":"old"}]`, "sent"
	second.Confidence, second.LastNotifiedAt = 88, &now
	if err := repository.SaveEvent(second); err != nil {
		t.Fatal(err)
	}
	resolved, err := repository.ResolveStale(now.Add(time.Minute))
	if err != nil || len(resolved) != 1 || resolved[0].Status != model.SecurityEventResolved {
		t.Fatalf("resolve: %#v err=%v", resolved, err)
	}
	reopened, fired, err := repository.UpsertEvent(candidate, 2)
	if err != nil || fired || reopened.ID != first.ID || reopened.Status != model.SecurityEventPending || reopened.HitCount != 1 ||
		reopened.AIConclusion != "" || reopened.NotifyStatus != "" || reopened.LastNotifiedAt != nil {
		t.Fatalf("reopen: event=%#v fired=%v err=%v", reopened, fired, err)
	}
}
