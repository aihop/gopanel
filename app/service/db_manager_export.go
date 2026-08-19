package service

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
)

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

// postgresInboundForeignKey 是一条「别的表指向本表」的外键。
type postgresInboundForeignKey struct {
	owner string // 引用方表名，取自 regclass，PG 已按需加好引号
	name  string
	def   string
}

// postgresInboundForeignKeys 查出所有指向 tableName 的外键。
//
// 这类约束不属于本表——getCreateTableSQL 里那段只导出本表自己拥有的外键（conrelid），
// 抓不到反向指过来的（confrelid）。但它们正是 DROP TABLE 报 2BP01 的原因：
// 不先解开，导出的备份就永远回灌不了自己。
//
// 只取顶层表：分区表的子分区会自动派生同名约束，单独去 DROP/ADD 它们会报错，
// 对父表操作时 PG 自己会处理所有分区。
func postgresInboundForeignKeys(db *sql.DB, dbType model.DatabaseType, regclass string) []postgresInboundForeignKey {
	if dbType != model.DatabaseTypePostgresql {
		return nil
	}
	rows, err := db.Query(fmt.Sprintf(`
		SELECT c.conrelid::regclass::text, c.conname, pg_catalog.pg_get_constraintdef(c.oid)
		FROM pg_catalog.pg_constraint c
		JOIN pg_catalog.pg_class cl ON cl.oid = c.conrelid
		WHERE c.confrelid = '%s'::regclass AND c.contype = 'f' AND NOT cl.relispartition
		ORDER BY c.conname`, regclass))
	if err != nil {
		return nil
	}
	defer rows.Close()

	var keys []postgresInboundForeignKey
	for rows.Next() {
		var fk postgresInboundForeignKey
		if err := rows.Scan(&fk.owner, &fk.name, &fk.def); err != nil {
			continue
		}
		if strings.TrimSpace(fk.def) == "" {
			continue
		}
		keys = append(keys, fk)
	}
	return keys
}

// appendPostgresInboundForeignKeys 在数据插入完成后把上面解开的外键原样加回来。
//
// 用 ALTER TABLE IF EXISTS：往空库恢复单表时引用方还不存在，静默跳过即可——
// 那条外键属于引用方，等它自己被恢复时会随建表语句一起带回来。
//
// 这里刻意不加 BEGIN/COMMIT：execSQLImport 已经把整个文件包在一个事务里跑，
// 再写一个 COMMIT 反而会把外层事务提前提交，毁掉导入器自己的回滚保证。
func appendPostgresInboundForeignKeys(b *strings.Builder, keys []postgresInboundForeignKey, quote func(string) string) {
	if len(keys) == 0 {
		return
	}
	b.WriteString("\n-- 恢复指向本表的外键（开头已先解开，否则 DROP TABLE 会被它们挡住）\n")
	for _, fk := range keys {
		b.WriteString(fmt.Sprintf(
			"ALTER TABLE IF EXISTS %s ADD CONSTRAINT %s %s;\n", fk.owner, quote(fk.name), fk.def,
		))
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

	// 指向本表的外键：先解开，末尾再原样加回。查一次，两处复用。
	// 刻意不用 DROP TABLE ... CASCADE——CASCADE 连视图等依赖对象一起删，
	// 而我们只有能力还原外键。真有别的依赖，就让 DROP TABLE 照常报错，
	// 那比无声无息删掉一个视图安全得多。
	var inboundFKs []postgresInboundForeignKey
	if req.IncludeDropTable {
		inboundFKs = postgresInboundForeignKeys(db, dbType, quote(tableName))
	}

	// DROP TABLE
	if req.IncludeDropTable {
		for _, fk := range inboundFKs {
			b.WriteString(fmt.Sprintf(
				"ALTER TABLE IF EXISTS %s DROP CONSTRAINT IF EXISTS %s;\n", fk.owner, quote(fk.name),
			))
		}
		if len(inboundFKs) > 0 {
			b.WriteString("\n")
		}
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
		appendPostgresInboundForeignKeys(&b, inboundFKs, quote)
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
	appendPostgresInboundForeignKeys(&b, inboundFKs, quote)
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
