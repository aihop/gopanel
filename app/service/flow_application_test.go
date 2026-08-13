package service

import (
	"testing"
	"time"

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
	if err := repo.NewPipelineRecord(database).MigrateTable(); err != nil {
		t.Fatal(err)
	}
	if err := repo.NewRelease(database).MigrateTable(); err != nil {
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

func TestFlowCreateRejectsCodePipelineFromAnotherProject(t *testing.T) {
	database := flowTestDatabase(t)
	project := model.AIProject{Name: "Selected", CreatorID: 7}
	otherProject := model.AIProject{Name: "Other", CreatorID: 7}
	website := model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&project, &otherProject, &website} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	pipeline := model.Pipeline{Name: "Other Build", SourceType: "code", CodeProjectID: otherProject.ID, BuildImage: "host"}
	if err := database.Create(&pipeline).Error; err != nil {
		t.Fatal(err)
	}
	_, err := NewFlowApplication(database).Create(FlowCreateInput{
		Name: "Delivery", ProjectID: project.ID, PipelineID: pipeline.ID,
		Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: website.ID}},
	}, 7, false)
	if err == nil {
		t.Fatal("Code pipeline bound to another project should be rejected")
	}
}

func TestFlowRunLocksVersionCommitAndRelease(t *testing.T) {
	database := flowTestDatabase(t)
	project := model.AIProject{Name: "Shoply", CreatorID: 7}
	pipeline := model.Pipeline{Name: "Build", RepoUrl: "https://example.com/shoply.git", Branch: "main", Version: "1.4.2", BuildImage: "host"}
	website := model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&project, &pipeline, &website} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	flow, err := NewFlowApplication(database).Create(FlowCreateInput{
		Name: "Delivery", ProjectID: project.ID, PipelineID: pipeline.ID,
		Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: website.ID}},
	}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	commit := "0123456789abcdef0123456789abcdef01234567"
	service := NewFlowRunApplication(database)
	service.autoStart = false
	run, err := service.Create(FlowRunCreateInput{FlowID: flow.ID, SourceCommit: commit}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if run.Version != "1.4.3" || run.SourceCommit != commit || run.Status != flowRunQueued {
		t.Fatalf("flow run did not lock identity: %+v", run)
	}
	service.runPipeline = func(pipelineID uint, version, expectedCommit string, source PipelineRunSource) (uint, error) {
		record := &model.PipelineRecord{
			PipelineID: pipelineID, Status: "success", Version: version, ExpectedCommit: expectedCommit,
			CommitHash: expectedCommit, SourceType: source.Type, SourceID: source.ID, IdempotencyKey: source.IdempotencyKey,
		}
		if err := database.Create(record).Error; err != nil {
			return 0, err
		}
		return record.ID, nil
	}
	service.publishRecord = func(recordID uint) (*model.Release, error) {
		release := &model.Release{
			PipelineID: pipeline.ID, PipelineRecordID: recordID, Version: run.Version,
			CommitHash: commit, SourceType: "archive", ArtifactDigest: "sha256:flow-artifact", Status: "ready",
		}
		if err := database.Create(release).Error; err != nil {
			return nil, err
		}
		return release, nil
	}
	service.pollInterval = time.Millisecond
	service.Advance(run.ID)
	stored, err := service.Get(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != flowRunWaitingDeployment || stored.CurrentStage != "waiting_deployment" || stored.ReleaseID == 0 {
		t.Fatalf("flow run did not reach release boundary: %+v", stored)
	}
	if stored.ArtifactDigest != "sha256:flow-artifact" || len(stored.Stages) != 5 {
		t.Fatalf("flow run summary or stages are incomplete: %+v", stored)
	}
	var record model.PipelineRecord
	if err := database.First(&record, stored.PipelineRecordID).Error; err != nil {
		t.Fatal(err)
	}
	if record.SourceType != "flow_run" || record.SourceID != run.ID || record.Version != run.Version || record.ExpectedCommit != commit {
		t.Fatalf("pipeline provenance mismatch: %+v", record)
	}
	if err := NewPipelineApplication(database).DeleteRecord(record.ID); err == nil {
		t.Fatal("pipeline records owned by flow runs should be protected")
	}
}

func TestFlowRunRejectsDuplicateExplicitVersion(t *testing.T) {
	database := flowTestDatabase(t)
	project := model.AIProject{Name: "Shoply", CreatorID: 7}
	pipeline := model.Pipeline{Name: "Build", RepoUrl: "https://example.com/shoply.git", Version: "1.0.0", BuildImage: "host"}
	website := model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&project, &pipeline, &website} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	flow, err := NewFlowApplication(database).Create(FlowCreateInput{
		Name: "Delivery", ProjectID: project.ID, PipelineID: pipeline.ID,
		Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: website.ID}},
	}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	service := NewFlowRunApplication(database)
	service.autoStart = false
	input := FlowRunCreateInput{FlowID: flow.ID, SourceCommit: "0123456789abcdef0123456789abcdef01234567", Version: "2.0.0"}
	if _, err := service.Create(input, 7, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(input, 7, false); err == nil {
		t.Fatal("duplicate explicit flow version should be rejected")
	}
}

func TestFlowRunRequiresPipelineRepository(t *testing.T) {
	database := flowTestDatabase(t)
	project := model.AIProject{Name: "Shoply", CreatorID: 7}
	pipeline := model.Pipeline{Name: "Script Only", Version: "1.0.0", BuildImage: "host"}
	website := model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&project, &pipeline, &website} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	flow, err := NewFlowApplication(database).Create(FlowCreateInput{
		Name: "Delivery", ProjectID: project.ID, PipelineID: pipeline.ID,
		Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: website.ID}},
	}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	service := NewFlowRunApplication(database)
	service.autoStart = false
	if _, err := service.Create(FlowRunCreateInput{
		FlowID: flow.ID, SourceCommit: "0123456789abcdef0123456789abcdef01234567",
	}, 7, false); err == nil {
		t.Fatal("flow run should reject a pipeline without repository")
	}
}
