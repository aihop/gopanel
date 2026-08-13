package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
)

const systemDiagnosticRowLimit = 100

var systemDiagnosticTableReferencePattern = regexp.MustCompile(`(?i)\b(?:from|join)\s+[\x60\x22\[]?([a-zA-Z_][a-zA-Z0-9_]*)`)

var systemDiagnosticAllowedTables = map[string]bool{
	"ai_approvals": true, "ai_dev_sessions": true, "ai_execution_runs": true,
	"ai_instructions": true, "ai_tasks": true, "alert_events": true,
	"app_deploys": true, "backup_records": true, "cronjobs": true,
	"database_servers": true, "flow_runs": true, "flow_stage_runs": true, "flows": true,
	"host_terminal_audit_events": true, "host_terminal_sessions": true, "job_records": true,
	"operation_logs": true, "pipeline_records": true, "pipelines": true, "releases": true,
	"security_analysis_runs": true, "security_events": true, "website_diagnostic_events": true,
	"website_diagnostic_probes": true, "website_diagnostic_timeline": true,
	"website_issues": true, "websites": true,
}

var systemDiagnosticBlockedIdentifiers = map[string]bool{
	"access_key": true, "api_key": true, "authorization": true, "cookie": true,
	"codex_api_key": true, "credential": true, "hook_secret_encrypted": true,
	"content": true, "output": true, "password": true, "private_key": true,
	"prompt": true, "provider_api_key": true, "raw_output": true, "salt": true,
	"script": true, "secret": true, "token": true, "vars": true,
}

func systemDiagnosticSQLFingerprint(statement string) string {
	normalized := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(statement))), " ")
	digest := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(digest[:])
}

func listSystemDiagnosticTables(keyword string) ([]string, error) {
	if global.DB == nil {
		return nil, errors.New("GoPanel 数据库不可用")
	}
	tables, err := global.DB.Migrator().GetTables()
	if err != nil {
		return nil, err
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	allowed := make([]string, 0, len(tables))
	for _, table := range tables {
		lower := strings.ToLower(table)
		if !systemDiagnosticAllowedTables[lower] || (keyword != "" && !strings.Contains(lower, keyword)) {
			continue
		}
		allowed = append(allowed, table)
	}
	return allowed, nil
}

func describeSystemDiagnosticTable(table string) ([]map[string]any, error) {
	if global.DB == nil {
		return nil, errors.New("GoPanel 数据库不可用")
	}
	table = strings.ToLower(strings.TrimSpace(table))
	if table == "" || !systemDiagnosticAllowedTables[table] {
		return nil, errors.New("诊断中心不允许读取该表")
	}
	allowedTables, err := listSystemDiagnosticTables("")
	if err != nil {
		return nil, err
	}
	found := false
	for _, allowed := range allowedTables {
		if strings.EqualFold(allowed, table) {
			table, found = allowed, true
			break
		}
	}
	if !found {
		return nil, errors.New("诊断表不存在或不可读取")
	}
	columnTypes, err := global.DB.Migrator().ColumnTypes(table)
	if err != nil {
		return nil, err
	}
	columns := make([]map[string]any, 0, len(columnTypes))
	for _, column := range columnTypes {
		if isSystemDiagnosticSensitiveIdentifier(column.Name()) {
			continue
		}
		columns = append(columns, map[string]any{"name": column.Name(), "type": column.DatabaseTypeName()})
	}
	return columns, nil
}

func querySystemDiagnosticDatabase(statement string) (map[string]any, error) {
	if global.DB == nil {
		return nil, errors.New("GoPanel 数据库不可用")
	}
	statement = strings.TrimSpace(statement)
	if len(statement) > 8000 {
		return nil, errors.New("诊断 SQL 过长")
	}
	if err := service.ValidateCodeReadOnlySQL(statement); err != nil {
		return nil, err
	}
	identifiers := systemDiagnosticSQLIdentifiers(statement)
	if len(identifiers) == 0 || identifiers[0] != "select" {
		return nil, errors.New("诊断中心只允许 SELECT 查询")
	}
	lowerStatement := strings.ToLower(statement)
	for _, fragment := range []string{
		"sqlite_", "pragma_", "information_schema", "pg_catalog", "load_extension(",
		"readfile(", "writefile(", "randomblob(", "zeroblob(", "generate_series(",
	} {
		if strings.Contains(lowerStatement, fragment) {
			return nil, errors.New("诊断 SQL 包含禁止访问的系统对象或函数")
		}
	}
	for _, match := range systemDiagnosticTableReferencePattern.FindAllStringSubmatch(statement, -1) {
		if len(match) < 2 || !systemDiagnosticAllowedTables[strings.ToLower(match[1])] {
			return nil, fmt.Errorf("诊断中心不允许读取表 %s", match[1])
		}
	}
	allTables, err := global.DB.Migrator().GetTables()
	if err != nil {
		return nil, err
	}
	knownTables := make(map[string]bool, len(allTables))
	for _, table := range allTables {
		knownTables[strings.ToLower(table)] = true
	}
	for _, identifier := range identifiers {
		if knownTables[identifier] && !systemDiagnosticAllowedTables[identifier] {
			return nil, fmt.Errorf("诊断中心不允许读取表 %s", identifier)
		}
		if isSystemDiagnosticSensitiveIdentifier(identifier) {
			return nil, fmt.Errorf("诊断中心不允许读取敏感字段 %s", identifier)
		}
	}
	requestCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := global.DB.WithContext(requestCtx).Raw(statement).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	for _, column := range columns {
		if isSystemDiagnosticSensitiveIdentifier(column) {
			return nil, fmt.Errorf("查询结果包含敏感字段 %s", column)
		}
	}
	items := make([]map[string]any, 0, systemDiagnosticRowLimit)
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for index := range values {
		pointers[index] = &values[index]
	}
	truncated := false
	for rows.Next() {
		if len(items) >= systemDiagnosticRowLimit {
			truncated = true
			break
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		item := make(map[string]any, len(columns))
		for index, column := range columns {
			item[column] = sanitizeSystemDiagnosticValue(values[index])
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"columns": columns, "rows": items, "limit": systemDiagnosticRowLimit, "truncated": truncated}, nil
}

func isSystemDiagnosticSensitiveIdentifier(identifier string) bool {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	if systemDiagnosticBlockedIdentifiers[identifier] {
		return true
	}
	for _, fragment := range []string{"api_key", "access_key", "authorization", "cookie", "credential", "password", "private_key", "secret"} {
		if strings.Contains(identifier, fragment) {
			return true
		}
	}
	return false
}

func systemDiagnosticSQLIdentifiers(statement string) []string {
	identifiers := make([]string, 0, 32)
	start := -1
	for index, character := range statement {
		valid := character == '_' || unicode.IsLetter(character) || unicode.IsDigit(character)
		if valid && start < 0 {
			start = index
		} else if !valid && start >= 0 {
			identifiers = append(identifiers, strings.ToLower(statement[start:index]))
			start = -1
		}
	}
	if start >= 0 {
		identifiers = append(identifiers, strings.ToLower(statement[start:]))
	}
	return identifiers
}

func sanitizeSystemDiagnosticValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return sanitizeSystemDiagnosticText(string(typed))
	case string:
		return sanitizeSystemDiagnosticText(typed)
	default:
		return value
	}
}

func sanitizeSystemDiagnosticText(value string) string {
	value = service.ScrubSecurityLogText(value)
	runes := []rune(value)
	if len(runes) > 2000 {
		return string(runes[:2000]) + "…"
	}
	return value
}
