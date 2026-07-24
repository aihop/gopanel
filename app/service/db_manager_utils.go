package service

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
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

func quoteChar(dbType model.DatabaseType) string {
	if dbType == model.DatabaseTypeMysql || dbType == model.DatabaseTypeMariaDB {
		return "`"
	}
	return `"`
}

// extractNextvalSeq 从 PG 列默认值表达式 nextval('<seq>'::regclass) 中提取序列名。
// 提取出的序列名保留了 PG 自身的引用形式（如 public.t_id_seq 或 "Schema"."Seq"），
// 可直接用作 regclass 引用。非 nextval 默认值返回空字符串。
func extractNextvalSeq(expr string) string {
	if !strings.Contains(expr, "nextval(") {
		return ""
	}
	start := strings.Index(expr, "'")
	if start < 0 {
		return ""
	}
	rest := expr[start+1:]
	end := strings.Index(rest, "'")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// postgresSequenceStatements 返回 PG 表的序列语句：
// pre 为建表前的 CREATE SEQUENCE，post 为数据插入后的 setval（按导出时序列当前值）。
func postgresSequenceStatements(db *sql.DB, regclass string) (pre []string, post []string) {
	rows, err := db.Query(fmt.Sprintf(`
		SELECT pg_catalog.pg_get_expr(d.adbin, d.adrelid)
		FROM pg_catalog.pg_attrdef d
		WHERE d.adrelid = '%s'::regclass`, regclass))
	if err != nil {
		return nil, nil
	}
	var seqs []string
	for rows.Next() {
		var expr string
		if err := rows.Scan(&expr); err != nil {
			continue
		}
		if seq := extractNextvalSeq(expr); seq != "" {
			seqs = append(seqs, seq)
		}
	}
	rows.Close()

	for _, seq := range seqs {
		pre = append(pre, fmt.Sprintf("CREATE SEQUENCE IF NOT EXISTS %s;", seq))
		var lastVal int64
		var isCalled bool
		if err := db.QueryRow(fmt.Sprintf("SELECT last_value, is_called FROM %s", seq)).Scan(&lastVal, &isCalled); err == nil {
			post = append(post, fmt.Sprintf("SELECT setval('%s', %d, %t);", seq, lastVal, isCalled))
		}
	}
	return pre, post
}

// appendPostgresSetval 在数据插入之后追加 setval，把序列重置到导出时的当前值，
// 保证 SERIAL/自增列在导入后继续正确自增。非 PG 或无序列时不产生任何输出。
func appendPostgresSetval(b *strings.Builder, db *sql.DB, dbType model.DatabaseType, tableName string, quote func(string) string) {
	if dbType != model.DatabaseTypePostgresql {
		return
	}
	if _, postSeq := postgresSequenceStatements(db, quote(tableName)); len(postSeq) > 0 {
		b.WriteString("\n")
		for _, s := range postSeq {
			b.WriteString(s + "\n")
		}
	}
}

func getCreateTableSQL(db *sql.DB, dbType model.DatabaseType, tableName string, quote func(string) string) string {
	switch dbType {
	case model.DatabaseTypeMysql, model.DatabaseTypeMariaDB:
		row := db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE %s", quote(tableName)))
		var tName, createSQL string
		if err := row.Scan(&tName, &createSQL); err == nil {
			return createSQL + ";\n\n"
		}
	case model.DatabaseSQLite:
		row := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName)
		var createSQL string
		if err := row.Scan(&createSQL); err == nil {
			var b strings.Builder
			b.WriteString(createSQL + ";\n\n")
			// 独立索引：PK/UNIQUE 的自动索引 sql 为 NULL（已随建表语句内联），只导出显式 CREATE INDEX
			if idxRows, err := db.Query("SELECT sql FROM sqlite_master WHERE type='index' AND tbl_name=? AND sql IS NOT NULL", tableName); err == nil {
				for idxRows.Next() {
					var idxSQL string
					if err := idxRows.Scan(&idxSQL); err == nil && strings.TrimSpace(idxSQL) != "" {
						b.WriteString(idxSQL + ";\n")
					}
				}
				idxRows.Close()
				b.WriteString("\n")
			}
			return b.String()
		}
	case model.DatabaseTypePostgresql:
		// PostgreSQL: 从 pg_catalog 重建 CREATE TABLE
		// 注意：默认值必须用 pg_get_expr(adbin, adrelid)，adsrc 列在 PostgreSQL 12 中已被移除
		regclass := quote(tableName)
		rows, err := db.Query(fmt.Sprintf(`
			SELECT a.attname, pg_catalog.format_type(a.atttypid, a.atttypmod), a.attnotnull,
				pg_catalog.pg_get_expr(d.adbin, d.adrelid) AS default_val
			FROM pg_catalog.pg_attribute a
			LEFT JOIN pg_catalog.pg_attrdef d ON a.attrelid = d.adrelid AND a.attnum = d.adnum
			WHERE a.attrelid = '%s'::regclass AND a.attnum > 0 AND NOT a.attisdropped
			ORDER BY a.attnum`, regclass))
		if err != nil {
			return ""
		}
		defer rows.Close()

		var b strings.Builder
		// 先建表引用的序列：保留列上的 nextval 默认值需要序列已存在，
		// 否则导入报 "relation xxx_seq does not exist"。setval 在数据插入后由 generateSQLDump 追加。
		if preSeq, _ := postgresSequenceStatements(db, regclass); len(preSeq) > 0 {
			for _, s := range preSeq {
				b.WriteString(s + "\n")
			}
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", quote(tableName)))
		var cols []string
		for rows.Next() {
			var colName, colType string
			var notNull bool
			var defaultVal *string
			if err := rows.Scan(&colName, &colType, &notNull, &defaultVal); err != nil {
				continue
			}
			def := fmt.Sprintf("  %s %s", quote(colName), colType)
			if notNull {
				def += " NOT NULL"
			}
			// 保留默认值（含 nextval 序列默认值）：序列已在上面 CREATE SEQUENCE，可正常导入
			if defaultVal != nil && *defaultVal != "" {
				def += " DEFAULT " + *defaultVal
			}
			cols = append(cols, def)
		}
		if len(cols) == 0 {
			return ""
		}

		// 主键
		pkRows, err := db.Query(fmt.Sprintf(`
			SELECT a.attname
			FROM pg_catalog.pg_index i
			JOIN pg_catalog.pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
			WHERE i.indrelid = '%s'::regclass AND i.indisprimary
			ORDER BY array_position(i.indkey::int2[], a.attnum)`, regclass))
		if err == nil {
			var pkCols []string
			for pkRows.Next() {
				var colName string
				if err := pkRows.Scan(&colName); err == nil {
					pkCols = append(pkCols, quote(colName))
				}
			}
			pkRows.Close()
			if len(pkCols) > 0 {
				cols = append(cols, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
			}
		}

		b.WriteString(strings.Join(cols, ",\n"))
		b.WriteString("\n);\n\n")

		// 索引：主键索引已随建表语句内联，这里导出其余全部索引（含唯一索引，即唯一约束的实现）。
		// 用 pg_get_indexdef 直接拿到官方 DDL 文本，避免手工拼装出错。
		if idxRows, err := db.Query(fmt.Sprintf(`
			SELECT pg_catalog.pg_get_indexdef(i.indexrelid)
			FROM pg_catalog.pg_index i
			WHERE i.indrelid = '%s'::regclass AND NOT i.indisprimary
			ORDER BY i.indexrelid`, regclass)); err == nil {
			for idxRows.Next() {
				var def string
				if err := idxRows.Scan(&def); err == nil && strings.TrimSpace(def) != "" {
					b.WriteString(def)
					b.WriteString(";\n")
				}
			}
			idxRows.Close()
		}

		// 约束：外键(f)、CHECK(c)。主键(p)已内联，唯一约束(u)已由上面的唯一索引覆盖，均跳过。
		if conRows, err := db.Query(fmt.Sprintf(`
			SELECT conname, pg_catalog.pg_get_constraintdef(oid)
			FROM pg_catalog.pg_constraint
			WHERE conrelid = '%s'::regclass AND contype IN ('f','c')
			ORDER BY conname`, regclass)); err == nil {
			for conRows.Next() {
				var name, def string
				if err := conRows.Scan(&name, &def); err == nil && strings.TrimSpace(def) != "" {
					b.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s;\n", quote(tableName), quote(name), def))
				}
			}
			conRows.Close()
		}
		b.WriteString("\n")
		return b.String()
	}
	return ""
}

func generateSQLDump(db *sql.DB, dbType model.DatabaseType, tableName string, req request.ExportTableReq, columns []string, rows *sql.Rows, quote func(string) string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("-- GoPanel Export: %s\n", tableName))
	b.WriteString(fmt.Sprintf("-- Database: %s\n", req.DatabaseName))
	b.WriteString(fmt.Sprintf("-- Date: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// DROP TABLE
	if req.IncludeDropTable {
		b.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n\n", quote(tableName)))
	}

	// CREATE TABLE
	if req.IncludeCreateTable || req.IncludeDropTable {
		// 重新获取 db 连接来查询表结构
		if createSQL := getCreateTableSQL(db, dbType, tableName, quote); createSQL != "" {
			b.WriteString(createSQL)
		}
	}

	// 只有有数据列时才输出 INSERT
	if len(columns) == 0 {
		appendPostgresSetval(&b, db, dbType, tableName, quote)
		return b.String()
	}

	// INSERT header
	b.WriteString(fmt.Sprintf("INSERT INTO %s (", quote(tableName)))
	colParts := make([]string, len(columns))
	for i, col := range columns {
		colParts[i] = quote(col)
	}
	b.WriteString(strings.Join(colParts, ", "))
	b.WriteString(") VALUES\n")

	// 列类型：用于识别二进制列，按十六进制字面量导出
	colTypes, _ := rows.ColumnTypes()

	// Values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var rowValues []string
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		rowParts := make([]string, len(columns))
		for i := range columns {
			binary := i < len(colTypes) && isBinaryColumn(colTypes[i])
			rowParts[i] = formatExportValue(dbType, values[i], binary)
		}
		rowValues = append(rowValues, "("+strings.Join(rowParts, ", ")+")")
	}

	b.WriteString(strings.Join(rowValues, ",\n"))
	b.WriteString(";\n")
	appendPostgresSetval(&b, db, dbType, tableName, quote)
	return b.String()
}

func generateCSV(columns []string, rows *sql.Rows) string {
	var b strings.Builder

	// BOM for Excel to recognize UTF-8
	b.WriteString("\xef\xbb\xbf")

	// Header
	csvCols := make([]string, len(columns))
	for i, col := range columns {
		csvCols[i] = escapeCSVField(col)
	}
	b.WriteString(strings.Join(csvCols, ","))
	b.WriteString("\n")

	// Data rows
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		rowParts := make([]string, len(columns))
		for i := range columns {
			rowParts[i] = formatCSVValue(values[i])
		}
		b.WriteString(strings.Join(rowParts, ","))
		b.WriteString("\n")
	}

	return b.String()
}

func escapeCSVField(field string) string {
	if strings.ContainsAny(field, "\",\n\r") {
		escaped := strings.ReplaceAll(field, `"`, `""`)
		return `"` + escaped + `"`
	}
	return field
}

func formatCSVValue(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return escapeCSVField(v)
	case []byte:
		return escapeCSVField(string(v))
	default:
		return escapeCSVField(fmt.Sprint(val))
	}
}
