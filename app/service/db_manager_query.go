package service

import (
	"context"
	"database/sql"
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

	return execSQLScript(context.Background(), db, req.SQL)
}

func execSQLScript(ctx context.Context, db *sql.DB, content string) (map[string]interface{}, error) {
	statements := splitSQLStatements(content)
	if len(statements) == 0 {
		return nil, fmt.Errorf("SQL statement is empty")
	}
	connection, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	result := map[string]interface{}{"type": "exec", "affected": int64(0), "statements": len(statements)}
	var affectedTotal int64
	for index, statement := range statements {
		statementResult, execErr := execSQLStatement(ctx, connection, statement)
		if execErr != nil {
			_, _ = connection.ExecContext(ctx, "ROLLBACK")
			return nil, fmt.Errorf("statement %d failed: %w", index+1, execErr)
		}
		if statementResult["type"] == "query" {
			result = statementResult
			result["statements"] = len(statements)
			continue
		}
		if affected, ok := statementResult["affected"].(int64); ok {
			affectedTotal += affected
		}
		if result["type"] != "query" {
			result["affected"] = affectedTotal
		}
	}
	return result, nil
}

func execSQLStatement(ctx context.Context, connection *sql.Conn, statement string) (map[string]interface{}, error) {
	if !isSQLQueryStatement(statement) {
		result, err := connection.ExecContext(ctx, statement)
		if err != nil {
			return nil, err
		}
		affected, _ := result.RowsAffected()
		return map[string]interface{}{"type": "exec", "affected": affected}, nil
	}
	rows, err := connection.QueryContext(ctx, statement)
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

func isSQLQueryStatement(statement string) bool {
	sqlUpper := strings.ToUpper(strings.TrimSpace(statement))
	return strings.HasPrefix(sqlUpper, "SELECT") || strings.HasPrefix(sqlUpper, "SHOW") ||
		strings.HasPrefix(sqlUpper, "EXPLAIN") || strings.HasPrefix(sqlUpper, "DESCRIBE") ||
		strings.HasPrefix(sqlUpper, "DESC ") || strings.HasPrefix(sqlUpper, "PRAGMA") ||
		strings.HasPrefix(sqlUpper, "WITH") || strings.HasPrefix(sqlUpper, "VALUES") ||
		strings.Contains(sqlUpper, " RETURNING ")
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
	orderClause := ""
	if req.SortField != "" && req.SortOrder != "" {
		columnRows, queryErr := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 0", quoteTable(server.Type, tableName)))
		if queryErr != nil {
			return nil, queryErr
		}
		columns, columnsErr := columnRows.Columns()
		columnRows.Close()
		if columnsErr != nil {
			return nil, columnsErr
		}
		orderClause = buildTableDataOrderClause(server.Type, req.SortField, req.SortOrder, columns)
	}
	offset := (req.Page - 1) * req.Limit
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", quoteTable(server.Type, tableName), whereClause)
	dataSQL := fmt.Sprintf("SELECT %s FROM %s%s%s LIMIT %d OFFSET %d", selectCols, quoteTable(server.Type, tableName), whereClause, orderClause, req.Limit, offset)

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

func buildTableDataOrderClause(dbType model.DatabaseType, sortField, sortOrder string, columns []string) string {
	field := sanitizeIdent(sortField)
	if field == "" {
		return ""
	}
	found := false
	for _, column := range columns {
		if field == column {
			found = true
			break
		}
	}
	if !found {
		return ""
	}

	direction := ""
	switch strings.ToLower(strings.TrimSpace(sortOrder)) {
	case "asc", "ascend":
		direction = "ASC"
	case "desc", "descend":
		direction = "DESC"
	default:
		return ""
	}
	return fmt.Sprintf(" ORDER BY %s %s", quoteIdent(dbType, field), direction)
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
