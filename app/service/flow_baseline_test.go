package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

func TestFlowRunUsesCommittedProjectBaselineWithoutDelivery(t *testing.T) {
	database := flowTestDatabase(t)
	repository := flowTestGitRepository(t, "baseline-v1")
	project, flow := createFlowBaselineFixture(t, database, repository)
	if err := os.WriteFile(filepath.Join(repository, "uncommitted.txt"), []byte("not released\n"), 0644); err != nil {
		t.Fatal(err)
	}
	service := NewFlowRunApplication(database)
	service.autoStart = false
	baseline, err := service.CodeBaselineSource(flow.ID, project.CreatorID, false)
	if err != nil {
		t.Fatal(err)
	}
	if !baseline.Available || !baseline.HasUncommittedChanges || len(baseline.Repositories) != 1 {
		t.Fatalf("baseline source = %+v", baseline)
	}
	run, err := service.Create(FlowRunCreateInput{FlowID: flow.ID, UseProjectBaseline: true}, project.CreatorID, false)
	if err != nil {
		t.Fatal(err)
	}
	if run.SourceType != "code_baseline" || run.CodeDeliveryJobID != 0 || run.SourceCommit != baseline.Repositories[0].Commit {
		t.Fatalf("baseline run = %+v", run)
	}
	var manifest flowSourceManifest
	if err := json.Unmarshal([]byte(run.SourceManifest), &manifest); err != nil || manifest.SourceType != "code_baseline" {
		t.Fatalf("baseline manifest = %+v, %v", manifest, err)
	}
	workspace := t.TempDir()
	if err := materializeFlowSourceManifest(context.Background(), manifest, workspace); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(workspace, "source.txt"))
	if err != nil || string(content) != "baseline-v1\n" {
		t.Fatalf("baseline source content = %q, %v", content, err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "uncommitted.txt")); !os.IsNotExist(err) {
		t.Fatalf("uncommitted file entered baseline: %v", err)
	}
}

func TestFlowProjectBaselineUnavailableWhenValidDeliveryExists(t *testing.T) {
	database := flowTestDatabase(t)
	repository := flowTestGitRepository(t, "delivered")
	project, flow := createFlowBaselineFixture(t, database, repository)
	commit := gitInRepo(t, repository, "rev-parse", "HEAD")
	repositories, _ := json.Marshal([]flowStoredDeliveryRepository{{
		RepositoryName: "repository", RepositoryPath: repository,
		Status: "completed", TargetBranch: "main", Commit: commit,
	}})
	now := time.Now()
	job := model.AICodeDeliveryJob{
		SessionID: 41, ProjectID: project.ID, UserID: project.CreatorID,
		Status: "completed", RepositoryResults: string(repositories), CompletedAt: &now,
	}
	if err := database.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	service := NewFlowRunApplication(database)
	service.autoStart = false
	baseline, err := service.CodeBaselineSource(flow.ID, project.CreatorID, false)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.Available {
		t.Fatalf("baseline should be unavailable: %+v", baseline)
	}
	if _, err := service.Create(FlowRunCreateInput{FlowID: flow.ID, UseProjectBaseline: true}, project.CreatorID, false); err == nil {
		t.Fatal("baseline run should be rejected when a valid delivery exists")
	}
}

func createFlowBaselineFixture(t *testing.T, database *gorm.DB, repository string) (*model.AIProject, *model.Flow) {
	t.Helper()
	project := &model.AIProject{
		Name: "Baseline", CreatorID: 7, SourceDirs: []string{repository}, PrimaryRepository: repository,
	}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	pipeline := &model.Pipeline{Name: "Build", SourceType: "code", CodeProjectID: project.ID, Version: "1.0.0", BuildImage: "host"}
	website := &model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []any{pipeline, website} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	flow, err := NewFlowApplication(database).Create(FlowCreateInput{
		Name: "Baseline Delivery", ProjectID: project.ID, PipelineID: pipeline.ID,
		Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: website.ID}},
	}, project.CreatorID, false)
	if err != nil {
		t.Fatal(err)
	}
	return project, flow
}
