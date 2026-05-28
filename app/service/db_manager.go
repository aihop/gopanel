package service

import (
	"database/sql"
	"fmt"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DBManagerService struct{ serverRepo *repo.DatabaseServerRepo }

func NewDBManagerService() *DBManagerService {
	return &DBManagerService{serverRepo: repo.NewDatabaseServer()}
}
func (s *DBManagerService) getDBConn(serverID uint, databaseName string) (*sql.DB, error) {
	server, err := s.serverRepo.Get(serverID)
	if err != nil {
		return nil, err
	}
	var dsn string
	var driver string
	switch server.Type {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		driver = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", server.Username, server.Password, server.Host, server.Port, databaseName)
	case model.DatabaseTypePostgresql:
		driver = "pgx"
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", server.Username, server.Password, server.Host, server.Port, databaseName)
	case model.DatabaseSQLite:
		driver = "sqlite"
		dsn = fmt.Sprintf("%s", server.Host)
	default:
		return nil, fmt.Errorf("unsupported database type for manager: %s", server.Type)
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func (s *DBManagerService) ExecSql(req request.ExecSqlReq) (map[string]interface{}, error) {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	sqlUpper := strings.ToUpper(strings.TrimSpace(req.SQL))
	isQuery := strings.HasPrefix(sqlUpper, "SELECT") || strings.HasPrefix(sqlUpper, "SHOW") || strings.HasPrefix(sqlUpper, "EXPLAIN") || strings.HasPrefix(sqlUpper, "DESCRIBE") || strings.HasPrefix(sqlUpper, "PRAGMA")
	if !isQuery {
		result, err := db.Exec(req.SQL)
		if err != nil {
			return nil, err
		}
		affected, _ := result.RowsAffected()
		return map[string]interface{}{"type": "exec", "affected": affected}, nil
	}
	rows, err := db.Query(req.SQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	count := len(columns)
	var tableData []map[string]interface{}
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for i := 0; i < count; i++ {
		valuePtrs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}
	return map[string]interface{}{"type": "query", "columns": columns, "rows": tableData}, nil
}
func (s *DBManagerService) GetTableData(req request.GetTableDataReq) (map[string]interface{}, error) {
	offset := (req.Page - 1) * req.Limit
	tableName := sanitizeIdent(req.TableName)
	server, _ := s.serverRepo.Get(req.ServerID)
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var countSql, dataSql string
	var whereClauses []string
	if req.SearchColumn != "" && req.SearchValue != "" {
		col := sanitizeIdent(req.SearchColumn)
		val := strings.ReplaceAll(req.SearchValue, "'", "''")
		whereClauses = append(whereClauses, fmt.Sprintf("%s LIKE '%%%s%%'", quoteIdent(server.Type, col), val))
	}
	for _, cond := range req.AdvancedSearch {
		col := sanitizeIdent(cond.Column)
		val := strings.ReplaceAll(cond.Value, "'", "''")
		op := strings.ToUpper(cond.Operator)
		validOps := map[string]bool{"=": true, "!=": true, ">": true, "<": true, ">=": true, "<=": true, "LIKE": true, "NOT LIKE": true, "IS NULL": true, "IS NOT NULL": true}
		if !validOps[op] {
			op = "="
		}
		var clause string
		if op == "IS NULL" || op == "IS NOT NULL" {
			clause = fmt.Sprintf("%s %s", quoteIdent(server.Type, col), op)
		} else {
			if op == "LIKE" || op == "NOT LIKE" {
				if !strings.Contains(val, "%") {
					val = "%" + val + "%"
				}
			}
			clause = fmt.Sprintf("%s %s '%s'", quoteIdent(server.Type, col), op, val)
		}
		whereClauses = append(whereClauses, clause)
	}
	var whereClause string
	if len(whereClauses) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauses, " AND ")
	}
	countSql = fmt.Sprintf("SELECT COUNT(*) FROM %s%s", quoteTable(server.Type, tableName), whereClause)
	selectCols := "*"
	if server.Type == model.DatabaseSQLite && sqliteTableHasRowid(db, tableName) {
		selectCols = "rowid AS \"__rowid__\", *"
	}
	dataSql = fmt.Sprintf("SELECT %s FROM %s%s LIMIT %d OFFSET %d", selectCols, quoteTable(server.Type, tableName), whereClause, req.Limit, offset)
	var total int64
	if err := db.QueryRow(countSql).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := db.Query(dataSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	count := len(columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for i := 0; i < count; i++ {
		valuePtrs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		entry := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				entry[col] = string(b)
			} else {
				entry[col] = val
			}
		}
		tableData = append(tableData, entry)
	}
	return map[string]interface{}{"type": "query", "columns": columns, "rows": tableData, "total": total}, nil
}
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

func (s *DBManagerService) ImportTable(req request.ImportTableReq) (int, error) {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	tableName := sanitizeIdent(req.TableName)
	q := string(quoteChar(server.Type))
	quote := func(name string) string {
		return q + name + q
	}

	switch req.Format {
	case "sql":
		return execSQLImport(db, req.Content)
	case "csv":
		// Normalize line endings (support Windows \r\n)
		normalized := strings.ReplaceAll(req.Content, "\r\n", "\n")
		normalized = strings.ReplaceAll(normalized, "\r", "")
		rows := strings.Split(strings.TrimSpace(normalized), "\n")
		if len(rows) < 2 {
			return 0, fmt.Errorf("csv must have at least a header row and one data row")
		}

		// Parse header
		headers := parseCSVFields(strings.TrimSpace(rows[0]))
		if len(headers) == 0 {
			return 0, fmt.Errorf("no columns found in CSV header")
		}

		imported := 0
		for _, line := range rows[1:] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			values := parseCSVFields(line)
			if len(values) != len(headers) {
				continue
			}

			// Build INSERT
			var cols, placeholders []string
			var args []interface{}
			for i, header := range headers {
				col := sanitizeIdent(header)
				if col == "" {
					continue
				}
				cols = append(cols, quote(col))
				placeholders = append(placeholders, "?")
				if values[i] == "" && isNumericTypeGuess(server.Type, col) {
					args = append(args, nil)
				} else {
					args = append(args, values[i])
				}
			}

			if len(cols) == 0 {
				continue
			}

			sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
				quote(tableName),
				strings.Join(cols, ", "),
				strings.Join(placeholders, ", "))

			_, err := db.Exec(sqlStr, args...)
			if err != nil {
				continue
			}
			imported++
		}

		return imported, nil
	default:
		return 0, fmt.Errorf("unsupported import format: %s", req.Format)
	}
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

func execSQLImport(db *sql.DB, content string) (int, error) {
	// Split by semicolons and execute each statement
	statements := strings.Split(content, ";")
	executed := 0
	for _, stmt := range statements {
		// Remove comment lines and trim
		lines := strings.Split(stmt, "\n")
		var cleanLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				continue
			}
			cleanLines = append(cleanLines, line)
		}
		cleanStmt := strings.TrimSpace(strings.Join(cleanLines, "\n"))
		if cleanStmt == "" {
			continue
		}

		_, err := db.Exec(cleanStmt)
		if err != nil {
			return executed, fmt.Errorf("statement %d failed: %v", executed+1, err)
		}
		executed++
	}
	return executed, nil
}

