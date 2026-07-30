package db

import (
	"net/url"
	"path/filepath"
	"testing"
)

func TestSQLiteDSNEnablesConcurrentWriteProtection(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "panel data.db")
	dsn, err := url.Parse(sqliteDSN(databasePath))
	if err != nil {
		t.Fatal(err)
	}
	if dsn.Scheme != "file" || filepath.Clean(dsn.Path) != filepath.Clean(databasePath) {
		t.Fatalf("unexpected database path: %s", dsn.String())
	}
	pragmas := dsn.Query()["_pragma"]
	want := map[string]bool{
		"busy_timeout(5000)":  false,
		"journal_mode(WAL)":   false,
		"synchronous(NORMAL)": false,
	}
	for _, pragma := range pragmas {
		if _, exists := want[pragma]; exists {
			want[pragma] = true
		}
	}
	for pragma, found := range want {
		if !found {
			t.Fatalf("missing SQLite pragma %s in %v", pragma, pragmas)
		}
	}
}
