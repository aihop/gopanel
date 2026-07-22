package service

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
)

func sanitizeIdent(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "`", "")
	value = strings.ReplaceAll(value, "\"", "")
	return strings.TrimSpace(value)
}
func quoteIdent(dbType model.DatabaseType, ident string) string {
	ident = sanitizeIdent(ident)
	if dbType == model.DatabaseTypeMysql || dbType == model.DatabaseTypeMariaDB {
		return fmt.Sprintf("`%s`", ident)
	}
	return fmt.Sprintf("\"%s\"", ident)
}
func quoteTable(dbType model.DatabaseType, table string) string {
	table = sanitizeIdent(table)
	if dbType == model.DatabaseTypeMysql || dbType == model.DatabaseTypeMariaDB {
		return fmt.Sprintf("`%s`", table)
	}
	return fmt.Sprintf("\"%s\"", table)
}
func buildWhereClause(conditions map[string]interface{}, paramOffset int, dbType model.DatabaseType) (string, []interface{}) {
	var where []string
	var args []interface{}
	keys := make([]string, 0, len(conditions))
	for k := range conditions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := conditions[k]
		col := sanitizeIdent(k)
		if col == "" {
			continue
		}
		if v == nil {
			where = append(where, fmt.Sprintf("%s IS NULL", quoteIdent(dbType, col)))
		} else {
			if dbType == model.DatabaseTypePostgresql {
				where = append(where, fmt.Sprintf("%s = $%d", quoteIdent(dbType, col), paramOffset))
			} else {
				where = append(where, fmt.Sprintf("%s = ?", quoteIdent(dbType, col)))
			}
			args = append(args, v)
			paramOffset++
		}
	}
	if len(where) == 0 {
		return "1=1", args
	}
	return strings.Join(where, " AND "), args
}
func sqliteTableHasRowid(db *sql.DB, tableName string) bool {
	var sqlText sql.NullString
	err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&sqlText)
	if err != nil || !sqlText.Valid {
		return true
	}
	return !strings.Contains(strings.ToUpper(sqlText.String), "WITHOUT ROWID")
}
func normalizeRowid(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return 0, false
		}
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		s := strings.TrimSpace(fmt.Sprint(value))
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false
		}
		return i, true
	}
}
func popRowidCondition(conditions map[string]interface{}) (interface{}, bool) {
	if conditions == nil {
		return nil, false
	}
	for k, v := range conditions {
		if strings.EqualFold(strings.TrimSpace(k), "__rowid__") {
			delete(conditions, k)
			return v, true
		}
	}
	return nil, false
}
func popCondition(conditions map[string]interface{}, key string) (interface{}, bool) {
	if conditions == nil {
		return nil, false
	}
	for k, v := range conditions {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			delete(conditions, k)
			return v, true
		}
	}
	return nil, false
}
func removeVirtualColumns(values map[string]interface{}) {
	if values == nil {
		return
	}
	for k := range values {
		key := strings.TrimSpace(k)
		if key == "" {
			delete(values, k)
			continue
		}
		if strings.EqualFold(key, "__rowid__") || strings.EqualFold(key, "rowid") {
			delete(values, k)
		}
	}
}
func stripComplexConditions(conditions map[string]interface{}) {
	if conditions == nil {
		return
	}
	for k, v := range conditions {
		switch v.(type) {
		case map[string]interface{}, []interface{}:
			delete(conditions, k)
		}
	}
}

func quoteSQLString(val string) string {
	escaped := strings.ReplaceAll(val, "'", "''")
	escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
	return "'" + escaped + "'"
}

func formatExportValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case int64, int, float64, float32:
		return fmt.Sprint(v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case string:
		return quoteSQLString(v)
	case []byte:
		return quoteSQLString(string(v))
	default:
		return quoteSQLString(fmt.Sprint(val))
	}
}