func (s *DBManagerService) ExportTable(req request.ExportTableReq) (string, string, error) {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	tableName := sanitizeIdent(req.TableName)
	q := string(quoteChar(server.Type))
	quote := func(name string) string {
		return q + name + q
	}

	// Get columns info
	colSQL := fmt.Sprintf("SELECT %s FROM %s LIMIT 0", "*", quote(tableName))
	colRows, err := db.Query(colSQL)
	if err != nil {
		return "", "", err
	}
	columns, err := colRows.Columns()
	colRows.Close()
	if err != nil {
		return "", "", err
	}

	// Get all data
	dataSQL := fmt.Sprintf("SELECT %s FROM %s", "*", quote(tableName))
	rows, err := db.Query(dataSQL)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()

	if req.Format == "sql" {
		dump := generateSQLDump(server.Type, tableName, columns, rows, quote)
		filename := fmt.Sprintf("%s_%s.sql", req.DatabaseName, tableName)
		return dump, filename, nil
	}

	// CSV format
	csv := generateCSV(columns, rows)
	filename := fmt.Sprintf("%s_%s.csv", req.DatabaseName, tableName)
	return csv, filename, nil
}

func quoteChar(dbType model.DatabaseType) string {
	if dbType == model.DatabaseTypeMysql || dbType == model.DatabaseTypeMariaDB {
		return "`"
	}
	return `"`
}

func generateSQLDump(dbType model.DatabaseType, tableName string, columns []string, rows *sql.Rows, quote func(string) string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("-- GoPanel Export: %s\n", tableName))
	b.WriteString(fmt.Sprintf("-- Date: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

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
