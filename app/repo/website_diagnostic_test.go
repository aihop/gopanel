package repo

import (
	"context"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestWebsiteDiagnosticRepositoryPreservesDisabledOptions(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.WebsiteDiagnosticSetting{}); err != nil {
		t.Fatal(err)
	}
	repository := NewWebsiteDiagnostic(database)
	setting := &model.WebsiteDiagnosticSetting{
		WebsiteID: 9, SlowRequestThresholdMS: 1500, TriggerCount: 5,
		TriggerWindowMinutes: 10, RetentionDays: 7, DefaultExecutorID: "codex", ApprovalPolicy: "safe_auto",
	}
	if err := repository.Save(setting); err != nil {
		t.Fatal(err)
	}
	saved, err := repository.GetByWebsiteID(setting.WebsiteID)
	if err != nil || saved == nil {
		t.Fatalf("load setting: setting=%#v err=%v", saved, err)
	}
	if saved.CaddyMonitoring || saved.MonitorHTTP4xx || saved.MonitorHTTP5xx {
		t.Fatalf("disabled options changed during create: %#v", saved)
	}

	setting.Enabled = true
	setting.CaddyMonitoring = true
	setting.MonitorHTTP5xx = true
	if err := repository.Save(setting); err != nil {
		t.Fatal(err)
	}
	saved, err = repository.GetByWebsiteID(setting.WebsiteID)
	if err != nil || !saved.Enabled || !saved.CaddyMonitoring || !saved.MonitorHTTP5xx {
		t.Fatalf("upsert did not persist changes: setting=%#v err=%v", saved, err)
	}
	if err := repository.DeleteByWebsiteID(context.Background(), setting.WebsiteID); err != nil {
		t.Fatal(err)
	}
	saved, err = repository.GetByWebsiteID(setting.WebsiteID)
	if err != nil || saved != nil {
		t.Fatalf("delete left setting behind: setting=%#v err=%v", saved, err)
	}
}
