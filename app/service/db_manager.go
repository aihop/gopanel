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

// getRawDBConn 连接数据库服务器，不指定具体数据库（用于创建/删除数据库等操作）
func (s *DBManagerService) getRawDBConn(serverID uint) (*sql.DB, error) {
	server, err := s.serverRepo.Get(serverID)
	if err != nil {
		return nil, err
	}
	var dsn string
	var driver string
	switch server.Type {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		driver = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true", server.Username, server.Password, server.Host, server.Port)
	case model.DatabaseTypePostgresql:
		driver = "pgx"
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable", server.Username, server.Password, server.Host, server.Port)
	case model.DatabaseSQLite:
		return nil, fmt.Errorf("sqlite does not support create/drop database via this API")
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

// CreateDatabase 创建数据库
func (s *DBManagerService) CreateDatabase(req request.CreateDatabaseReq) error {
	db, err := s.getRawDBConn(req.ServerID)
	if err != nil {
		return err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type

	ident := sanitizeIdent(req.DatabaseName)
	switch dbType {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		sqlStr := fmt.Sprintf("CREATE DATABASE `%s`", ident)
		if req.Charset != "" {
			sqlStr += fmt.Sprintf(" CHARACTER SET %s", sanitizeIdent(req.Charset))
		}
		if req.Collation != "" {
			sqlStr += fmt.Sprintf(" COLLATE %s", sanitizeIdent(req.Collation))
		}
		_, err = db.Exec(sqlStr)
	case model.DatabaseTypePostgresql:
		// PostgreSQL CREATE DATABASE 不支持在事务内执行
		_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, ident))
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	return err
}

// DropDatabase 删除数据库
func (s *DBManagerService) DropDatabase(req request.DropDatabaseReq) error {
	db, err := s.getRawDBConn(req.ServerID)
	if err != nil {
		return err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type
	ident := sanitizeIdent(req.DatabaseName)

	switch dbType {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", ident))
	case model.DatabaseTypePostgresql:
		// 先终止连接，再删除
		_, err = db.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, ident))
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	return err
}

// GetTableInfo 获取表的创建语句（SHOW CREATE TABLE）
func (s *DBManagerService) GetTableInfo(req request.GetTableInfoReq) (map[string]interface{}, error) {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	tableName := sanitizeIdent(req.TableName)
	q := string(quoteChar(server.Type))

	var sqlStr string
	switch server.Type {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		sqlStr = fmt.Sprintf("SHOW CREATE TABLE %s%s%s", q, tableName, q)
	case model.DatabaseTypePostgresql:
		// 获取建表语句
		sqlStr = fmt.Sprintf(`
			SELECT
				'CREATE TABLE ' || '%s' || '.' || '%s' || ' (' ||
				string_agg(
					col.column_name || ' ' || col.data_type ||
					CASE WHEN col.character_maximum_length IS NOT NULL THEN '(' || col.character_maximum_length || ')' ELSE '' END ||
					CASE WHEN col.is_nullable = 'NO' THEN ' NOT NULL' ELSE '' END ||
					CASE WHEN col.column_default IS NOT NULL THEN ' DEFAULT ' || col.column_default ELSE '' END,
					', '
				) || ');' AS "Create Table"
			FROM information_schema.columns col
			WHERE col.table_catalog = '%s' AND col.table_schema = 'public' AND col.table_name = '%s'
			GROUP BY col.table_catalog, col.table_schema, col.table_name
		`, q, tableName, req.DatabaseName, tableName)
	case model.DatabaseSQLite:
		sqlStr = fmt.Sprintf("SELECT sql AS \"Create Table\" FROM sqlite_master WHERE type='table' AND name='%s'", strings.ReplaceAll(tableName, "'", "''"))
	default:
		return nil, fmt.Errorf("unsupported database type: %s", server.Type)
	}

	rows, err := db.Query(sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
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
		results = append(results, entry)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("table %s not found", req.TableName)
	}

	return map[string]interface{}{
		"columns": columns,
		"rows":    results,
		"createSQL": results[0],
	}, nil
}

// GetDatabaseInfo 获取数据库统计信息
func (s *DBManagerService) GetDatabaseInfo(req request.GetDatabaseInfoReq) (map[string]interface{}, error) {
	server, err := s.serverRepo.Get(req.ServerID)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"name":      req.DatabaseName,
		"type":      server.Type,
		"server":    server.Name,
	}

	switch server.Type {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		db, err := s.getDBConn(req.ServerID, req.DatabaseName)
		if err != nil {
			return info, nil // 返回部分信息
		}
		defer db.Close()

		// 字符集和排序规则
		db.QueryRow("SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", req.DatabaseName).Scan(&info["charset"], &info["collation"])

		// 表数量
		var tableCount int64
		db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'", req.DatabaseName).Scan(&tableCount)
		info["tableCount"] = tableCount

		// 总大小
		var totalSize int64
		db.QueryRow("SELECT COALESCE(SUM(DATA_LENGTH + INDEX_LENGTH), 0) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?", req.DatabaseName).Scan(&totalSize)
		info["totalSizeBytes"] = totalSize

	case model.DatabaseTypePostgresql:
		db, err := s.getDBConn(req.ServerID, req.DatabaseName)
		if err != nil {
			return info, nil
		}
		defer db.Close()

		var tableCount int64
		db.QueryRow("SELECT COUNT(*) FROM pg_catalog.pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema')").Scan(&tableCount)
		info["tableCount"] = tableCount

		// 数据库大小
		var dbSize int64
		db.QueryRow("SELECT COALESCE(pg_database_size($1), 0)", req.DatabaseName).Scan(&dbSize)
		info["totalSizeBytes"] = dbSize

		// 字符集（PostgreSQL encoding）
		var encoding string
		db.QueryRow("SELECT pg_encoding_to_char(encoding) FROM pg_database WHERE datname = $1", req.DatabaseName).Scan(&encoding)
		info["charset"] = encoding

	case model.DatabaseSQLite:
		db, err := s.getDBConn(req.ServerID, req.DatabaseName)
		if err != nil {
			return info, nil
		}
		defer db.Close()

		var tableCount int64
		db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tableCount)
		info["tableCount"] = tableCount

		// SQLite 文件大小
		var fileSize int64
		db.QueryRow("SELECT COALESCE(page_count * page_size, 0) FROM pragma_page_count(), pragma_page_size()").Scan(&fileSize)
		info["totalSizeBytes"] = fileSize
	}

	return info, nil
}

