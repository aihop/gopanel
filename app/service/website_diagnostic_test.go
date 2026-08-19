package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestWebsiteTrackingDirRejectsTraversal(t *testing.T) {
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	t.Cleanup(func() { global.CONF.System.BaseDir = oldBaseDir })

	for _, alias := range []string{"..", "../outside", "/"} {
		if _, err := websiteTrackingDir(alias); err == nil {
			t.Fatalf("tracking directory accepted unsafe alias %q", alias)
		}
	}
	dir, err := websiteTrackingDir("gopanel.cn")
	if err != nil {
		t.Fatalf("valid alias rejected: %v", err)
	}
	want := filepath.Join(global.CONF.System.BaseDir, "log", "website", "gopanel.cn", "tracking")
	if dir != want {
		t.Fatalf("tracking directory = %q, want %q", dir, want)
	}
}

func TestEnsureWebsiteTrackingDirs(t *testing.T) {
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	t.Cleanup(func() { global.CONF.System.BaseDir = oldBaseDir })

	dir, err := ensureWebsiteTrackingDirs("example.com")
	if err != nil {
		t.Fatalf("ensure tracking directories: %v", err)
	}
	for _, name := range []string{"inbox", "processing", "processed", "rejected", "attachments"} {
		info, statErr := os.Stat(filepath.Join(dir, name))
		if statErr != nil || !info.IsDir() {
			t.Fatalf("tracking directory %s missing: %v", name, statErr)
		}
	}
}

func TestNormalizeWebsiteDiagnosticSetting(t *testing.T) {
	if err := normalizeWebsiteDiagnosticSetting(nil); err == nil {
		t.Fatal("nil setting was accepted")
	}
	setting := defaultWebsiteDiagnosticSetting(1)
	setting.Enabled = true
	if err := normalizeWebsiteDiagnosticSetting(&setting); err != nil {
		t.Fatalf("default enabled setting rejected: %v", err)
	}

	setting.CaddyMonitoring = false
	if err := normalizeWebsiteDiagnosticSetting(&setting); err == nil {
		t.Fatal("enabled setting without a monitoring source was accepted")
	}

	setting = defaultWebsiteDiagnosticSetting(1)
	setting.AutoAnalysis = true
	if err := normalizeWebsiteDiagnosticSetting(&setting); err == nil {
		t.Fatal("auto analysis without a Code project was accepted")
	}

	setting = model.WebsiteDiagnosticSetting{WebsiteID: 1, CaddyMonitoring: true}
	if err := normalizeWebsiteDiagnosticSetting(&setting); err == nil {
		t.Fatal("zero thresholds were accepted")
	}

	setting = defaultWebsiteDiagnosticSetting(1)
	setting.DefaultExecutorID = "grok"
	setting.ApprovalPolicy = "safe_auto"
	if err := normalizeWebsiteDiagnosticSetting(&setting); err != nil {
		t.Fatalf("Grok diagnostic setting rejected: %v", err)
	}

	setting = defaultWebsiteDiagnosticSetting(1)
	setting.DefaultExecutorID = "opencode"
	setting.ApprovalPolicy = "manual"
	if err := normalizeWebsiteDiagnosticSetting(&setting); err == nil {
		t.Fatal("unsupported executor approval policy was accepted")
	}
}
