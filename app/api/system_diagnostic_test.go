package api

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func withSystemDiagnosticDB(t *testing.T) *gorm.DB {
	t.Helper()
	previous := global.DB
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "diagnostic.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
	return database
}

func TestSystemDiagnosticDatabaseAllowsOperationalFacts(t *testing.T) {
	database := withSystemDiagnosticDB(t)
	if err := database.AutoMigrate(&model.BackupRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.BackupRecord{Type: "mysql", Name: "main", DetailName: "app", Source: "LOCAL", FileName: "app.sql.gz"}).Error; err != nil {
		t.Fatal(err)
	}
	result, err := querySystemDiagnosticDatabase("SELECT id, type, name, detail_name, file_name FROM backup_records ORDER BY id DESC")
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := result["rows"].([]map[string]any)
	if !ok || len(rows) != 1 || rows[0]["file_name"] != "app.sql.gz" {
		t.Fatalf("unexpected diagnostic rows: %#v", result)
	}
}

func TestSystemDiagnosticDatabaseBlocksSecrets(t *testing.T) {
	database := withSystemDiagnosticDB(t)
	if err := database.AutoMigrate(&model.DatabaseServer{}, &model.AIProviderAccount{}); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"SELECT password FROM database_servers",
		"SELECT * FROM database_servers",
		"SELECT id, name FROM ai_provider_accounts",
		"SELECT content FROM ai_messages",
		"SELECT content FROM user_notes",
		"SELECT sql FROM sqlite_master",
		"SELECT * FROM pragma_table_info('database_servers')",
		"SELECT readfile('/etc/passwd')",
		"WITH data AS (SELECT * FROM backup_records) SELECT * FROM data",
		"UPDATE backup_records SET name = 'changed'",
	} {
		if _, err := querySystemDiagnosticDatabase(statement); err == nil {
			t.Fatalf("unsafe diagnostic SQL was accepted: %s", statement)
		}
	}
}

func TestSystemDiagnosticTableDescriptionHidesSecrets(t *testing.T) {
	database := withSystemDiagnosticDB(t)
	if err := database.AutoMigrate(&model.DatabaseServer{}); err != nil {
		t.Fatal(err)
	}
	columns, err := describeSystemDiagnosticTable("database_servers")
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool, len(columns))
	for _, column := range columns {
		names[column["name"].(string)] = true
	}
	if !names["name"] || !names["host"] || names["password"] {
		t.Fatalf("unsafe diagnostic schema: %#v", columns)
	}
}

func TestSystemDiagnosticDatabaseScrubsAndTruncatesText(t *testing.T) {
	database := withSystemDiagnosticDB(t)
	if err := database.AutoMigrate(&model.JobRecords{}); err != nil {
		t.Fatal(err)
	}
	longSecret := "token=secret-value " + strings.Repeat("x", 2200)
	if err := database.Create(&model.JobRecords{Status: "failed", Message: longSecret}).Error; err != nil {
		t.Fatal(err)
	}
	result, err := querySystemDiagnosticDatabase("SELECT message FROM job_records")
	if err != nil {
		t.Fatal(err)
	}
	rows := result["rows"].([]map[string]any)
	message := rows[0]["message"].(string)
	if strings.Contains(message, "secret-value") || !strings.Contains(message, "[REDACTED]") || len([]rune(message)) > 2001 {
		t.Fatalf("diagnostic text was not safely bounded: %q", message)
	}
}

func TestSystemDiagnosticPromptEnforcesReadOnlyEvidenceBasedAnalysis(t *testing.T) {
	for _, required := range []string{"GoPanel", "describe_panel_table", "query_panel_database", "当前诊断中心只读", "高风险动作", "验证与回滚"} {
		if !strings.Contains(systemDiagnosticPrompt, required) {
			t.Fatalf("system diagnostic prompt missing %q", required)
		}
	}
}
