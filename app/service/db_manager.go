package service

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
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

// ImportSQLContent 导入原始 SQL 内容（用于分片上传场景，仅支持 SQL 格式）
func (s *DBManagerService) ImportSQLContent(serverID uint, databaseName, content string) (int, error) {
	db, err := s.getDBConn(serverID, databaseName)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	return execSQLImport(db, content)
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

	// 确定导出的列
	var colList string
	var exportColumns []string
	if len(req.Columns) > 0 {
		quotedCols := make([]string, len(req.Columns))
		for i, c := range req.Columns {
			quotedCols[i] = quote(sanitizeIdent(c))
		}
		colList = strings.Join(quotedCols, ", ")
		exportColumns = req.Columns
	} else {
		colList = "*"
		// 用 LIMIT 0 获取列名
		colSQL := fmt.Sprintf("SELECT %s FROM %s LIMIT 0", colList, quote(tableName))
		colRows, err := db.Query(colSQL)
		if err != nil {
			return "", "", err
		}
		exportColumns, err = colRows.Columns()
		colRows.Close()
		if err != nil {
			return "", "", err
		}
	}

	// 构建数据查询 SQL
	dataSQL := fmt.Sprintf("SELECT %s FROM %s", colList, quote(tableName))
	if req.Where != "" {
		dataSQL += " WHERE " + req.Where
	}
	rows, err := db.Query(dataSQL)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()

	if req.Format == "sql" {
		dump := generateSQLDump(db, server.Type, tableName, req, exportColumns, rows, quote)
		filename := fmt.Sprintf("%s_%s.sql", req.DatabaseName, tableName)
		return dump, filename, nil
	}

	// CSV 格式
	csv := generateCSV(exportColumns, rows)
	filename := fmt.Sprintf("%s_%s.csv", req.DatabaseName, tableName)
	return csv, filename, nil
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
		"columns":   columns,
		"rows":      results,
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
		"name":   req.DatabaseName,
		"type":   server.Type,
		"server": server.Name,
	}

	switch server.Type {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		db, err := s.getDBConn(req.ServerID, req.DatabaseName)
		if err != nil {
			return info, nil // 返回部分信息
		}
		defer db.Close()

		// 字符集和排序规则
		var charset, collation string
		db.QueryRow("SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", req.DatabaseName).Scan(&charset, &collation)
		info["charset"] = charset
		info["collation"] = collation

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

// CopyTable 复制表（结构 + 可选数据）
func (s *DBManagerService) CopyTable(req request.CopyTableReq) error {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type
	src := sanitizeIdent(req.SourceTable)
	dst := sanitizeIdent(req.TargetTable)
	q := string(quoteChar(dbType))
	quote := func(name string) string { return q + name + q }

	switch dbType {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		// MySQL: CREATE TABLE ... LIKE （复制结构、索引、约束）
		_, err = db.Exec(fmt.Sprintf("CREATE TABLE %s LIKE %s", quote(dst), quote(src)))
	case model.DatabaseTypePostgresql:
		// PostgreSQL: CREATE TABLE ... (LIKE ... INCLUDING ALL)
		_, err = db.Exec(fmt.Sprintf("CREATE TABLE %s (LIKE %s INCLUDING ALL)", quote(dst), quote(src)))
	case model.DatabaseSQLite:
		// SQLite: 从 sqlite_master 获取 CREATE TABLE SQL 并替换表名
		var createSQL sql.NullString
		if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", src).Scan(&createSQL); err != nil {
			return fmt.Errorf("source table not found: %v", err)
		}
		if !createSQL.Valid {
			return fmt.Errorf("source table %s has no CREATE definition", req.SourceTable)
		}
		modified := strings.Replace(createSQL.String, fmt.Sprintf("%q", src), fmt.Sprintf("%q", dst), 1)
		if modified == createSQL.String {
			modified = strings.Replace(createSQL.String, fmt.Sprintf("`%s`", src), fmt.Sprintf("`%s`", dst), 1)
		}
		if modified == createSQL.String {
			modified = strings.Replace(createSQL.String, src, dst, 1)
		}
		_, err = db.Exec(modified)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	if err != nil {
		return fmt.Errorf("create table structure failed: %v", err)
	}

	// 复制数据
	if req.CopyData {
		_, err = db.Exec(fmt.Sprintf("INSERT INTO %s SELECT * FROM %s", quote(dst), quote(src)))
		if err != nil {
			return fmt.Errorf("copy data failed: %v", err)
		}
	}

	return nil
}