// parseCSVFields splits a CSV line into fields, handling quoted fields
func parseCSVFields(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false
	for _, ch := range line {
		if ch == '"' {
			inQuotes = !inQuotes
		} else if ch == ',' && !inQuotes {
			fields = append(fields, current.String())
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}
	fields = append(fields, current.String())
	return fields
}

func isNumericTypeGuess(dbType model.DatabaseType, colName string) bool {
	// Simple heuristic: columns named "id" or ending with "_id" might be numeric
	lower := strings.ToLower(colName)
	if lower == "id" {
		return true
	}
	if strings.HasSuffix(lower, "_id") {
		return true
	}
	return false
}

// splitSQLStatements 智能分割 SQL 语句，识别引号和注释内的分号不切割
func splitSQLStatements(content string) []string {
	var statements []string
	var cur strings.Builder
	inSingleQ := false  // '
	inDoubleQ := false  // "
	inBacktick := false // `
	inLineComment := false  // --
	inBlockComment := false // /* */

	runes := []rune(content)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		// 转义字符（MySQL 用 '' 转义单引号）
		if inSingleQ && ch == '\'' && i+1 < len(runes) && runes[i+1] == '\'' {
			cur.WriteString("''")
			i++
			continue
		}
		if inDoubleQ && ch == '"' && i+1 < len(runes) && runes[i+1] == '"' {
			cur.WriteString(`""`)
			i++
			continue
		}

		// 引号切换
		if !inLineComment && !inBlockComment {
			if ch == '\'' && !inDoubleQ && !inBacktick {
				inSingleQ = !inSingleQ
				cur.WriteRune(ch)
				continue
			}
			if ch == '"' && !inSingleQ && !inBacktick {
				inDoubleQ = !inDoubleQ
				cur.WriteRune(ch)
				continue
			}
			if ch == '`' && !inSingleQ && !inDoubleQ {
				inBacktick = !inBacktick
				cur.WriteRune(ch)
				continue
			}
		}

		// 行注释 -- (需要后面跟空格或控制字符)
		if !inSingleQ && !inDoubleQ && !inBacktick && !inBlockComment && !inLineComment &&
			ch == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			if i+2 >= len(runes) || runes[i+2] == ' ' || runes[i+2] == '\t' || runes[i+2] == '\n' || runes[i+2] == '\r' {
				inLineComment = true
				cur.WriteString("--")
				i++
				continue
			}
		}
		if inLineComment && (ch == '\n' || ch == '\r') {
			inLineComment = false
			cur.WriteRune(ch)
			continue
		}

		// 块注释 /* */
		if !inSingleQ && !inDoubleQ && !inBacktick && !inLineComment && !inBlockComment &&
			ch == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			inBlockComment = true
			cur.WriteString("/*")
			i++
			continue
		}
		if inBlockComment && ch == '*' && i+1 < len(runes) && runes[i+1] == '/' {
			inBlockComment = false
			cur.WriteString("*/")
			i++
			continue
		}

		// 分号切分（仅在正常状态下）
		if !inSingleQ && !inDoubleQ && !inBacktick && !inLineComment && !inBlockComment && ch == ';' {
			stmt := strings.TrimSpace(cur.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			cur.Reset()
			continue
		}

		cur.WriteRune(ch)
	}

	// 收尾
	stmt := strings.TrimSpace(cur.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

func execSQLImport(db *sql.DB, content string) (int, error) {
	statements := splitSQLStatements(content)
	executed := 0
	for i, stmt := range statements {
		_, err := db.Exec(stmt)
		if err != nil {
			return executed, fmt.Errorf("statement %d failed: %v", i+1, err)
		}
		executed++
	}
	return executed, nil
}

func quoteChar(dbType model.DatabaseType) string {
	if dbType == model.DatabaseTypeMysql || dbType == model.DatabaseTypeMariaDB {
		return "`"
	}
	return `"`
}

func getCreateTableSQL(db *sql.DB, dbType model.DatabaseType, tableName string, quote func(string) string) string {
	switch dbType {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		row := db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE %s", quote(tableName)))
		var tName, createSQL string
		if err := row.Scan(&tName, &createSQL); err == nil {
			return createSQL + ";\n\n"
		}
	case model.DatabaseSQLite:
		row := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName)
		var createSQL string
		if err := row.Scan(&createSQL); err == nil {
			return createSQL + ";\n\n"
		}
	case model.DatabaseTypePostgresql:
		// PostgreSQL: 从 pg_catalog 重建 CREATE TABLE
		// 注意：默认值必须用 pg_get_expr(adbin, adrelid)，adsrc 列在 PostgreSQL 12 中已被移除
		regclass := quote(tableName)
		rows, err := db.Query(fmt.Sprintf(`
			SELECT a.attname, pg_catalog.format_type(a.atttypid, a.atttypmod), a.attnotnull,
				pg_catalog.pg_get_expr(d.adbin, d.adrelid) AS default_val
			FROM pg_catalog.pg_attribute a
			LEFT JOIN pg_catalog.pg_attrdef d ON a.attrelid = d.adrelid AND a.attnum = d.adnum
			WHERE a.attrelid = '%s'::regclass AND a.attnum > 0 AND NOT a.attisdropped
			ORDER BY a.attnum`, regclass))
		if err != nil {
			return ""
		}
		defer rows.Close()

		var b strings.Builder
		b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", quote(tableName)))
		var cols []string
		for rows.Next() {
			var colName, colType string
			var notNull bool
			var defaultVal *string
			if err := rows.Scan(&colName, &colType, &notNull, &defaultVal); err != nil {
				continue
			}
			def := fmt.Sprintf("  %s %s", quote(colName), colType)
			if notNull {
				def += " NOT NULL"
			}
			// 跳过 nextval(...) 序列默认值：导出文件里没有对应的 CREATE SEQUENCE，
			// 带上它会导致导入到新库时报 "relation xxx_seq does not exist"
			if defaultVal != nil && *defaultVal != "" && !strings.Contains(*defaultVal, "nextval(") {
				def += " DEFAULT " + *defaultVal
			}
			cols = append(cols, def)
		}
		if len(cols) == 0 {
			return ""
		}

		// 主键
		pkRows, err := db.Query(fmt.Sprintf(`
			SELECT a.attname
			FROM pg_catalog.pg_index i
			JOIN pg_catalog.pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
			WHERE i.indrelid = '%s'::regclass AND i.indisprimary
			ORDER BY array_position(i.indkey::int2[], a.attnum)`, regclass))
		if err == nil {
			var pkCols []string
			for pkRows.Next() {
				var colName string
				if err := pkRows.Scan(&colName); err == nil {
					pkCols = append(pkCols, quote(colName))
				}
			}
			pkRows.Close()
			if len(pkCols) > 0 {
				cols = append(cols, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
			}
		}

		b.WriteString(strings.Join(cols, ",\n"))
		b.WriteString("\n);\n\n")
		return b.String()
	}
	return ""
}

func generateSQLDump(db *sql.DB, dbType model.DatabaseType, tableName string, req request.ExportTableReq, columns []string, rows *sql.Rows, quote func(string) string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("-- GoPanel Export: %s\n", tableName))
	b.WriteString(fmt.Sprintf("-- Database: %s\n", req.DatabaseName))
	b.WriteString(fmt.Sprintf("-- Date: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// DROP TABLE
	if req.IncludeDropTable {
		b.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n\n", quote(tableName)))
	}

	// CREATE TABLE
	if req.IncludeCreateTable || req.IncludeDropTable {
		// 重新获取 db 连接来查询表结构
		if createSQL := getCreateTableSQL(db, dbType, tableName, quote); createSQL != "" {
			b.WriteString(createSQL)
		}
	}

	// 只有有数据列时才输出 INSERT
	if len(columns) == 0 {
		return b.String()
	}

	// INSERT header
	b.WriteString(fmt.Sprintf("INSERT INTO %s (", quote(tableName)))
	colParts := make([]string, len(columns))
	for i, col := range columns {
		colParts[i] = quote(col)
	}
	b.WriteString(strings.Join(colParts, ", "))
	b.WriteString(") VALUES\n")

	// Values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var rowValues []string
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		rowParts := make([]string, len(columns))
		for i := range columns {
			rowParts[i] = formatExportValue(values[i])
		}
		rowValues = append(rowValues, "("+strings.Join(rowParts, ", ")+")")
	}

	b.WriteString(strings.Join(rowValues, ",\n"))
	b.WriteString(";\n")
	return b.String()
}

func generateCSV(columns []string, rows *sql.Rows) string {
	var b strings.Builder

	// BOM for Excel to recognize UTF-8
	b.WriteString("\xef\xbb\xbf")

	// Header
	csvCols := make([]string, len(columns))
	for i, col := range columns {
		csvCols[i] = escapeCSVField(col)
	}
	b.WriteString(strings.Join(csvCols, ","))
	b.WriteString("\n")

	// Data rows
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		rowParts := make([]string, len(columns))
		for i := range columns {
			rowParts[i] = formatCSVValue(values[i])
		}
		b.WriteString(strings.Join(rowParts, ","))
		b.WriteString("\n")
	}

	return b.String()
}

func escapeCSVField(field string) string {
	if strings.ContainsAny(field, "\",\n\r") {
		escaped := strings.ReplaceAll(field, `"`, `""`)
		return `"` + escaped + `"`
	}
	return field
}

func formatCSVValue(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return escapeCSVField(v)
	case []byte:
		return escapeCSVField(string(v))
	default:
		return escapeCSVField(fmt.Sprint(val))
	}
}
