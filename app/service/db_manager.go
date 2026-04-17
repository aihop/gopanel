package service

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBManagerService struct {
	serverRepo *repo.DatabaseServerRepo
}

func NewDBManagerService() *DBManagerService {
	return &DBManagerService{
		serverRepo: repo.NewDatabaseServer(),
	}
}

// 获取原始数据库连接
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
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			server.Username, server.Password, server.Host, server.Port, databaseName)
	case model.DatabaseTypePostgresql:
		driver = "pgx"
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			server.Username, server.Password, server.Host, server.Port, databaseName)
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

// 获取所有表名
func (s *DBManagerService) GetTables(req request.GetTablesReq) ([]string, error) {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	var query string
	switch server.Type {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		query = "SHOW TABLES"
	case model.DatabaseTypePostgresql:
		query = "SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname != 'pg_catalog' AND schemaname != 'information_schema'"
	case model.DatabaseSQLite:
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	default:
		return nil, fmt.Errorf("unsupported database type for manager: %s", server.Type)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if server.Type == model.DatabaseSQLite && strings.Contains(query, "PRAGMA") {
			var schema, typeStr string
			var ncol, wr, strict int
			// 扫描全部 6 列，但只取我们需要的一列
			if err := rows.Scan(&schema, &tableName, &typeStr, &ncol, &wr, &strict); err != nil {
				return nil, err
			}
			// 过滤掉系统内部表
			if schema != "main" || typeStr != "table" {
				continue
			}
		} else {
			if err := rows.Scan(&tableName); err != nil {
				return nil, err
			}
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

// 通用执行 SQL 并返回动态结构
func (s *DBManagerService) ExecSql(req request.ExecSqlReq) (map[string]interface{}, error) {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 简单区分查询和执行
	sqlUpper := strings.ToUpper(strings.TrimSpace(req.SQL))
	isQuery := strings.HasPrefix(sqlUpper, "SELECT") ||
		strings.HasPrefix(sqlUpper, "SHOW") ||
		strings.HasPrefix(sqlUpper, "EXPLAIN") ||
		strings.HasPrefix(sqlUpper, "DESCRIBE") ||
		strings.HasPrefix(sqlUpper, "PRAGMA")

	if !isQuery {
		result, err := db.Exec(req.SQL)
		if err != nil {
			return nil, err
		}
		affected, _ := result.RowsAffected()
		return map[string]interface{}{
			"type":     "exec",
			"affected": affected,
		}, nil
	}

	// 执行查询
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

	return map[string]interface{}{
		"type":    "query",
		"columns": columns,
		"rows":    tableData,
	}, nil
}

// 获取表数据（带分页）
func (s *DBManagerService) GetTableData(req request.GetTableDataReq) (map[string]interface{}, error) {
	offset := (req.Page - 1) * req.PageSize

	// 防注入简单处理
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
		val := strings.ReplaceAll(req.SearchValue, "'", "''") // 简单转义单引号
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
	dataSql = fmt.Sprintf("SELECT %s FROM %s%s LIMIT %d OFFSET %d", selectCols, quoteTable(server.Type, tableName), whereClause, req.PageSize, offset)

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

	return map[string]interface{}{
		"type":    "query",
		"columns": columns,
		"rows":    tableData,
		"total":   total,
	}, nil
}

// 构建 WHERE 条件和参数
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

// 插入记录
func (s *DBManagerService) InsertRecord(req request.InsertRecordReq) error {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type
	tableName := sanitizeIdent(req.TableName)

	var cols []string
	var placeholders []string
	var args []interface{}
	paramOffset := 1

	for k, v := range req.Data {
		col := sanitizeIdent(k)
		if col == "" {
			continue
		}
		cols = append(cols, quoteIdent(dbType, col))
		if dbType == model.DatabaseTypePostgresql {
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramOffset))
		} else {
			placeholders = append(placeholders, "?")
		}
		args = append(args, v)
		paramOffset++
	}

	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoteTable(dbType, tableName), strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	_, err = db.Exec(sqlStr, args...)
	return err
}

// 更新记录
func (s *DBManagerService) UpdateRecord(req request.UpdateRecordReq) error {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type
	tableName := sanitizeIdent(req.TableName)

	var setCols []string
	var args []interface{}
	paramOffset := 1

	if dbType == model.DatabaseSQLite {
		if v, ok := req.Conditions["__rowid__"]; ok && v != nil {
			delete(req.Conditions, "__rowid__")
			if rowid, ok := normalizeRowid(v); ok {
				for k, val := range req.Data {
					col := sanitizeIdent(k)
					if col == "" {
						continue
					}
					setCols = append(setCols, fmt.Sprintf("%s = ?", quoteIdent(dbType, col)))
					args = append(args, val)
				}
				args = append(args, rowid)
				sqlStr := fmt.Sprintf("UPDATE %s SET %s WHERE rowid = ?", quoteTable(dbType, tableName), strings.Join(setCols, ", "))
				_, err = db.Exec(sqlStr, args...)
				return err
			}
		}
		delete(req.Conditions, "__rowid__")
	}

	for k, v := range req.Data {
		col := sanitizeIdent(k)
		if col == "" {
			continue
		}
		if dbType == model.DatabaseTypePostgresql {
			setCols = append(setCols, fmt.Sprintf("%s = $%d", quoteIdent(dbType, col), paramOffset))
		} else {
			setCols = append(setCols, fmt.Sprintf("%s = ?", quoteIdent(dbType, col)))
		}
		args = append(args, v)
		paramOffset++
	}

	whereSql, whereArgs := buildWhereClause(req.Conditions, paramOffset, dbType)
	args = append(args, whereArgs...)

	sqlStr := fmt.Sprintf("UPDATE %s SET %s WHERE %s", quoteTable(dbType, tableName), strings.Join(setCols, ", "), whereSql)

	_, err = db.Exec(sqlStr, args...)
	return err
}

// 删除记录
func (s *DBManagerService) DeleteRecord(req request.DeleteRecordReq) error {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return err
	}
	defer db.Close()

	server, _ := s.serverRepo.Get(req.ServerID)
	dbType := server.Type
	tableName := sanitizeIdent(req.TableName)

	if dbType == model.DatabaseSQLite {
		if v, ok := req.Conditions["__rowid__"]; ok && v != nil {
			delete(req.Conditions, "__rowid__")
			if rowid, ok := normalizeRowid(v); ok {
				_, err = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE rowid = ?", quoteTable(dbType, tableName)), rowid)
				return err
			}
		}
		delete(req.Conditions, "__rowid__")
	}

	whereSql, args := buildWhereClause(req.Conditions, 1, dbType)
	sqlStr := fmt.Sprintf("DELETE FROM %s WHERE %s", quoteTable(dbType, tableName), whereSql)

	_, err = db.Exec(sqlStr, args...)
	return err
}
