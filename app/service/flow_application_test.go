package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func flowTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/flow.db"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&model.AIProject{}, &model.AITask{}, &model.AICodeDeliveryJob{}, &model.AICodeDelivery{},
		&model.AIDevSessionRepository{}, &model.Pipeline{}, &model.Website{},
	); err != nil {
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

func TestFlowRunLocksCompletedCodeDeliveryManifest(t *testing.T) {
	database := flowTestDatabase(t)
	first := flowTestGitRepository(t, "api")
	second := flowTestGitRepository(t, "web")
	project := model.AIProject{Name: "Multi", CreatorID: 7, SourceDirs: []string{first, second}, PrimaryRepository: first}
	if err := database.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	pipeline := model.Pipeline{Name: "Build", SourceType: "code", CodeProjectID: project.ID, Version: "1.0.0", BuildImage: "host"}
	website := model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&pipeline, &website} {
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
	task := model.AITask{UserID: 7, ProjectID: project.ID, Title: "multi repository change", WorkDir: first}
	if err := database.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	firstCommit := gitInRepo(t, first, "rev-parse", "HEAD")
	secondCommit := gitInRepo(t, second, "rev-parse", "HEAD")
	repositories, _ := json.Marshal([]flowStoredDeliveryRepository{
		{RepositoryName: "api", RepositoryPath: first, Status: "completed", TargetBranch: "main", Commit: firstCommit},
		{RepositoryName: "web", RepositoryPath: second, Status: "completed", TargetBranch: "main", Commit: secondCommit},
	})
	now := time.Now()
	job := model.AICodeDeliveryJob{
		SessionID: 91, TaskID: task.ID, ProjectID: project.ID, UserID: 7, Status: "completed",
		Stage: "completed", Progress: 100, RepositoryResults: string(repositories), CompletedAt: &now,
	}
	if err := database.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	service := NewFlowRunApplication(database)
	service.autoStart = false
	run, err := service.Create(FlowRunCreateInput{FlowID: flow.ID, CodeDeliveryJobID: job.ID}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if run.SourceType != "code_delivery" || run.CodeDeliveryJobID != job.ID || run.TaskID != task.ID || run.SourceDigest == "" {
		t.Fatalf("flow run source was not locked: %+v", run)
	}
	var manifest flowSourceManifest
	if err := json.Unmarshal([]byte(run.SourceManifest), &manifest); err != nil || len(manifest.Repositories) != 2 {
		t.Fatalf("source manifest = %+v, %v", manifest, err)
	}
	if err := database.Model(&job).Updates(map[string]any{"repository_results": "[]", "status": "failed"}).Error; err != nil {
		t.Fatal(err)
	}
	stored, err := service.Get(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if stored.SourceTaskTitle != task.Title || len(stored.SourceRepositories) != 2 || stored.SourceDigest != run.SourceDigest {
		t.Fatalf("locked source changed with delivery job: %+v", stored)
	}
}

func flowTestGitRepository(t *testing.T, content string) string {
	t.Helper()
	repository := t.TempDir()
	gitInRepo(t, repository, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repository, "source.txt"), []byte(content+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, repository, "add", "source.txt")
	gitInRepo(t, repository, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-q", "-m", "initial")
	return repository
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

func TestFlowUpdateReplacesConfigurationAndPreservesProject(t *testing.T) {
	database := flowTestDatabase(t)
	project := model.AIProject{Name: "Shoply", CreatorID: 7}
	otherProject := model.AIProject{Name: "Other", CreatorID: 7}
	initialPipeline := model.Pipeline{Name: "Initial Build", PipelineKey: "initial-build", BuildImage: "host"}
	replacementPipeline := model.Pipeline{Name: "Replacement Build", PipelineKey: "replacement-build", SourceType: "code", BuildImage: "host"}
	foreignPipeline := model.Pipeline{Name: "Foreign Build", PipelineKey: "foreign-build", SourceType: "code", BuildImage: "host"}
	preview := model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	production := model.Website{Alias: "production", PrimaryDomain: "example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&project, &otherProject, &initialPipeline, &preview, &production} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	replacementPipeline.CodeProjectID = project.ID
	foreignPipeline.CodeProjectID = otherProject.ID
	for _, pipeline := range []*model.Pipeline{&replacementPipeline, &foreignPipeline} {
		if err := database.Create(pipeline).Error; err != nil {
			t.Fatal(err)
		}
	}
	flow, err := NewFlowApplication(database).Create(FlowCreateInput{
		Name: "Initial Delivery", ProjectID: project.ID, PipelineID: initialPipeline.ID,
		Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: preview.ID}},
	}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&model.FlowEnvironment{}).Where("flow_id = ? AND name = ?", flow.ID, "preview").Update("health_check_success_count", 9).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewFlowApplication(database).Update(flow.ID, FlowUpdateInput{
		Name: "Forbidden", PipelineID: initialPipeline.ID,
		Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: preview.ID}},
	}, 9, false); !isBusinessError(err, constant.ErrFlowForbidden) {
		t.Fatalf("expected owner check, got %v", err)
	}
	if _, err := NewFlowApplication(database).Update(flow.ID, FlowUpdateInput{
		Name: "Mismatch", PipelineID: foreignPipeline.ID,
		Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: preview.ID}},
	}, 7, false); !isBusinessError(err, constant.ErrFlowPipelineProjectMismatch) {
		t.Fatalf("expected project mismatch, got %v", err)
	}
	updated, err := NewFlowApplication(database).Update(flow.ID, FlowUpdateInput{
		Name: "Production Delivery", PipelineID: replacementPipeline.ID, AutoStartAfterCodeDelivery: true,
		Environments: []FlowEnvironmentInput{
			{Name: "preview", WebsiteID: production.ID, AutoDeploy: true},
			{Name: "production", WebsiteID: production.ID, ApprovalRequired: true},
		},
	}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ProjectID != project.ID || updated.PipelineID != replacementPipeline.ID || !updated.AutoStartAfterCodeDelivery {
		t.Fatalf("unexpected updated flow: %+v", updated)
	}
	if len(updated.Environments) != 2 || updated.Environments[0].HealthCheckSuccessCount != 9 {
		t.Fatalf("updated response did not return persisted environment policy: %+v", updated.Environments)
	}
	stored, err := repo.NewFlow(database).Get(flow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Name != "Production Delivery" || stored.ProjectID != project.ID || len(stored.Environments) != 2 || stored.Environments[0].Name != "preview" {
		t.Fatalf("configuration was not replaced atomically: %+v", stored)
	}
	if stored.Environments[0].WebsiteID != production.ID || stored.Environments[0].HealthCheckSuccessCount != 9 {
		t.Fatalf("existing environment policy was not preserved: %+v", stored.Environments[0])
	}
}

func TestFlowDeleteRemovesUnusedConfigurationAndProtectsHistory(t *testing.T) {
	database := flowTestDatabase(t)
	project := model.AIProject{Name: "Shoply", CreatorID: 7}
	pipeline := model.Pipeline{Name: "Build", BuildImage: "host"}
	website := model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&project, &pipeline, &website} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	create := func(name string) *model.Flow {
		flow, err := NewFlowApplication(database).Create(FlowCreateInput{
			Name: name, ProjectID: project.ID, PipelineID: pipeline.ID,
			Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: website.ID}},
		}, 7, false)
		if err != nil {
			t.Fatal(err)
		}
		return flow
	}
	unused := create("Unused")
	if err := NewFlowApplication(database).Delete(unused.ID, 9, false); !isBusinessError(err, constant.ErrFlowForbidden) {
		t.Fatalf("expected owner check, got %v", err)
	}
	if err := NewFlowApplication(database).Delete(unused.ID, 7, false); err != nil {
		t.Fatal(err)
	}
	var flowCount, environmentCount int64
	database.Model(&model.Flow{}).Where("id = ?", unused.ID).Count(&flowCount)
	database.Model(&model.FlowEnvironment{}).Where("flow_id = ?", unused.ID).Count(&environmentCount)
	if flowCount != 0 || environmentCount != 0 {
		t.Fatalf("unused configuration remains: flow=%d environments=%d", flowCount, environmentCount)
	}

	protected := create("Protected")
	run := model.FlowRun{
		FlowID: protected.ID, ProjectID: project.ID, PipelineID: pipeline.ID, Version: "1.0.0",
		SourceType: "git", SourceCommit: "0123456789abcdef0123456789abcdef01234567",
		CurrentStage: "created", Status: "queued", RequestedBy: 7,
	}
	if err := database.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewFlowApplication(database).Delete(protected.ID, 7, false); !isBusinessError(err, constant.ErrFlowDeleteHistory) {
		t.Fatalf("expected history protection, got %v", err)
	}
	if err := database.First(&model.Flow{}, protected.ID).Error; err != nil {
		t.Fatalf("protected flow was deleted: %v", err)
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
