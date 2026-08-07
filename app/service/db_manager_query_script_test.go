package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/glebarez/go-sqlite"
)

func TestExecSQLScriptRunsStatementsOnOneConnection(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	result, err := execSQLScript(context.Background(), database, `
BEGIN;
CREATE TEMP TABLE repair_items (id INTEGER PRIMARY KEY, name TEXT);
INSERT INTO repair_items VALUES (1, 'first');
INSERT INTO repair_items VALUES (2, 'second');
SELECT id, name FROM repair_items ORDER BY id;
COMMIT;
`)
	if err != nil {
		t.Fatal(err)
	}
	if result["type"] != "query" || result["statements"] != 6 {
		t.Fatalf("result = %#v", result)
	}
	rows, ok := result["rows"].([]map[string]interface{})
	if !ok || len(rows) != 2 || rows[1]["name"] != "second" {
		t.Fatalf("rows = %#v", result["rows"])
	}
}

func TestExecSQLScriptRollsBackExplicitTransactionOnFailure(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec(`CREATE TABLE repair_items (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}

	_, err = execSQLScript(context.Background(), database, `
BEGIN;
INSERT INTO repair_items VALUES (1);
INSERT INTO repair_items VALUES (1);
COMMIT;
`)
	if err == nil || !strings.Contains(err.Error(), "statement 3 failed") {
		t.Fatalf("error = %v", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM repair_items`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want rollback", count)
	}
}
