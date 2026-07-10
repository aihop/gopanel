package service

import (
	"fmt"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"strings"
)

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
			if err := rows.Scan(&schema, &tableName, &typeStr, &ncol, &wr, &strict); err != nil {
				return nil, err
			}
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
	return map[string]interface{}{"items": items, "total": total, "page": req.Page, "limit": req.Limit}, nil
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
		whereClauses := []string{"n.nspname NOT IN ('pg_catalog', 'information_schema')", "c.relkind IN ('r', 'p')", "pg_catalog.pg_table_is_visible(c.oid)"}
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
pg_catalog.obj_description(c.oid, 'pg_class') AS comment,
pg_catalog.pg_get_userbyid(c.relowner) AS owner
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
		case "name":
			return fmt.Sprintf("ORDER BY TABLE_NAME %s", order)
		case "rowCount":
			return fmt.Sprintf("ORDER BY TABLE_ROWS %s, TABLE_NAME ASC", order)
		case "sizeBytes":
			return fmt.Sprintf("ORDER BY (COALESCE(DATA_LENGTH, 0) + COALESCE(INDEX_LENGTH, 0)) %s, TABLE_NAME ASC", order)
		case "updateTime":
			return fmt.Sprintf("ORDER BY UPDATE_TIME %s, TABLE_NAME ASC", order)
		default:
			return "ORDER BY TABLE_NAME ASC"
		}
	case model.DatabaseTypePostgresql:
		switch field {
		case "name":
			return fmt.Sprintf("ORDER BY c.relname %s", order)
		case "rowCount":
			return fmt.Sprintf("ORDER BY COALESCE(s.n_live_tup::bigint, c.reltuples::bigint, 0) %s, c.relname ASC", order)
		case "sizeBytes":
			return fmt.Sprintf("ORDER BY pg_total_relation_size(c.oid) %s, c.relname ASC", order)
		case "updateTime":
			return "ORDER BY c.relname ASC"
		default:
			return "ORDER BY c.relname ASC"
		}
	case model.DatabaseSQLite:
		switch field {
		case "name":
			return fmt.Sprintf("ORDER BY name %s", order)
		case "updateTime":
			return "ORDER BY name ASC"
		default:
			return "ORDER BY name ASC"
		}
	default:
		return "ORDER BY 1 ASC"
	}
}
