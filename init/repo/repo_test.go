package repo

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInitReturnsMigrationError(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "readonly.db")
	database, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDatabase.Close(); err != nil {
		t.Fatal(err)
	}

	readonlyDatabase, err := gorm.Open(sqlite.Open("file:"+databasePath+"?mode=ro"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = readonlyDatabase
	t.Cleanup(func() { global.DB = previous })

	err = Init()
	if err == nil || !strings.Contains(err.Error(), "migrate user") {
		t.Fatalf("expected contextual migration error, got %v", err)
	}
}
