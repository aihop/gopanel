package repo

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseListReturnsAvailableItemsAndConnectionWarnings(t *testing.T) {
	oldDatabase := global.DB
	oldKey := global.CONF.System.EncryptKey
	t.Cleanup(func() {
		global.DB = oldDatabase
		global.CONF.System.EncryptKey = oldKey
	})
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"

	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "database-list.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = database
	if err := database.AutoMigrate(&model.DatabaseServer{}); err != nil {
		t.Fatal(err)
	}

	repository := NewDatabaseServer()
	available := &model.DatabaseServer{
		Name: "local-sqlite",
		Type: model.DatabaseSQLite,
		Host: "/data/local.db",
	}
	unavailable := &model.DatabaseServer{
		Name:     "offline-mysql",
		Type:     model.DatabaseTypeMysql,
		Host:     "127.0.0.1",
		Port:     1,
		Username: "root",
		Password: "private-password",
	}
	if err := repository.Create(available); err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(unavailable); err != nil {
		t.Fatal(err)
	}

	result, err := NewDatabase().List(&gormx.Contextx{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ServerID != available.ID {
		t.Fatalf("unexpected available databases: %#v", result)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected one connection warning, got %#v", result.Warnings)
	}
	warning := result.Warnings[0]
	if warning.ServerID != unavailable.ID || warning.Code != "connection_failed" {
		t.Fatalf("unexpected warning: %#v", warning)
	}
	if strings.Contains(warning.Message, unavailable.Password) {
		t.Fatalf("warning leaked database password: %q", warning.Message)
	}
}

func TestDatabaseListFiltersBeforeLoadingServers(t *testing.T) {
	oldDatabase := global.DB
	t.Cleanup(func() { global.DB = oldDatabase })

	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "database-filter.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = database
	if err := database.AutoMigrate(&model.DatabaseServer{}); err != nil {
		t.Fatal(err)
	}
	servers := []*model.DatabaseServer{
		{Name: "first", Type: model.DatabaseSQLite, Host: "/data/first.db"},
		{Name: "second", Type: model.DatabaseSQLite, Host: "/data/second.db"},
	}
	for _, server := range servers {
		if err := database.Create(server).Error; err != nil {
			t.Fatal(err)
		}
	}

	result, err := NewDatabase().List(&gormx.Contextx{
		Page:  1,
		Limit: 20,
		Wheres: []*gormx.WhereOne{
			{Field: "server_id", Rule: gormx.WhereRuleEq, Val: strconv.FormatUint(uint64(servers[1].ID), 10)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ServerID != servers[1].ID {
		t.Fatalf("server filter was not applied: %#v", result)
	}
}

func TestDatabaseListWarningRedactsPasswordVariants(t *testing.T) {
	server := &model.DatabaseServer{
		ID:       1,
		Name:     "postgres",
		Type:     model.DatabaseTypePostgresql,
		Password: "secret/value",
	}
	warning := databaseListWarning(server, "connection_failed", errors.New("dsn contains secret/value and secret%2Fvalue"))
	if strings.Contains(warning.Message, "secret/value") || strings.Contains(warning.Message, "secret%2Fvalue") {
		t.Fatalf("warning leaked password: %q", warning.Message)
	}
}
