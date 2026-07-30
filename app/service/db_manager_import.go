package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

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

// splitSQLStatements 智能分割 SQL 语句，识别引号和注释内的分号不切割
func splitSQLStatements(content string) []string {
	var statements []string
	var cur strings.Builder
	inSingleQ := false      // '
	inDoubleQ := false      // "
	inBacktick := false     // `
	inLineComment := false  // --
	inBlockComment := false // /* */

	runes := []rune(content)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		// 转义字符（MySQL 用 '' 转义单引号）
		if inSingleQ && ch == '\'' && i+1 < len(runes) && runes[i+1] == '\'' {
			cur.WriteString("''")
			i++
			continue
		}
		if inDoubleQ && ch == '"' && i+1 < len(runes) && runes[i+1] == '"' {
			cur.WriteString(`""`)
			i++
			continue
		}

		// 引号切换
		if !inLineComment && !inBlockComment {
			if ch == '\'' && !inDoubleQ && !inBacktick {
				inSingleQ = !inSingleQ
				cur.WriteRune(ch)
				continue
			}
			if ch == '"' && !inSingleQ && !inBacktick {
				inDoubleQ = !inDoubleQ
				cur.WriteRune(ch)
				continue
			}
			if ch == '`' && !inSingleQ && !inDoubleQ {
				inBacktick = !inBacktick
				cur.WriteRune(ch)
				continue
			}
		}

		// 行注释 -- (需要后面跟空格或控制字符)
		if !inSingleQ && !inDoubleQ && !inBacktick && !inBlockComment && !inLineComment &&
			ch == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			if i+2 >= len(runes) || runes[i+2] == ' ' || runes[i+2] == '\t' || runes[i+2] == '\n' || runes[i+2] == '\r' {
				inLineComment = true
				cur.WriteString("--")
				i++
				continue
			}
		}
		if inLineComment && (ch == '\n' || ch == '\r') {
			inLineComment = false
			cur.WriteRune(ch)
			continue
		}

		// 块注释 /* */
		if !inSingleQ && !inDoubleQ && !inBacktick && !inLineComment && !inBlockComment &&
			ch == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			inBlockComment = true
			cur.WriteString("/*")
			i++
			continue
		}
		if inBlockComment && ch == '*' && i+1 < len(runes) && runes[i+1] == '/' {
			inBlockComment = false
			cur.WriteString("*/")
			i++
			continue
		}

		// 分号切分（仅在正常状态下）
		if !inSingleQ && !inDoubleQ && !inBacktick && !inLineComment && !inBlockComment && ch == ';' {
			stmt := strings.TrimSpace(cur.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			cur.Reset()
			continue
		}

		cur.WriteRune(ch)
	}

	// 收尾
	stmt := strings.TrimSpace(cur.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

func execSQLImport(db *sql.DB, content string) (int, error) {
	statements := splitSQLStatements(content)
	// 包在事务里：任一语句失败整体回滚，避免留下半成品（已 DROP/CREATE 的表）。
	// 注意 MySQL 的 DDL 会隐式提交、无法回滚，属其固有行为；PG/SQLite 可完整回滚。
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	executed := 0
	for i, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			return executed, fmt.Errorf("statement %d failed: %v", i+1, err)
		}
		executed++
	}
	if err := tx.Commit(); err != nil {
		return executed, fmt.Errorf("commit failed: %v", err)
	}
	return executed, nil
}
