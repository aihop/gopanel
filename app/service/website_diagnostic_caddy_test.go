package service

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestCollectWebsiteCaddyEventsSkipsMalformedLines(t *testing.T) {
	database := setupWebsiteDiagnosticTestDB(t)
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	t.Cleanup(func() { global.CONF.System.BaseDir = oldBaseDir })
	website := model.Website{Alias: "logs.example", PrimaryDomain: "logs.example", Protocol: "HTTP"}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	setting := defaultWebsiteDiagnosticSetting(website.ID)
	setting.Enabled = true
	if err := database.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureWebsiteLogDir(website.Alias); err != nil {
		t.Fatal(err)
	}
	content := "not-json\n" + `{"ts":` + strconv.FormatFloat(float64(time.Now().Unix()), 'f', -1, 64) + `,"status":500,"duration":0.25,"request":{"method":"GET","uri":"/api/orders/12345"}}` + "\n"
	if err := os.WriteFile(websiteAccessLogPath(website.Alias), []byte(content), 0640); err != nil {
		t.Fatal(err)
	}
	if err := collectWebsiteCaddyEvents(&website, &setting); err != nil {
		t.Fatal(err)
	}
	var events []model.WebsiteDiagnosticEvent
	if err := database.Find(&events).Error; err != nil || len(events) != 1 {
		t.Fatalf("events=%#v err=%v", events, err)
	}
	if events[0].Kind != "http_5xx" || events[0].Route != "/api/orders/12345" {
		t.Fatalf("event=%#v", events[0])
	}
	if err := collectWebsiteCaddyEvents(&website, &setting); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.Model(&model.WebsiteDiagnosticEvent{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestCollectWebsiteCaddyEventsPreservesPartialLine(t *testing.T) {
	database := setupWebsiteDiagnosticTestDB(t)
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	t.Cleanup(func() { global.CONF.System.BaseDir = oldBaseDir })
	website := model.Website{Alias: "partial.example", PrimaryDomain: "partial.example", Protocol: "HTTP"}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	setting := defaultWebsiteDiagnosticSetting(website.ID)
	setting.Enabled = true
	if err := database.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureWebsiteLogDir(website.Alias); err != nil {
		t.Fatal(err)
	}
	partial := `{"status":500,"request":{"method":"GET","uri":"/partial"}}`
	if err := os.WriteFile(websiteAccessLogPath(website.Alias), []byte(partial), 0640); err != nil {
		t.Fatal(err)
	}
	if err := collectWebsiteCaddyEvents(&website, &setting); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(websiteAccessLogPath(website.Alias), []byte(partial+"\n"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := collectWebsiteCaddyEvents(&website, &setting); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.Model(&model.WebsiteDiagnosticEvent{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
