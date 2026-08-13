package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestWebsiteDiagnosticConsumerIgnoresTmpAndRecoversProcessing(t *testing.T) {
	database := setupWebsiteDiagnosticTestDB(t)
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	t.Cleanup(func() { global.CONF.System.BaseDir = oldBaseDir })
	website := model.Website{Alias: "example.com", PrimaryDomain: "example.com", Type: "static", Status: "Running", Protocol: "HTTP"}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	setting := defaultWebsiteDiagnosticSetting(website.ID)
	setting.Enabled, setting.BackendHook = true, true
	if err := database.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}
	trackingDir, err := ensureWebsiteTrackingDirs(website.Alias)
	if err != nil {
		t.Fatal(err)
	}
	event := WebsiteDiagnosticEnvelope{Schema: websiteDiagnosticSchema, EventID: "evt-1", WebsiteID: website.ID, Source: "backend", Kind: "runtime_error", Severity: "error", OccurredAt: time.Now()}
	if _, err = WriteWebsiteDiagnosticEvent(website.Alias, event); err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(trackingDir, "inbox", "evt-1.ready")
	processing := filepath.Join(trackingDir, "processing", "evt-1.ready")
	if err = os.Rename(ready, processing); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(trackingDir, "inbox", "ignored.tmp"), []byte("{}"), 0640); err != nil {
		t.Fatal(err)
	}
	if err = NewWebsiteDiagnosticConsumer().RunOnce(); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err = database.Model(&model.WebsiteDiagnosticEvent{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("event count=%d err=%v", count, err)
	}
	if _, err = os.Stat(filepath.Join(trackingDir, "processed", "evt-1.ready")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(trackingDir, "inbox", "ignored.tmp")); err != nil {
		t.Fatal(err)
	}
}

func TestWebsiteDiagnosticConsumerRejectsInvalidSchema(t *testing.T) {
	database := setupWebsiteDiagnosticTestDB(t)
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	t.Cleanup(func() { global.CONF.System.BaseDir = oldBaseDir })
	website := model.Website{Alias: "invalid.example", PrimaryDomain: "invalid.example", Type: "static", Status: "Running", Protocol: "HTTP"}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	setting := defaultWebsiteDiagnosticSetting(website.ID)
	setting.Enabled, setting.BackendHook = true, true
	if err := database.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}
	trackingDir, err := ensureWebsiteTrackingDirs(website.Alias)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(trackingDir, "inbox", "bad.ready"), []byte(`{"schema":"bad"}`), 0640); err != nil {
		t.Fatal(err)
	}
	if err = NewWebsiteDiagnosticConsumer().RunOnce(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(trackingDir, "rejected", "bad.ready")); err != nil {
		t.Fatal(err)
	}
}