// CreateTable 结构化创建表
func (s *DBManagerService) CreateTable(req request.CreateTableReq) error {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type
	tableName := sanitizeIdent(req.TableName)
	q := string(quoteChar(dbType))

	var b strings.Builder
	b.WriteString(fmt.Sprintf("CREATE TABLE %s%s%s (\n", q, tableName, q))

	var colDefs []string
	var primaryCols []string

	for _, col := range req.Columns {
		colName := sanitizeIdent(col.Name)
		if colName == "" {
			continue
		}

		colDef := fmt.Sprintf("  %s%s%s %s", q, colName, q, col.Type)
		if col.Length != "" {
			// 某些类型不需要长度，但由前端控制
		}
		if !col.Nullable {
			colDef += " NOT NULL"
		}
		if col.AutoIncrement {
			switch dbType {
			case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
				colDef += " AUTO_INCREMENT"
			case model.DatabaseSQLite:
				colDef += " AUTOINCREMENT"
			}
		}
		if col.DefaultValue != "" {
			defaultVal := col.DefaultValue
			// 判断是否是函数/表达式
			upper := strings.ToUpper(defaultVal)
			if upper == "CURRENT_TIMESTAMP" || upper == "NOW()" || strings.Contains(upper, "(") {
				colDef += " DEFAULT " + defaultVal
			} else {
				// 数字不做引号
				if _, err := strconv.ParseFloat(defaultVal, 64); err == nil {
					colDef += " DEFAULT " + defaultVal
				} else {
					colDef += " DEFAULT '" + strings.ReplaceAll(defaultVal, "'", "''") + "'"
				}
			}
		}
		if col.Comment != "" && (dbType == model.DatabaseTypeMysql || dbType == model.DatabaseTypeMariaDB) {
			colDef += " COMMENT '" + strings.ReplaceAll(col.Comment, "'", "''") + "'"
		}

		colDefs = append(colDefs, colDef)

		if col.IsPrimary {
			primaryCols = append(primaryCols, fmt.Sprintf("%s%s%s", q, colName, q))
		}
	}

	if len(primaryCols) > 0 {
		colDefs = append(colDefs, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(primaryCols, ", ")))
	}

	b.WriteString(strings.Join(colDefs, ",\n"))
	b.WriteString("\n)")

	// MySQL/MariaDB 引擎、字符集
	switch dbType {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		if req.Engine != "" {
			b.WriteString(fmt.Sprintf(" ENGINE=%s", req.Engine))
		}
		if req.Charset != "" {
			b.WriteString(fmt.Sprintf(" DEFAULT CHARSET=%s", req.Charset))
		}
		if req.Collation != "" {
			b.WriteString(fmt.Sprintf(" COLLATE=%s", req.Collation))
		}
		if req.Comment != "" {
			b.WriteString(fmt.Sprintf(" COMMENT='%s'", strings.ReplaceAll(req.Comment, "'", "''")))
		}
	case model.DatabaseTypePostgresql:
		// PostgreSQL 不指定 engine
	}

	_, err = db.Exec(b.String())
	return err
}
