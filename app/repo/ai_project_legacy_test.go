package repo

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openAIProjectLegacyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Table(legacyAIProjectTable).AutoMigrate(&model.AIProject{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestDropLegacyAIProjectTableDoesNotMigrateData(t *testing.T) {
	db := openAIProjectLegacyTestDB(t)
	if err := db.Table(legacyAIProjectTable).Create(&model.AIProject{ID: 7, Name: "legacy", WorkDir: "/legacy", CreatorID: 3}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIProject{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AIProject{ID: 9, Name: "current", WorkDir: "/current", CreatorID: 3}).Error; err != nil {
		t.Fatal(err)
	}

	if err := DropLegacyAIProjectTable(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable(legacyAIProjectTable) {
		t.Fatal("legacy project table should be removed")
	}
	var project model.AIProject
	if err := db.First(&project, 9).Error; err != nil {
		t.Fatal(err)
	}
	if project.Name != "current" {
		t.Fatalf("unexpected current project: %#v", project)
	}
	var count int64
	if err := db.Model(&model.AIProject{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("legacy data should not be migrated: count=%d", count)
	}
}

func TestDropLegacyAIProjectTableIsIdempotent(t *testing.T) {
	db := openAIProjectLegacyTestDB(t)
	if err := DropLegacyAIProjectTable(db); err != nil {
		t.Fatal(err)
	}
	if err := DropLegacyAIProjectTable(db); err != nil {
		t.Fatalf("repeated cleanup should succeed: %v", err)
	}
}
