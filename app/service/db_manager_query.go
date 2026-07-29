package service

import (
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
)

func (s *DBManagerService) ExecSql(req request.ExecSqlReq) (map[string]interface{}, error) {
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	sqlUpper := strings.ToUpper(strings.TrimSpace(req.SQL))
	isQuery := strings.HasPrefix(sqlUpper, "SELECT") || strings.HasPrefix(sqlUpper, "SHOW") ||
		strings.HasPrefix(sqlUpper, "EXPLAIN") || strings.HasPrefix(sqlUpper, "DESCRIBE") ||
		strings.HasPrefix(sqlUpper, "DESC ") || strings.HasPrefix(sqlUpper, "PRAGMA") ||
		strings.HasPrefix(sqlUpper, "WITH") || strings.HasPrefix(sqlUpper, "VALUES") ||
		strings.Contains(sqlUpper, " RETURNING ")
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
	tableData, err := scanRows(rows, columns)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"type": "query", "columns": columns, "rows": tableData}, nil
}

func (s *DBManagerService) GetTableData(req request.GetTableDataReq) (map[string]interface{}, error) {
	tableName := sanitizeIdent(req.TableName)
	server, err := s.serverRepo.Get(req.ServerID)
	if err != nil {
		return nil, err
	}
	db, err := s.getDBConn(req.ServerID, req.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	selectCols := "*"
	if server.Type == model.DatabaseSQLite && sqliteTableHasRowid(db, tableName) {
		selectCols = "rowid AS \"__rowid__\", *"
	}
	whereClause, args := buildTableSearchClause(server.Type, req.SearchColumn, req.SearchValue, req.AdvancedSearch)
	offset := (req.Page - 1) * req.Limit
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", quoteTable(server.Type, tableName), whereClause)
	dataSQL := fmt.Sprintf("SELECT %s FROM %s%s LIMIT %d OFFSET %d", selectCols, quoteTable(server.Type, tableName), whereClause, req.Limit, offset)

	var total int64
	if err := db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := db.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	tableData, err := scanRows(rows, columns)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"type": "query", "columns": columns, "rows": tableData, "total": total}, nil
}

func buildTableSearchClause(dbType model.DatabaseType, searchColumn, searchValue string, advancedSearch []request.SearchCondition) (string, []interface{}) {
	clauses := make([]string, 0, len(advancedSearch)+1)
	args := make([]interface{}, 0, len(advancedSearch)+1)
	addParam := func(value interface{}) string {
		args = append(args, value)
		if dbType == model.DatabaseTypePostgresql {
			return fmt.Sprintf("$%d", len(args))
		}
		return "?"
	}

	if column := sanitizeIdent(searchColumn); column != "" && searchValue != "" {
		clauses = append(clauses, buildTextSearchCondition(dbType, column, "LIKE", addParam("%"+searchValue+"%")))
	}
	validOps := map[string]bool{"=": true, "!=": true, ">": true, "<": true, ">=": true, "<=": true, "LIKE": true, "NOT LIKE": true, "IS NULL": true, "IS NOT NULL": true}
	for _, condition := range advancedSearch {
		column := sanitizeIdent(condition.Column)
		if column == "" {
			continue
		}
		operator := strings.ToUpper(strings.TrimSpace(condition.Operator))
		if !validOps[operator] {
			operator = "="
		}
		if operator == "IS NULL" || operator == "IS NOT NULL" {
			clauses = append(clauses, fmt.Sprintf("%s %s", quoteIdent(dbType, column), operator))
			continue
		}

		value := condition.Value
		if operator == "LIKE" || operator == "NOT LIKE" {
			if !strings.Contains(value, "%") {
				value = "%" + value + "%"
			}
			clauses = append(clauses, buildTextSearchCondition(dbType, column, operator, addParam(value)))
			continue
		}
		clauses = append(clauses, fmt.Sprintf("%s %s %s", quoteIdent(dbType, column), operator, addParam(value)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildTextSearchCondition(dbType model.DatabaseType, column, operator, placeholder string) string {
	quotedColumn := quoteIdent(dbType, column)
	switch dbType {
	case model.DatabaseTypePostgresql:
		if operator == "NOT LIKE" {
			operator = "NOT ILIKE"
		} else {
			operator = "ILIKE"
		}
		return fmt.Sprintf("CAST(%s AS TEXT) %s %s", quotedColumn, operator, placeholder)
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		return fmt.Sprintf("LOWER(CAST(%s AS CHAR)) %s LOWER(%s)", quotedColumn, operator, placeholder)
	default:
		return fmt.Sprintf("LOWER(CAST(%s AS TEXT)) %s LOWER(%s)", quotedColumn, operator, placeholder)
	}
}

type rowScanner interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}

func scanRows(rows rowScanner, columns []string) ([]map[string]interface{}, error) {
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for index := range values {
		valuePtrs[index] = &values[index]
	}
	tableData := make([]map[string]interface{}, 0)
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		entry := make(map[string]interface{}, len(columns))
		for index, column := range columns {
			value := values[index]
			if bytes, ok := value.([]byte); ok {
				entry[column] = string(bytes)
			} else {
				entry[column] = value
			}
		}
		tableData = append(tableData, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tableData, nil
}
