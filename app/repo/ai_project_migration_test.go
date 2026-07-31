package repo

import (
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type minimalLegacyAIProject struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string `gorm:"type:varchar(255);not null"`
	Description string `gorm:"type:text"`
	CreatorID   uint   `gorm:"type:integer;not null"`
}

func openAIProjectMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMigrateLegacyAIProjectsPreservesDataAndBacksUpTable(t *testing.T) {
	db := openAIProjectMigrationTestDB(t)
	if err := db.Table(legacyAIProjectTable).AutoMigrate(&model.AIProject{}); err != nil {
		t.Fatal(err)
	}
	legacy := &model.AIProject{
		ID: 7, Name: "legacy", Description: "existing project", WorkDir: "/workspace/legacy",
		SourceDirs: []string{"/workspace/api", "/workspace/web"}, CreatorID: 3,
		RequireQualityGate: true, MonthlyTokenBudget: 12000,
	}
	if err := db.Table(legacyAIProjectTable).Create(legacy).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyAIProjects(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable(legacyAIProjectTable) || !db.Migrator().HasTable(legacyAIProjectBackupTable) {
		t.Fatal("legacy project table should be renamed to the backup table")
	}
	var project model.AIProject
	if err := db.First(&project, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if project.Name != legacy.Name || project.WorkDir != legacy.WorkDir || len(project.SourceDirs) != 2 ||
		!project.RequireQualityGate || project.MonthlyTokenBudget != legacy.MonthlyTokenBudget {
		t.Fatalf("unexpected migrated project: %#v", project)
	}
	if err := MigrateLegacyAIProjects(db); err != nil {
		t.Fatalf("repeated migration should succeed: %v", err)
	}
}

func TestMigrateLegacyAIProjectsSupportsMinimalLegacySchema(t *testing.T) {
	db := openAIProjectMigrationTestDB(t)
	if err := db.Table(legacyAIProjectTable).AutoMigrate(&minimalLegacyAIProject{}); err != nil {
		t.Fatal(err)
	}
	legacy := &minimalLegacyAIProject{ID: 4, Name: "old", Description: "minimal", CreatorID: 2}
	if err := db.Table(legacyAIProjectTable).Create(legacy).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyAIProjects(db); err != nil {
		t.Fatal(err)
	}
	var project model.AIProject
	if err := db.First(&project, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if project.Name != legacy.Name || project.CreatorID != legacy.CreatorID || project.WorkDir != "" || len(project.SourceDirs) != 0 {
		t.Fatalf("unexpected migrated minimal project: %#v", project)
	}
}

func TestMigrateLegacyAIProjectsRollsBackConflicts(t *testing.T) {
	db := openAIProjectMigrationTestDB(t)
	if err := db.Table(legacyAIProjectTable).AutoMigrate(&minimalLegacyAIProject{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Table(legacyAIProjectTable).Create(&minimalLegacyAIProject{ID: 9, Name: "legacy", CreatorID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIProject{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AIProject{ID: 9, Name: "current", CreatorID: 1}).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyAIProjects(db); err == nil {
		t.Fatal("expected conflicting project IDs to stop migration")
	}
	if !db.Migrator().HasTable(legacyAIProjectTable) || db.Migrator().HasTable(legacyAIProjectBackupTable) {
		t.Fatal("failed migration should preserve the original legacy table")
	}
}

func TestMigrateLegacyAIProjectsCompletesPartialMigration(t *testing.T) {
	db := openAIProjectMigrationTestDB(t)
	if err := db.Table(legacyAIProjectTable).AutoMigrate(&model.AIProject{}); err != nil {
		t.Fatal(err)
	}
	legacyProjects := []*model.AIProject{
		{ID: 4, Name: "copied", WorkDir: "/copied", CreatorID: 1},
		{ID: 5, Name: "pending", WorkDir: "/pending", CreatorID: 1},
	}
	if err := db.Table(legacyAIProjectTable).Create(&legacyProjects).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIProject{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(legacyProjects[0]).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyAIProjects(db); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.AIProject{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 || !db.Migrator().HasTable(legacyAIProjectBackupTable) {
		t.Fatalf("unexpected partial migration result: count=%d", count)
	}
}

func TestMigrateLegacyAIProjectsRejectsExistingBackup(t *testing.T) {
	db := openAIProjectMigrationTestDB(t)
	if err := db.Table(legacyAIProjectTable).AutoMigrate(&minimalLegacyAIProject{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Table(legacyAIProjectBackupTable).AutoMigrate(&minimalLegacyAIProject{}); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacyAIProjects(db); err == nil {
		t.Fatal("expected an existing backup table to stop migration")
	}
}
