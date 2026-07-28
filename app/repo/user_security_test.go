package repo

import (
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/init/conf"
	"github.com/aihop/gopanel/utils/cryptx"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUserMigrationRequiresBootstrapCredentials(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "user.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	old := conf.InitInstall
	conf.InitInstall.User = ""
	conf.InitInstall.Password = ""
	t.Cleanup(func() { conf.InitInstall = old })

	if err := NewUser(db).MigrateTable(); err == nil {
		t.Fatal("expected missing bootstrap credentials to fail")
	}
	if db.Migrator().HasTable(&model.User{}) {
		t.Fatal("user table must not be created before bootstrap credentials are validated")
	}
}

func TestUserMigrationCreatesConfiguredAdministrator(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "user.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	old := conf.InitInstall
	conf.InitInstall.User = "owner@example.com"
	conf.InitInstall.Password = "configured-password"
	t.Cleanup(func() { conf.InitInstall = old })

	if err := NewUser(db).MigrateTable(); err != nil {
		t.Fatal(err)
	}
	var user model.User
	if err := db.First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.Email != conf.InitInstall.User || !cryptx.ValidatePassword(user.Password, conf.InitInstall.Password) {
		t.Fatalf("unexpected bootstrap administrator: %#v", user)
	}
}
