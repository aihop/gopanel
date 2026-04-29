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

// 获取数据库表列表（带分页和搜索）
func (s *DBManagerService) GetTableList(req request.GetTableListReq) (map[string]interface{}, error) {
	offset := (req.Page - 1) * req.Limit

	server, err := s.serverRepo.Get(req.ServerID)
	if err != nil {
		return nil, err
	}

	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	countSQL, countArgs, dataSQL, dataArgs, err := buildTableListQueries(server.Type, req.DatabaseName, req.Keyword, req.SortField, req.SortOrder, req.Limit, offset)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := db.QueryRow(countSQL, countArgs...).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := db.Query(dataSQL, dataArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		entry := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				entry[col] = string(b)
			} else {
				entry[col] = val
			}
		}
		items = append(items, entry)
	}

	return map[string]interface{}{
		"items": items,
		"total": total,
		"page":  req.Page,
		"limit": req.Limit,
	}, nil
}

func buildTableListQueries(dbType model.DatabaseType, databaseName, keyword, sortField, sortOrder string, limit, offset int) (string, []interface{}, string, []interface{}, error) {
	keyword = strings.TrimSpace(keyword)
	sortSQL := buildTableListSortSQL(dbType, sortField, sortOrder)

	switch dbType {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		whereClauses := []string{"TABLE_SCHEMA = ?", "TABLE_TYPE = 'BASE TABLE'"}
		countArgs := []interface{}{databaseName}
		dataArgs := []interface{}{databaseName}
		if keyword != "" {
			whereClauses = append(whereClauses, "TABLE_NAME LIKE ?")
			likeKeyword := "%" + keyword + "%"
			countArgs = append(countArgs, likeKeyword)
			dataArgs = append(dataArgs, likeKeyword)
		}
		whereSQL := strings.Join(whereClauses, " AND ")
		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.TABLES WHERE %s", whereSQL)
		dataSQL := `SELECT
TABLE_NAME AS name,
TABLE_TYPE AS tableType,
ENGINE AS engine,
TABLE_ROWS AS rowCount,
COALESCE(DATA_LENGTH, 0) + COALESCE(INDEX_LENGTH, 0) AS sizeBytes,
TABLE_COLLATION AS collation,
CREATE_TIME AS createTime,
UPDATE_TIME AS updateTime,
TABLE_COMMENT AS comment
FROM information_schema.TABLES
WHERE %s
%s
LIMIT ? OFFSET ?`
		dataArgs = append(dataArgs, limit, offset)
		return countSQL, countArgs, fmt.Sprintf(dataSQL, whereSQL, sortSQL), dataArgs, nil
	case model.DatabaseTypePostgresql:
		whereClauses := []string{
			"n.nspname NOT IN ('pg_catalog', 'information_schema')",
			"c.relkind IN ('r', 'p')",
			"pg_catalog.pg_table_is_visible(c.oid)",
		}
		countArgs := make([]interface{}, 0)
		paramIndex := 1
		if keyword != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("c.relname ILIKE $%d", paramIndex))
			countArgs = append(countArgs, "%"+keyword+"%")
			paramIndex++
		}
		whereSQL := strings.Join(whereClauses, " AND ")
		countSQL := fmt.Sprintf(`SELECT COUNT(*)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE %s`, whereSQL)
		dataArgs := append([]interface{}{}, countArgs...)
		dataSQL := `SELECT
c.relname AS name,
CASE c.relkind
	WHEN 'r' THEN 'TABLE'
	WHEN 'p' THEN 'PARTITIONED TABLE'
	ELSE c.relkind::text
END AS "tableType",
NULL::text AS engine,
CASE WHEN c.relkind IN ('r', 'p') THEN COALESCE(s.n_live_tup::bigint, c.reltuples::bigint, 0) ELSE NULL END AS "rowCount",
CASE WHEN c.relkind IN ('r', 'p', 'm') THEN pg_total_relation_size(c.oid) ELSE NULL END AS "sizeBytes",
NULL::text AS collation,
NULL::timestamp AS "createTime",
NULL::timestamp AS "updateTime",
pg_catalog.obj_description(c.oid, 'pg_class') AS comment
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_stat_user_tables s ON s.relid = c.oid
WHERE %s
%s
LIMIT $%d OFFSET $%d`
		dataArgs = append(dataArgs, limit, offset)
		return countSQL, countArgs, fmt.Sprintf(dataSQL, whereSQL, sortSQL, paramIndex, paramIndex+1), dataArgs, nil
	case model.DatabaseSQLite:
		whereClauses := []string{"type = 'table'", "name NOT LIKE 'sqlite_%'"}
		countArgs := make([]interface{}, 0)
		dataArgs := make([]interface{}, 0)
		if keyword != "" {
			whereClauses = append(whereClauses, "name LIKE ?")
			likeKeyword := "%" + keyword + "%"
			countArgs = append(countArgs, likeKeyword)
			dataArgs = append(dataArgs, likeKeyword)
		}
		whereSQL := strings.Join(whereClauses, " AND ")
		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE %s", whereSQL)
		dataSQL := `SELECT
name,
'TABLE' AS tableType,
NULL AS engine,
NULL AS rowCount,
NULL AS sizeBytes,
NULL AS collation,
NULL AS createTime,
NULL AS updateTime,
'' AS comment
FROM sqlite_master
WHERE %s
%s
LIMIT ? OFFSET ?`
		dataArgs = append(dataArgs, limit, offset)
		return countSQL, countArgs, fmt.Sprintf(dataSQL, whereSQL, sortSQL), dataArgs, nil
	default:
		return "", nil, "", nil, fmt.Errorf("unsupported database type for manager: %s", dbType)
	}
}

