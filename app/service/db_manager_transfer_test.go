package service

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	_ "github.com/glebarez/go-sqlite"
)

func TestImportCSVHandlesQuotedRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, note TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}

	content := "\ufeffid,note\n1,\"hello,\nworld\"\n2,\"say \"\"hi\"\"\"\n"
	imported, err := importCSV(database, model.DatabaseSQLite, "items", content)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 2 {
		t.Fatalf("imported = %d, want 2", imported)
	}
	var note string
	if err := database.QueryRow(`SELECT note FROM items WHERE id = 1`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if note != "hello,\nworld" {
		t.Fatalf("note = %q, want quoted multiline value", note)
	}
}

func TestImportCSVRollsBackOnRowFailure(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT UNIQUE)`); err != nil {
		t.Fatal(err)
	}

	imported, err := importCSV(database, model.DatabaseSQLite, "items", "id,name\n1,duplicate\n2,duplicate\n")
	if err == nil || !strings.Contains(err.Error(), "row 3") {
		t.Fatalf("error = %v, want row 3 failure", err)
	}
	if imported != 1 {
		t.Fatalf("imported before rollback = %d, want 1", imported)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("row count = %d, want rollback to leave 0", count)
	}
}
