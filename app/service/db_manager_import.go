package service

import (
	"database/sql"
	"fmt"
	"strings"
)

// splitSQLStatements 智能分割 SQL 语句，识别引号和注释内的分号不切割
func splitSQLStatements(content string) []string {
	var statements []string
	var cur strings.Builder
	inSingleQ := false      // '
	inDoubleQ := false      // "
	inBacktick := false     // `
	inLineComment := false  // --
	inBlockComment := false // /* */
	dollarQuote := ""       // PostgreSQL $$ or $tag$

	runes := []rune(content)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if dollarQuote != "" {
			closing := []rune(dollarQuote)
			if i+len(closing) <= len(runes) && string(runes[i:i+len(closing)]) == dollarQuote {
				cur.WriteString(dollarQuote)
				i += len(closing) - 1
				dollarQuote = ""
				continue
			}
			cur.WriteRune(ch)
			continue
		}

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
			if ch == '$' && !inSingleQ && !inDoubleQ && !inBacktick {
				if tag, end := postgresDollarQuoteAt(runes, i); tag != "" {
					dollarQuote = tag
					cur.WriteString(tag)
					i = end
					continue
				}
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

func postgresDollarQuoteAt(runes []rune, start int) (string, int) {
	if start >= len(runes) || runes[start] != '$' {
		return "", start
	}
	for end := start + 1; end < len(runes); end++ {
		ch := runes[end]
		if ch == '$' {
			return string(runes[start : end+1]), end
		}
		if end == start+1 {
			if ch != '_' && (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') {
				return "", start
			}
			continue
		}
		if ch != '_' && (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') {
			return "", start
		}
	}
	return "", start
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
