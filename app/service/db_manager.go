package service

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	gomysql "github.com/go-sql-driver/mysql"
)

// buildManagerDSN 安全地构造数据库连接串。用驱动自带的配置/URL 编码来转义账号密码，
// 避免密码含 @ : / ? # 等特殊字符时破坏连接串（此前用 fmt.Sprintf 直接拼接会连不上库）。
// dbName 为空时：MySQL 连到无默认库，PostgreSQL 连到 postgres 库。
func buildManagerDSN(server *model.DatabaseServer, dbName string, multiStatements bool) (driver string, dsn string, err error) {
	switch server.Type {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		cfg := gomysql.NewConfig()
		cfg.User = server.Username
		cfg.Passwd = server.Password
		cfg.Net = "tcp"
		cfg.Addr = fmt.Sprintf("%s:%d", server.Host, server.Port)
		cfg.DBName = dbName
		cfg.Params = map[string]string{"charset": "utf8mb4"}
		cfg.ParseTime = true
		cfg.Loc = time.Local
		cfg.MultiStatements = multiStatements
		return "mysql", cfg.FormatDSN(), nil
	case model.DatabaseTypePostgresql:
		if dbName == "" {
			dbName = "postgres"
		}
		u := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(server.Username, server.Password),
			Host:     fmt.Sprintf("%s:%d", server.Host, server.Port),
			Path:     "/" + dbName,
			RawQuery: "sslmode=disable",
		}
		return "pgx", u.String(), nil
	case model.DatabaseSQLite:
		return "sqlite", server.Host, nil
	default:
		return "", "", fmt.Errorf("unsupported database type for manager: %s", server.Type)
	}
}

type DBManagerService struct{ serverRepo *repo.DatabaseServerRepo }

func NewDBManagerService() *DBManagerService {
	return &DBManagerService{serverRepo: repo.NewDatabaseServer()}
}
func (s *DBManagerService) getDBConn(serverID uint, databaseName string) (*sql.DB, error) {
	server, err := s.serverRepo.Get(serverID)
	if err != nil {
		return nil, err
	}
	driver, dsn, err := buildManagerDSN(server, databaseName, false)
	if err != nil {
		return nil, err
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

// getRawDBConn 连接数据库服务器，不指定具体数据库（用于创建/删除数据库等操作）
func (s *DBManagerService) getRawDBConn(serverID uint) (*sql.DB, error) {
	server, err := s.serverRepo.Get(serverID)
	if err != nil {
		return nil, err
	}
	if server.Type == model.DatabaseSQLite {
		return nil, fmt.Errorf("sqlite does not support create/drop database via this API")
	}
	// 不指定具体库（MySQL 连到无默认库、PG 连到 postgres 库），MySQL 开启 multiStatements
	driver, dsn, err := buildManagerDSN(server, "", true)
	if err != nil {
		return nil, err
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

		// SQLite 自增必须写成 "INTEGER PRIMARY KEY AUTOINCREMENT" 内联形式，
		// 且不能再配独立的 PRIMARY KEY 约束，否则语法错误。这里单独处理并跳过后续拼接。
		if col.AutoIncrement && dbType == model.DatabaseSQLite {
			colDefs = append(colDefs, fmt.Sprintf("  %s%s%s INTEGER PRIMARY KEY AUTOINCREMENT", q, colName, q))
			continue
		}

		// 注：列长度由前端并入 col.Type（如 VARCHAR(255)），此处不再单独处理 col.Length
		colDef := fmt.Sprintf("  %s%s%s %s", q, colName, q, col.Type)
		if !col.Nullable {
			colDef += " NOT NULL"
		}
		if col.AutoIncrement {
			switch dbType {
			case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
				colDef += " AUTO_INCREMENT"
			case model.DatabaseTypePostgresql:
				// PG 用标准 IDENTITY 实现自增（要求列为整数类型）
				colDef += " GENERATED BY DEFAULT AS IDENTITY"
				// SQLite 的自增在上面已内联处理
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

// ChangeTableOwner 修改表的所有者，仅 PostgreSQL 有真正的表级 owner 概念
func (s *DBManagerService) ChangeTableOwner(req request.ChangeTableOwnerReq) error {
	server, err := s.serverRepo.Get(req.ServerID)
	if err != nil {
		return err
	}
	if server.Type != model.DatabaseTypePostgresql {
		return fmt.Errorf("当前数据库类型不支持修改表所有者")
	}
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s OWNER TO %s", quoteTable(server.Type, req.TableName), quoteIdent(server.Type, req.Owner)))
	return err
}
