package service

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
)

// 这组是针对真实 PostgreSQL 的往返测试：导出的备份必须能原样导回自己。
// 不配 GOPANEL_PG_TEST_DSN 就跳过，所以 CI 和本地常规 go test 都不受影响。
//
// 想跑的话：
//
//	GOPANEL_PG_TEST_DSN='postgres://user:pass@localhost:5432/db?sslmode=disable' go test ./app/service/ -run PostgresExport -v
func pgTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("GOPANEL_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("未设置 GOPANEL_PG_TEST_DSN，跳过 PostgreSQL 往返测试")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("Ping 失败：%v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// 建一对有外键关系的临时表，模拟「被别的表指着」这个真实场景。
func setupReferencedTable(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`DROP TABLE IF EXISTS export_rt_child`,
		`DROP TABLE IF EXISTS export_rt_parent`,
		`CREATE TABLE export_rt_parent (id integer PRIMARY KEY, name text NOT NULL)`,
		`CREATE TABLE export_rt_child (
			id integer PRIMARY KEY,
			parent_id integer REFERENCES export_rt_parent (id) ON DELETE CASCADE
		)`,
		`INSERT INTO export_rt_parent VALUES (1, 'a'), (2, 'b')`,
		`INSERT INTO export_rt_child VALUES (10, 1), (11, 2), (12, NULL)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("准备数据失败 %q：%v", s, err)
		}
	}
	t.Cleanup(func() {
		db.Exec(`DROP TABLE IF EXISTS export_rt_child`)
		db.Exec(`DROP TABLE IF EXISTS export_rt_parent`)
	})
}

func dumpParentTable(t *testing.T, db *sql.DB) string {
	t.Helper()
	quote := func(name string) string { return quoteIdent(model.DatabaseTypePostgresql, name) }
	req := request.ExportTableReq{
		DatabaseName:     "test",
		TableName:        "export_rt_parent",
		Format:           "sql",
		IncludeDropTable: true,
	}
	rows, err := db.Query(`SELECT id, name FROM export_rt_parent ORDER BY id`)
	if err != nil {
		t.Fatalf("读取数据失败：%v", err)
	}
	defer rows.Close()
	return generateSQLDump(
		db, model.DatabaseTypePostgresql, "export_rt_parent", req,
		[]string{"id", "name"}, rows, quote,
	)
}

// 核心保证：被外键指着的表，导出的备份要能原样导回去。
// 修复前这里会卡在 DROP TABLE 的 2BP01（dependent_objects_still_exist）。
func TestPostgresExportRoundTripsWhenTableIsReferenced(t *testing.T) {
	db := pgTestDB(t)
	setupReferencedTable(t, db)

	dump := dumpParentTable(t, db)

	// 用 execSQLImport 走真实导入路径：它自带事务，失败会整体回滚。
	if _, err := execSQLImport(db, dump); err != nil {
		t.Fatalf("导出的备份无法回灌：%v\n---- dump ----\n%s", err, dump)
	}

	var rowCount int
	if err := db.QueryRow(`SELECT count(*) FROM export_rt_parent`).Scan(&rowCount); err != nil {
		t.Fatal(err)
	}
	if rowCount != 2 {
		t.Fatalf("数据应完整恢复，实际 %d 行", rowCount)
	}

	// 外键必须被加回来：只把数据导回去、约束却没了，比导入失败更危险，
	// 因为没有任何人会发现数据库已经和 schema 不一致了。
	var fkCount int
	if err := db.QueryRow(`
		SELECT count(*) FROM pg_constraint
		WHERE confrelid = 'export_rt_parent'::regclass AND contype = 'f'`).Scan(&fkCount); err != nil {
		t.Fatal(err)
	}
	if fkCount != 1 {
		t.Fatalf("指向本表的外键应恢复为 1 条，实际 %d 条", fkCount)
	}

	// 约束不能只是「存在」，还得真的在拦人。
	if _, err := db.Exec(`INSERT INTO export_rt_child VALUES (99, 12345)`); err == nil {
		db.Exec(`DELETE FROM export_rt_child WHERE id = 99`)
		t.Fatal("恢复后的外键没有生效：写入不存在的 parent_id 竟然成功了")
	}
}

// CASCADE 会连视图一起删，而我们只有能力还原外键，所以不能用它。
// 视图仍然依赖时，DROP TABLE 就该老老实实报错——报错比无声删掉视图安全。
func TestPostgresExportDoesNotSilentlyDropDependentViews(t *testing.T) {
	db := pgTestDB(t)
	setupReferencedTable(t, db)
	if _, err := db.Exec(`CREATE VIEW export_rt_view AS SELECT id FROM export_rt_parent`); err != nil {
		t.Fatalf("建视图失败：%v", err)
	}
	t.Cleanup(func() { db.Exec(`DROP VIEW IF EXISTS export_rt_view`) })

	dump := dumpParentTable(t, db)
	if strings.Contains(strings.ToUpper(dump), "CASCADE") &&
		strings.Contains(strings.ToUpper(dump), "DROP TABLE") {
		for _, line := range strings.Split(dump, "\n") {
			upper := strings.ToUpper(line)
			if strings.HasPrefix(upper, "DROP TABLE") && strings.Contains(upper, "CASCADE") {
				t.Fatalf("DROP TABLE 不该带 CASCADE，会连视图一起删：%s", line)
			}
		}
	}

	// 导入应当失败（视图挡着），且失败后视图还在。
	if _, err := execSQLImport(db, dump); err == nil {
		t.Fatal("有视图依赖时导入应当报错，而不是悄悄把视图删掉")
	}
	var viewCount int
	if err := db.QueryRow(`
		SELECT count(*) FROM pg_views WHERE viewname = 'export_rt_view'`).Scan(&viewCount); err != nil {
		t.Fatal(err)
	}
	if viewCount != 1 {
		t.Fatal("导入失败后视图必须原样保留")
	}
}
