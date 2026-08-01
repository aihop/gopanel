package repo

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type legacyWebsitePipelineBinding struct {
	model.Website
	PipelineID uint `gorm:"column:pipeline_id"`
}

type legacyAppDeployPipelineBinding struct {
	model.AppDeploy
	ReleaseID        uint `gorm:"column:release_id"`
	PipelineRecordID uint `gorm:"column:pipeline_record_id"`
}

func TestWebsiteMigrationDropsPipelineBindingColumn(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&legacyWebsitePipelineBinding{}); err != nil {
		t.Fatal(err)
	}
	if !database.Migrator().HasColumn(&model.Website{}, "pipeline_id") {
		t.Fatal("expected legacy pipeline_id column")
	}
	legacy := legacyWebsitePipelineBinding{
		Website:    model.Website{PrimaryDomain: "example.com", Type: "proxy", Alias: "example", Status: "Running", Protocol: "HTTP", CodeSource: "pipeline"},
		PipelineID: 3,
	}
	if err := database.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}

	repository := &WebsiteRepo{db: database}
	if err := repository.MigrateTable(); err != nil {
		t.Fatal(err)
	}
	if database.Migrator().HasColumn(&model.Website{}, "pipeline_id") {
		t.Fatal("website migration should drop pipeline_id")
	}
	var website model.Website
	if err := database.First(&website, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if website.CodeSource != "container" {
		t.Fatalf("expected legacy pipeline website source to become container, got %q", website.CodeSource)
	}
	if err := repository.MigrateTable(); err != nil {
		t.Fatalf("repeated migration should succeed: %v", err)
	}
}

func TestAppDeployMigrationRemovesPipelineLinks(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&legacyAppDeployPipelineBinding{}); err != nil {
		t.Fatal(err)
	}
	rows := []legacyAppDeployPipelineBinding{
		{AppDeploy: model.AppDeploy{WebsiteID: 1, Version: "manual", SourceType: "image", Status: "Running"}},
		{AppDeploy: model.AppDeploy{WebsiteID: 1, Version: "pipeline", SourceType: "pipeline_sync", Status: "Running"}, PipelineRecordID: 9},
		{AppDeploy: model.AppDeploy{WebsiteID: 1, Version: "release", SourceType: "release", Status: "Running"}, ReleaseID: 7},
	}
	if err := database.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	repository := NewAppDeploy(database)
	if err := repository.MigrateTable(); err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"release_id", "pipeline_record_id"} {
		if database.Migrator().HasColumn(&model.AppDeploy{}, column) {
			t.Fatalf("app deploy migration should drop %s", column)
		}
	}
	var remaining []model.AppDeploy
	if err := database.Order("id asc").Find(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].SourceType != "image" {
		t.Fatalf("unexpected remaining deploys: %#v", remaining)
	}
	if err := repository.MigrateTable(); err != nil {
		t.Fatalf("repeated migration should succeed: %v", err)
	}
}
