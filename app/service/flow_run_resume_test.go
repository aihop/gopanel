package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
)

func createFlowResumeFixture(t *testing.T) (*FlowRunApplicationService, *model.FlowRun, *model.Pipeline) {
	t.Helper()
	database := flowTestDatabase(t)
	project := model.AIProject{Name: "Resume", CreatorID: 7}
	pipeline := model.Pipeline{Name: "Resume Build", PipelineKey: "resume-build", RepoUrl: "https://example.com/resume.git", Version: "1.0.0", BuildImage: "host"}
	website := model.Website{Alias: "preview", PrimaryDomain: "preview.example.com", Type: "proxy", Status: "Running", Protocol: "HTTP"}
	for _, item := range []interface{}{&project, &pipeline, &website} {
		if err := database.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	flow, err := NewFlowApplication(database).Create(FlowCreateInput{
		Name: "Resume Delivery", ProjectID: project.ID, PipelineID: pipeline.ID,
		Environments: []FlowEnvironmentInput{{Name: "preview", WebsiteID: website.ID}},
	}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	service := NewFlowRunApplication(database)
	service.autoStart = false
	service.pollInterval = time.Millisecond
	run, err := service.Create(FlowRunCreateInput{
		FlowID: flow.ID, SourceCommit: "0123456789abcdef0123456789abcdef01234567",
	}, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	return service, run, &pipeline
}

func TestFlowRunResumeRebuildsAfterBuildFailure(t *testing.T) {
	service, run, pipeline := createFlowResumeFixture(t)
	failedRecord := model.PipelineRecord{
		PipelineID: pipeline.ID, Status: "failed", Version: run.Version, ExpectedCommit: run.SourceCommit,
		SourceType: "flow_run", SourceID: run.ID, IdempotencyKey: "flow:1:build:1",
	}
	if err := service.db.Create(&failedRecord).Error; err != nil {
		t.Fatal(err)
	}
	run.PipelineRecordID = failedRecord.ID
	if err := service.repo.UpdateRun(run.ID, map[string]any{"pipeline_record_id": failedRecord.ID}); err != nil {
		t.Fatal(err)
	}
	service.failStage(run, "building", "pipeline_failed", "compile failed")

	if _, err := service.Resume(run.ID, 9, false); !isBusinessError(err, constant.ErrFlowForbidden) {
		t.Fatalf("expected owner check, got %v", err)
	}
	resumed, err := service.Resume(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != flowRunQueued || resumed.PipelineRecordID != 0 || resumed.Version != run.Version {
		t.Fatalf("build retry did not preserve identity and reset the record: %+v", resumed)
	}
	if _, err := service.Resume(run.ID, 7, false); !isBusinessError(err, constant.ErrFlowRunNotFailed) {
		t.Fatalf("expected duplicate resume rejection, got %v", err)
	}
	service.runPipeline = func(pipelineID uint, version, expectedCommit string, source PipelineRunSource) (uint, error) {
		record := model.PipelineRecord{
			PipelineID: pipelineID, Status: "success", Version: version, ExpectedCommit: expectedCommit,
			CommitHash: expectedCommit, SourceType: source.Type, SourceID: source.ID, IdempotencyKey: source.IdempotencyKey,
		}
		if err := service.db.Create(&record).Error; err != nil {
			return 0, err
		}
		return record.ID, nil
	}
	service.publishRecord = func(recordID uint) (*model.Release, error) {
		release := model.Release{PipelineID: pipeline.ID, PipelineRecordID: recordID, Version: run.Version, CommitHash: run.SourceCommit, SourceType: "archive", ArtifactDigest: "sha256:resume-build", Status: "ready"}
		if err := service.db.Create(&release).Error; err != nil {
			return nil, err
		}
		return &release, nil
	}
	service.Advance(run.ID)
	stored, err := service.Get(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != flowRunWaitingDeployment || stored.PipelineRecordID == failedRecord.ID {
		t.Fatalf("build retry did not complete with a new record: %+v", stored)
	}
	assertFlowStageAttempts(t, stored.Stages, "building", []string{"failed", "success"})
}

func TestFlowRunResumeRetriesOnlyReleasePublication(t *testing.T) {
	service, run, pipeline := createFlowResumeFixture(t)
	service.runPipeline = func(pipelineID uint, version, expectedCommit string, source PipelineRunSource) (uint, error) {
		record := model.PipelineRecord{
			PipelineID: pipelineID, Status: "success", Version: version, ExpectedCommit: expectedCommit,
			CommitHash: expectedCommit, SourceType: source.Type, SourceID: source.ID, IdempotencyKey: source.IdempotencyKey,
		}
		if err := service.db.Create(&record).Error; err != nil {
			return 0, err
		}
		return record.ID, nil
	}
	service.publishRecord = func(uint) (*model.Release, error) {
		return nil, errors.New("artifact registry unavailable")
	}
	service.Advance(run.ID)
	failed, err := service.Get(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != flowRunFailed || failed.PipelineRecordID == 0 {
		t.Fatalf("expected publication failure: %+v", failed)
	}
	buildRecordID := failed.PipelineRecordID
	resumed, err := service.Resume(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.CurrentStage != "publishing" || resumed.PipelineRecordID != buildRecordID {
		t.Fatalf("publication retry should reuse the successful build: %+v", resumed)
	}
	service.publishRecord = func(recordID uint) (*model.Release, error) {
		release := model.Release{PipelineID: pipeline.ID, PipelineRecordID: recordID, Version: run.Version, CommitHash: run.SourceCommit, SourceType: "archive", ArtifactDigest: "sha256:resume-publish", Status: "ready"}
		if err := service.db.Create(&release).Error; err != nil {
			return nil, err
		}
		return &release, nil
	}
	service.Advance(run.ID)
	stored, err := service.Get(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != flowRunWaitingDeployment || stored.PipelineRecordID != buildRecordID || stored.ReleaseID == 0 {
		t.Fatalf("publication retry did not complete: %+v", stored)
	}
	assertFlowStageAttempts(t, stored.Stages, "publishing", []string{"failed", "success"})
	var recordCount int64
	if err := service.db.Model(&model.PipelineRecord{}).Where("source_type = ? AND source_id = ?", "flow_run", run.ID).Count(&recordCount).Error; err != nil {
		t.Fatal(err)
	}
	if recordCount != 1 {
		t.Fatalf("publication retry rebuilt the pipeline: records=%d", recordCount)
	}
}

func TestFlowRunMarksStalePreparingPipelineInterrupted(t *testing.T) {
	service, run, pipeline := createFlowResumeFixture(t)
	record := model.PipelineRecord{
		PipelineID: pipeline.ID, Status: "preparing", Version: run.Version, ExpectedCommit: run.SourceCommit,
		SourceType: "flow_run", SourceID: run.ID, IdempotencyKey: "flow:stale:build:1",
	}
	if err := service.db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	staleAt := time.Now().Add(-3 * time.Minute)
	if err := service.db.Model(&record).Updates(map[string]any{"created_at": staleAt, "updated_at": staleAt}).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.repo.UpdateRun(run.ID, map[string]any{"pipeline_record_id": record.ID}); err != nil {
		t.Fatal(err)
	}
	GetPipelineLogger(record.ID)
	defer RemovePipelineLogger(record.ID)

	service.Advance(run.ID)
	stored, err := service.Get(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != flowRunFailed || stored.FailureCode != "pipeline_interrupted" {
		t.Fatalf("stale preparing pipeline was not closed: %+v", stored)
	}
	var updated model.PipelineRecord
	if err := service.db.First(&updated, record.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Status != "failed" {
		t.Fatalf("stale pipeline status = %q", updated.Status)
	}
}

func TestFlowRunAutoDeploysReadyRunner(t *testing.T) {
	service, run, pipeline := createFlowResumeFixture(t)
	if err := service.db.Model(&model.FlowEnvironment{}).Where("flow_id = ?", run.FlowID).Updates(map[string]any{
		"auto_deploy": true, "approval_required": false,
	}).Error; err != nil {
		t.Fatal(err)
	}
	service.runPipeline = func(pipelineID uint, version, expectedCommit string, source PipelineRunSource) (uint, error) {
		record := model.PipelineRecord{
			PipelineID: pipelineID, Status: "success", Version: version, ExpectedCommit: expectedCommit,
			CommitHash: expectedCommit, SourceType: source.Type, SourceID: source.ID, IdempotencyKey: source.IdempotencyKey,
			RunnerContainerID: "runner-ready", RunnerHostPort: 13000,
		}
		if err := service.db.Create(&record).Error; err != nil {
			return 0, err
		}
		return record.ID, nil
	}
	service.publishRecord = func(recordID uint) (*model.Release, error) {
		release := model.Release{
			PipelineID: pipeline.ID, PipelineRecordID: recordID, Version: run.Version, CommitHash: run.SourceCommit,
			SourceType: "archive", ArtifactDigest: "sha256:auto-deploy", Status: "ready",
		}
		if err := service.db.Create(&release).Error; err != nil {
			return nil, err
		}
		return &release, nil
	}
	deployed := 0
	service.deployRunner = func(_ context.Context, environment model.FlowEnvironment, record *model.PipelineRecord, version string) error {
		deployed++
		if environment.Name != "preview" || record.RunnerHostPort != 13000 || version != run.Version {
			t.Fatalf("unexpected deployment: environment=%+v record=%+v version=%s", environment, record, version)
		}
		return nil
	}

	service.Advance(run.ID)
	stored, err := service.Get(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if deployed != 1 || stored.Status != flowRunSuccess || stored.CurrentStage != "deployed" || stored.ReleaseID == 0 {
		t.Fatalf("automatic deployment did not finish: deployed=%d run=%+v", deployed, stored)
	}
	assertFlowStageAttempts(t, stored.Stages, "deploying", []string{"success"})
}

func TestFlowRunRetriesDeploymentWithoutRebuilding(t *testing.T) {
	service, run, pipeline := createFlowResumeFixture(t)
	if err := service.db.Model(&model.FlowEnvironment{}).Where("flow_id = ?", run.FlowID).Updates(map[string]any{
		"auto_deploy": true, "approval_required": false,
	}).Error; err != nil {
		t.Fatal(err)
	}
	builds := 0
	service.runPipeline = func(pipelineID uint, version, expectedCommit string, source PipelineRunSource) (uint, error) {
		builds++
		record := model.PipelineRecord{
			PipelineID: pipelineID, Status: "success", Version: version, ExpectedCommit: expectedCommit,
			CommitHash: expectedCommit, SourceType: source.Type, SourceID: source.ID, IdempotencyKey: source.IdempotencyKey,
			RunnerContainerID: "runner-retry", RunnerHostPort: 14000,
		}
		if err := service.db.Create(&record).Error; err != nil {
			return 0, err
		}
		return record.ID, nil
	}
	service.publishRecord = func(recordID uint) (*model.Release, error) {
		release := model.Release{
			PipelineID: pipeline.ID, PipelineRecordID: recordID, Version: run.Version, CommitHash: run.SourceCommit,
			SourceType: "archive", ArtifactDigest: "sha256:deploy-retry", Status: "ready",
		}
		if err := service.db.Create(&release).Error; err != nil {
			return nil, err
		}
		return &release, nil
	}
	deployments := 0
	service.deployRunner = func(context.Context, model.FlowEnvironment, *model.PipelineRecord, string) error {
		deployments++
		if deployments == 1 {
			return errors.New("caddy reload failed")
		}
		return nil
	}

	service.Advance(run.ID)
	failed, err := service.Get(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != flowRunFailed || failed.FailureCode != "website_deploy_failed" || failed.ReleaseID == 0 {
		t.Fatalf("deployment failure was not preserved: %+v", failed)
	}
	if _, err := service.Resume(run.ID, 7, false); err != nil {
		t.Fatal(err)
	}
	service.Advance(run.ID)
	stored, err := service.Get(run.ID, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if builds != 1 || deployments != 2 || stored.Status != flowRunSuccess || stored.CurrentStage != "deployed" {
		t.Fatalf("deployment retry rebuilt or did not finish: builds=%d deployments=%d run=%+v", builds, deployments, stored)
	}
	assertFlowStageAttempts(t, stored.Stages, "deploying", []string{"failed", "success"})
}

func assertFlowStageAttempts(t *testing.T, stages []model.FlowStageRun, stage string, statuses []string) {
	t.Helper()
	actual := make([]string, 0, len(statuses))
	for _, item := range stages {
		if item.Stage == stage {
			actual = append(actual, item.Status)
		}
	}
	if len(actual) != len(statuses) {
		t.Fatalf("%s attempts = %v, want %v", stage, actual, statuses)
	}
	for index := range statuses {
		if actual[index] != statuses[index] {
			t.Fatalf("%s attempts = %v, want %v", stage, actual, statuses)
		}
	}
}
