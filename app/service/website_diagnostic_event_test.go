package service

import (
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupWebsiteDiagnosticTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = database.AutoMigrate(
		&model.Website{}, &model.WebsiteDomain{}, &model.WebsiteUpstream{}, &model.AppDeploy{},
		&model.WebsiteDiagnosticSetting{}, &model.WebsiteDiagnosticEvent{},
		&model.WebsiteIssue{}, &model.WebsiteDiagnosticTimeline{}, &model.WebsiteProbe{},
		&model.WebsiteDiagnosticNonce{},
	); err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = oldDB })
	return database
}

func TestNormalizeWebsiteDiagnosticEnvelopeSanitizesEvidence(t *testing.T) {
	event, err := normalizeDiagnosticEnvelope(&WebsiteDiagnosticEnvelope{
		Schema: websiteDiagnosticSchema, EventID: "evt-1", Source: "backend", Kind: "business_error",
		Message: "Authorization: bearer-secret password=hello", Route: "/orders/9231?token=abc",
		Metadata: map[string]interface{}{"cookie": "session=secret", "safe": "ok"},
	}, 7)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(event.Message, "bearer-secret") || strings.Contains(event.Message, "hello") {
		t.Fatalf("message was not sanitized: %s", event.Message)
	}
	if event.Route != "/orders/9231" {
		t.Fatalf("route = %q", event.Route)
	}
	if strings.Contains(event.Metadata, "session=secret") {
		t.Fatalf("metadata was not sanitized: %s", event.Metadata)
	}
}

func TestWebsiteDiagnosticEventIdempotencyAndAggregation(t *testing.T) {
	database := setupWebsiteDiagnosticTestDB(t)
	repository := repo.NewWebsiteDiagnostic(database)
	now := time.Now()
	first := &model.WebsiteDiagnosticEvent{
		WebsiteID: 1, EventID: "evt-1", Source: "backend", Kind: "runtime_error", Severity: "error",
		Fingerprint: "same", Title: "failure", Route: "/api/orders/1", SessionID: "s1", OccurredAt: now,
	}
	issue, created, err := repository.IngestEvent(first)
	if err != nil || !created || issue.OccurrenceCount != 1 {
		t.Fatalf("first ingest: issue=%#v created=%v err=%v", issue, created, err)
	}
	duplicate := *first
	duplicate.ID = 0
	issue, created, err = repository.IngestEvent(&duplicate)
	if err != nil || created || issue.OccurrenceCount != 1 {
		t.Fatalf("duplicate ingest: issue=%#v created=%v err=%v", issue, created, err)
	}
	second := *first
	second.ID = 0
	second.EventID = "evt-2"
	second.SessionID = "s2"
	second.Release = "v2"
	issue, created, err = repository.IngestEvent(&second)
	if err != nil || created || issue.OccurrenceCount != 2 || issue.SessionCount != 2 {
		t.Fatalf("aggregate ingest: issue=%#v created=%v err=%v", issue, created, err)
	}
}

func TestIngestWebsiteDiagnosticEnvelopeAppliesSourceAndContentSettings(t *testing.T) {
	database := setupWebsiteDiagnosticTestDB(t)
	setting := defaultWebsiteDiagnosticSetting(9)
	setting.Enabled, setting.BackendHook = true, true
	setting.MonitorBusinessErrors = false
	if err := database.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}

	event := &WebsiteDiagnosticEnvelope{
		Schema: websiteDiagnosticSchema, EventID: "filtered-content", WebsiteID: 9,
		Source: "backend", Kind: "business_error", Severity: "error", OccurredAt: time.Now(),
	}
	if _, _, err := ingestWebsiteDiagnosticEnvelope(9, event); err == nil || !strings.Contains(err.Error(), "ErrWebsiteDiagnosticEventFiltered") {
		t.Fatalf("expected content filter error, got %v", err)
	}

	event.EventID, event.Source, event.Kind = "filtered-source", "browser", "vue_error"
	if _, _, err := ingestWebsiteDiagnosticEnvelope(9, event); err == nil || !strings.Contains(err.Error(), "ErrWebsiteDiagnosticEventFiltered") {
		t.Fatalf("expected source filter error, got %v", err)
	}

	setting.MonitorBusinessErrors = true
	if err := database.Save(&setting).Error; err != nil {
		t.Fatal(err)
	}
	event.EventID, event.Source, event.Kind = "allowed", "backend", "business_error"
	if issue, created, err := ingestWebsiteDiagnosticEnvelope(9, event); err != nil || !created || issue == nil {
		t.Fatalf("allowed ingest: issue=%#v created=%v err=%v", issue, created, err)
	}
	event.EventID, event.Kind = "allowed-runtime", "runtime_error"
	if issue, created, err := ingestWebsiteDiagnosticEnvelope(9, event); err != nil || !created || issue == nil {
		t.Fatalf("runtime ingest: issue=%#v created=%v err=%v", issue, created, err)
	}
}
