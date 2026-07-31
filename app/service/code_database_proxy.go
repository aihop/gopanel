package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

const codeDatabaseQueryRowLimit = 200

var codeDatabaseForbiddenWords = map[string]struct{}{
	"alter": {}, "analyze": {}, "attach": {}, "begin": {}, "call": {}, "commit": {},
	"copy": {}, "create": {}, "delete": {}, "detach": {}, "do": {}, "drop": {},
	"execute": {}, "grant": {}, "insert": {}, "lock": {}, "merge": {}, "prepare": {},
	"reindex": {}, "release": {}, "replace": {}, "revoke": {}, "rollback": {}, "savepoint": {},
	"set": {}, "truncate": {}, "update": {}, "vacuum": {},
}

var codeDatabaseReadOnlyPragmas = map[string]bool{
	"collation_list": true, "compile_options": true, "database_list": true,
	"foreign_key_list": true, "function_list": true, "index_info": true,
	"index_list": true, "index_xinfo": true, "module_list": true,
	"pragma_list": true, "table_info": true, "table_list": true, "table_xinfo": true,
}

func ValidateCodeReadOnlySQL(statement string) error {
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return errors.New("SQL 不能为空")
	}
	if strings.ContainsRune(statement, '\x00') || strings.Contains(statement, "--") ||
		strings.Contains(statement, "/*") || strings.Contains(statement, "*/") || strings.Contains(statement, "#") {
		return errors.New("只读代理不允许 SQL 注释")
	}
	statement = strings.TrimSpace(strings.TrimSuffix(statement, ";"))
	if strings.Contains(statement, ";") {
		return errors.New("只读代理每次只允许一条 SQL")
	}
	words := sqlIdentifierWords(statement)
	if len(words) == 0 {
		return errors.New("无法识别 SQL 类型")
	}
	allowedStart := map[string]bool{"select": true, "show": true, "explain": true, "describe": true, "desc": true, "pragma": true, "values": true, "with": true}
	if !allowedStart[words[0]] {
		return errors.New("只读代理仅允许查询语句")
	}
	for _, word := range words {
		if _, forbidden := codeDatabaseForbiddenWords[word]; forbidden {
			return errors.New("SQL 包含只读代理禁止的操作")
		}
	}
	lower := strings.ToLower(statement)
	for _, fragment := range []string{" for update", " for share", " into outfile", " into dumpfile", "pg_sleep(", "sleep(", "benchmark(", "load_file("} {
		if strings.Contains(lower, fragment) {
			return errors.New("SQL 包含只读代理禁止的操作")
		}
	}
	if words[0] == "with" && !containsSQLWord(words, "select") {
		return errors.New("WITH 查询必须包含 SELECT")
	}
	if words[0] == "pragma" && (len(words) < 2 || !codeDatabaseReadOnlyPragmas[words[1]] || strings.Contains(statement, "=")) {
		return errors.New("只读代理不允许该 PRAGMA")
	}
	return nil
}

func sqlIdentifierWords(statement string) []string {
	words := make([]string, 0, 16)
	start := -1
	for index, char := range statement {
		isIdentifier := char == '_' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
		if isIdentifier && start < 0 {
			start = index
		} else if !isIdentifier && start >= 0 {
			words = append(words, strings.ToLower(statement[start:index]))
			start = -1
		}
	}
	if start >= 0 {
		words = append(words, strings.ToLower(statement[start:]))
	}
	return words
}

func containsSQLWord(words []string, expected string) bool {
	for _, word := range words {
		if word == expected {
			return true
		}
	}
	return false
}

func (s *DBManagerService) ExecCodeReadOnlySQL(serverID uint, databaseName, statement string) (map[string]interface{}, error) {
	if err := ValidateCodeReadOnlySQL(statement); err != nil {
		return nil, err
	}
	db, err := s.getDBConn(serverID, databaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0)
	values := make([]interface{}, len(columns))
	pointers := make([]interface{}, len(columns))
	for index := range values {
		pointers[index] = &values[index]
	}
	truncated := false
	for rows.Next() {
		if len(result) >= codeDatabaseQueryRowLimit {
			truncated = true
			break
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		entry := make(map[string]interface{}, len(columns))
		for index, column := range columns {
			if bytes, ok := values[index].([]byte); ok {
				entry[column] = string(bytes)
			} else {
				entry[column] = values[index]
			}
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]interface{}{"type": "query", "columns": columns, "rows": result, "truncated": truncated, "limit": codeDatabaseQueryRowLimit}, nil
}
