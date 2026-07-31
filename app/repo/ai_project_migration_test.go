package repo

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openAIProjectMigrationTestDB(t *testing.T) *gorm.DB {
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

func TestMigrateLegacyAIProjectsPreservesDataAndReferences(t *testing.T) {
	db := openAIProjectMigrationTestDB(t)
	legacy := &model.AIProject{
		ID: 7, Name: "legacy", Description: "existing project", WorkDir: "/workspace/legacy",
		SourceDirs: []string{"/workspace/api", "/workspace/web"}, CreatorID: 3,
		RequireQualityGate: true, MonthlyTokenBudget: 12000,
	}
	if err := db.Table(legacyAIProjectTable).Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIProject{}, &model.AITask{}); err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{ProjectID: legacy.ID, UserID: 3, Title: "keep reference", WorkDir: legacy.WorkDir}
	if err := db.Create(task).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyAIProjects(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable(legacyAIProjectTable) {
		t.Fatal("legacy project table should be removed after migration")
	}
	var project model.AIProject
	if err := db.First(&project, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if project.Name != legacy.Name || project.Description != legacy.Description || project.WorkDir != legacy.WorkDir ||
		len(project.SourceDirs) != 2 || project.SourceDirs[1] != legacy.SourceDirs[1] || !project.RequireQualityGate ||
		project.MonthlyTokenBudget != legacy.MonthlyTokenBudget {
		t.Fatalf("unexpected migrated project: %#v", project)
	}
	var storedTask model.AITask
	if err := db.First(&storedTask, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedTask.ProjectID != legacy.ID {
		t.Fatalf("project reference changed during migration: %d", storedTask.ProjectID)
	}
	if err := MigrateLegacyAIProjects(db); err != nil {
		t.Fatalf("migration should be idempotent: %v", err)
	}
}

func TestMigrateLegacyAIProjectsRollsBackConflicts(t *testing.T) {
	db := openAIProjectMigrationTestDB(t)
	if err := db.Table(legacyAIProjectTable).Create(&model.AIProject{ID: 9, Name: "legacy", WorkDir: "/legacy", CreatorID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIProject{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AIProject{ID: 9, Name: "current", WorkDir: "/current", CreatorID: 1}).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyAIProjects(db); err == nil {
		t.Fatal("expected conflicting project IDs to stop migration")
	}
	if !db.Migrator().HasTable(legacyAIProjectTable) {
		t.Fatal("legacy table should remain after a failed migration")
	}
	var legacyCount, projectCount int64
	if err := db.Table(legacyAIProjectTable).Count(&legacyCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.AIProject{}).Count(&projectCount).Error; err != nil {
		t.Fatal(err)
	}
	if legacyCount != 1 || projectCount != 1 {
		t.Fatalf("migration conflict changed data: legacy=%d projects=%d", legacyCount, projectCount)
	}
}

func TestMigrateLegacyAIProjectsAcceptsMatchingPartialMigration(t *testing.T) {
	db := openAIProjectMigrationTestDB(t)
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
	if count != 2 || db.Migrator().HasTable(legacyAIProjectTable) {
		t.Fatalf("unexpected partial migration result: count=%d legacy=%v", count, db.Migrator().HasTable(legacyAIProjectTable))
	}
}
