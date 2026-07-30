package service

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

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

// isStringColumnType 判断列是否为文本类型。文本列的空字符串是合法值，应原样保留；
// 非文本列（数值/时间/布尔/JSON 等）的空字符串应转为 NULL，否则会报类型错误。
func isStringColumnType(name string) bool {
	name = strings.ToUpper(strings.TrimSpace(name))
	if strings.Contains(name, "CHAR") || strings.Contains(name, "TEXT") {
		return true
	}
	switch name {
	case "STRING", "NAME", "CITEXT", "ENUM", "SET":
		return true
	}
	return false
}

// tableStringColumns 返回表中文本类型列的集合（小写列名）。查询失败时返回空集合，
// 调用方据此退回到「空串一律转 NULL」的旧行为，保证安全。
func tableStringColumns(db *sql.DB, quotedTable string) map[string]bool {
	res := map[string]bool{}
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 0", quotedTable))
	if err != nil {
		return res
	}
	defer rows.Close()
	cts, err := rows.ColumnTypes()
	if err != nil {
		return res
	}
	for _, ct := range cts {
		if isStringColumnType(ct.DatabaseTypeName()) {
			res[strings.ToLower(ct.Name())] = true
		}
	}
	return res
}

// emptyStringToNull 对非文本列把空字符串转为 nil(NULL)，文本列保留空串。
func emptyStringToNull(v interface{}, col string, strCols map[string]bool) interface{} {
	if s, ok := v.(string); ok && s == "" && !strCols[strings.ToLower(col)] {
		return nil
	}
	return v
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
		// 返回空串而非 "1=1"：调用方（UPDATE/DELETE）必须据此拒绝无条件的全表写操作，
		// 避免误删/误更新整张表。
		return "", args
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

func quoteSQLString(dbType model.DatabaseType, val string) string {
	escaped := strings.ReplaceAll(val, "'", "''")
	// 只有 MySQL/MariaDB 默认把反斜杠当转义符；PostgreSQL(standard_conforming_strings)
	// 与 SQLite 中反斜杠是普通字面量，若在这些库上双写反斜杠会把 1 个反斜杠变成 2 个，
	// 造成静默的数据损坏。因此仅对 MySQL 系双写。
	if dbType == model.DatabaseTypeMysql || dbType == model.DatabaseTypeMariaDB {
		escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
	}
	return "'" + escaped + "'"
}

// isBinaryColumn 判断列是否为二进制类型（BYTEA/BLOB 等），用于决定按十六进制字面量导出。
func isBinaryColumn(ct *sql.ColumnType) bool {
	if ct == nil {
		return false
	}
	switch strings.ToUpper(ct.DatabaseTypeName()) {
	case "BYTEA", "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB", "BINARY", "VARBINARY", "BYTES", "IMAGE":
		return true
	}
	return false
}

// encodeBinaryLiteral 将二进制数据编码为对应数据库的十六进制字面量。
func encodeBinaryLiteral(dbType model.DatabaseType, b []byte) string {
	hexStr := hex.EncodeToString(b)
	switch dbType {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		if len(b) == 0 {
			return "''"
		}
		return "0x" + hexStr // MySQL 十六进制字面量，直接作为 BLOB
	case model.DatabaseTypePostgresql:
		return "'\\x" + hexStr + "'" // PG bytea hex 格式，插入 bytea 列时隐式转换
	default: // sqlite
		return "X'" + hexStr + "'" // SQLite blob 字面量
	}
}

func formatExportValue(dbType model.DatabaseType, val interface{}, binary bool) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case int64, int, float64, float32:
		return fmt.Sprint(v)
	case bool:
		// 用 TRUE/FALSE 而非裸 1/0：PG 的 boolean 列不接受整数字面量，
		// MySQL / SQLite 也都识别 TRUE/FALSE
		if v {
			return "TRUE"
		}
		return "FALSE"
	case time.Time:
		// 关键：不能用 fmt.Sprint(time.Time)，它会输出 Go 默认格式
		// "2006-01-02 15:04:05.999999 -0700 MST"，其中的时区名(CST 等)会被
		// PG/MySQL 拒绝。这里输出标准 SQL 时间字面量（不带时区偏移，
		// 对 timestamp / timestamptz / datetime 均可解析）。
		return "'" + v.Format("2006-01-02 15:04:05.999999") + "'"
	case []byte:
		// 二进制列按十六进制字面量导出，避免当作文本转义导致损坏；
		// 非二进制列（MySQL 文本列驱动常返回 []byte）仍按字符串处理。
		if binary {
			return encodeBinaryLiteral(dbType, v)
		}
		return quoteSQLString(dbType, string(v))
	case string:
		return quoteSQLString(dbType, v)
	default:
		return quoteSQLString(dbType, fmt.Sprint(val))
	}
}

func quoteChar(dbType model.DatabaseType) string {
	if dbType == model.DatabaseTypeMysql || dbType == model.DatabaseTypeMariaDB {
		return "`"
	}
	return `"`
}
