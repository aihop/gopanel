package repo

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSettingUpdateOrCreatePreservesEmptyValue(t *testing.T) {
	oldDatabase := global.DB
	t.Cleanup(func() { global.DB = oldDatabase })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	repository := NewSetting(database)
	if err := repository.UpdateOrCreate("clearable", "configured"); err != nil {
		t.Fatal(err)
	}
	if err := repository.UpdateOrCreate("clearable", ""); err != nil {
		t.Fatal(err)
	}
	setting, err := repository.Get(repository.WithByKey("clearable"))
	if err != nil {
		t.Fatal(err)
	}
	if setting.Value != "" {
		t.Fatalf("setting value = %q, want empty", setting.Value)
	}
}