func buildTableListSortSQL(dbType model.DatabaseType, sortField, sortOrder string) string {
	order := "ASC"
	switch strings.ToLower(strings.TrimSpace(sortOrder)) {
	case "descend", "desc":
		order = "DESC"
	}

	field := strings.TrimSpace(sortField)
	if field == "" {
		switch dbType {
		case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
			return "ORDER BY TABLE_NAME ASC"
		case model.DatabaseTypePostgresql:
			return "ORDER BY c.relname ASC"
		case model.DatabaseSQLite:
			return "ORDER BY name ASC"
		default:
			return "ORDER BY 1 ASC"
		}
	}

	switch dbType {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		switch field {
		case "rowCount":
			return fmt.Sprintf("ORDER BY TABLE_ROWS %s, TABLE_NAME ASC", order)
		case "sizeBytes":
			return fmt.Sprintf("ORDER BY (COALESCE(DATA_LENGTH, 0) + COALESCE(INDEX_LENGTH, 0)) %s, TABLE_NAME ASC", order)
		default:
			return "ORDER BY TABLE_NAME ASC"
		}
	case model.DatabaseTypePostgresql:
		switch field {
		case "rowCount":
			return fmt.Sprintf("ORDER BY COALESCE(s.n_live_tup::bigint, c.reltuples::bigint, 0) %s, c.relname ASC", order)
		case "sizeBytes":
			return fmt.Sprintf("ORDER BY pg_total_relation_size(c.oid) %s, c.relname ASC", order)
		default:
			return "ORDER BY c.relname ASC"
		}
	case model.DatabaseSQLite:
		return "ORDER BY name ASC"
	default:
		return "ORDER BY 1 ASC"
	}
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
	offset := (req.Page - 1) * req.Limit

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
		// SQLite rowid aliases are virtual selectors, not real table columns.
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
	removeVirtualColumns(req.Data)

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
	removeVirtualColumns(req.Data)

	var setCols []string
	var args []interface{}
	paramOffset := 1
	stripComplexConditions(req.Conditions)

	rowidValue, hasRowid := popRowidCondition(req.Conditions)
	idValue, hasID := popCondition(req.Conditions, "id")
	if hasID && idValue != nil {
		req.Conditions = map[string]interface{}{"id": idValue}
	}
	if dbType == model.DatabaseSQLite {
		if hasRowid && rowidValue != nil {
			if rowid, ok := normalizeRowid(rowidValue); ok {
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
	stripComplexConditions(req.Conditions)

	rowidValue, hasRowid := popRowidCondition(req.Conditions)
	idValue, hasID := popCondition(req.Conditions, "id")
	if hasID && idValue != nil {
		req.Conditions = map[string]interface{}{"id": idValue}
	}
	if dbType == model.DatabaseSQLite {
		if hasRowid && rowidValue != nil {
			if rowid, ok := normalizeRowid(rowidValue); ok {
				_, err = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE rowid = ?", quoteTable(dbType, tableName)), rowid)
				return err
			}
		}
	}

	whereSql, args := buildWhereClause(req.Conditions, 1, dbType)
	sqlStr := fmt.Sprintf("DELETE FROM %s WHERE %s", quoteTable(dbType, tableName), whereSql)

	_, err = db.Exec(sqlStr, args...)
	return err
}
