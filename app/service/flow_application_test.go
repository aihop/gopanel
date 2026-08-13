package service

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func flowTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/flow.db"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIProject{}, &model.Pipeline{}, &model.Website{}); err != nil {
		t.Fatal(err)
	}
	if err := repo.NewFlow(database).MigrateTable(); err != nil {
		t.Fatal(err)
	}
	return database
}

func TestFlowCreatePersistsConfigurationAndEnvironments(t *testing.T) {
	database := flowTestDatabase(t)
	project := model.AIProject{Name: "Shoply", CreatorID: 7}
	pipeline := model.Pipeline{Name: "Shoply Build", BuildImage: "host"}
	preview := model.Website{Alias: "shoply-preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	production := model.Website{Alias: "shoply", PrimaryDomain: "example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&project, &pipeline, &preview, &production} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	created, err := NewFlowApplication(database).Create(FlowCreateInput{
		Name: "Shoply Delivery", ProjectID: project.ID, PipelineID: pipeline.ID, AutoStartAfterCodeDelivery: true,
		Environments: []FlowEnvironmentInput{
			{Name: "preview", WebsiteID: preview.ID, AutoDeploy: true, ApprovalRequired: false},
			{Name: "production", WebsiteID: production.ID, ApprovalRequired: true},
		},
	}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || len(created.Environments) != 2 {
		t.Fatalf("flow was not persisted with environments: %+v", created)
	}
	total, items, err := NewFlowApplication(database).Page(7, false, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].ProjectName != project.Name || items[0].PipelineName != pipeline.Name {
		t.Fatalf("unexpected flow summary: total=%d items=%+v", total, items)
	}
	if items[0].Environments[0].WebsiteName != preview.Alias || items[0].Environments[1].WebsiteName != production.Alias {
		t.Fatalf("website names were not resolved: %+v", items[0].Environments)
	}
	if items[0].Environments[0].ApprovalRequired || !items[0].Environments[1].ApprovalRequired {
		t.Fatalf("environment approval policy was not persisted: %+v", items[0].Environments)
	}
}

func TestFlowCreateRejectsForeignAndDuplicateProjects(t *testing.T) {
	database := flowTestDatabase(t)
	project := model.AIProject{Name: "Private", CreatorID: 9}
	pipeline := model.Pipeline{Name: "Build", BuildImage: "host"}
	website := model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&project, &pipeline, &website} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	input := FlowCreateInput{Name: "Delivery", ProjectID: project.ID, PipelineID: pipeline.ID, Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: website.ID}}}
	if _, err := NewFlowApplication(database).Create(input, 7, false); err == nil {
		t.Fatal("foreign project should be rejected")
	}
	if _, err := NewFlowApplication(database).Create(input, 9, false); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFlowApplication(database).Create(input, 9, false); err == nil {
		t.Fatal("duplicate project flow should be rejected")
	}
}
