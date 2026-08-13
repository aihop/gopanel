package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
)

func TestFlowRunRebuildReusesLockedCodeSource(t *testing.T) {
	database := flowTestDatabase(t)
	repository := flowTestGitRepository(t, "rebuild-source")
	project, flow := createFlowBaselineFixture(t, database, repository)
	service := NewFlowRunApplication(database)
	service.autoStart = false
	source, err := service.Create(FlowRunCreateInput{FlowID: flow.ID, UseProjectBaseline: true}, project.CreatorID, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Model(source).Updates(map[string]any{"status": flowRunSuccess, "current_stage": "deployed"}).Error; err != nil {
		t.Fatal(err)
	}
	occupied := model.FlowRun{
		FlowID: flow.ID, ProjectID: project.ID, PipelineID: flow.PipelineID, Version: nextFlowPatchVersion(source.Version),
		SourceType: source.SourceType, SourceCommit: source.SourceCommit, CurrentStage: "created", Status: flowRunWaitingDeployment,
		RequestedBy: project.CreatorID,
	}
	if err := database.Create(&occupied).Error; err != nil {
		t.Fatal(err)
	}

	rebuilt, err := service.Rebuild(source.ID, project.CreatorID, false)
	if err != nil {
		t.Fatal(err)
	}
	if rebuilt.ID == source.ID || rebuilt.Version != nextFlowPatchVersion(occupied.Version) {
		t.Fatalf("unexpected rebuilt identity: source=%+v rebuilt=%+v", source, rebuilt)
	}
	if rebuilt.Status != flowRunQueued || rebuilt.CurrentStage != "created" || rebuilt.PipelineRecordID != 0 || rebuilt.ReleaseID != 0 {
		t.Fatalf("rebuilt run did not start cleanly: %+v", rebuilt)
	}
	if rebuilt.SourceCommit != source.SourceCommit || rebuilt.SourceDigest != source.SourceDigest || rebuilt.SourceManifest != source.SourceManifest {
		t.Fatalf("rebuilt run changed locked source: source=%+v rebuilt=%+v", source, rebuilt)
	}
	var manifest flowSourceManifest
	if err := json.Unmarshal([]byte(rebuilt.SourceManifest), &manifest); err != nil {
		t.Fatal(err)
	}
	ref := fmt.Sprintf("refs/gopanel/flows/%d/repositories/1", rebuilt.ID)
	if commit := gitInRepo(t, repository, "rev-parse", ref); commit != manifest.Repositories[0].Commit {
		t.Fatalf("rebuilt source ref = %s, want %s", commit, manifest.Repositories[0].Commit)
	}
	stored, err := service.Get(rebuilt.ID, project.CreatorID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Stages) != 1 || !strings.Contains(stored.Stages[0].Summary, fmt.Sprintf("#%d", source.ID)) {
		t.Fatalf("rebuild audit stage = %+v", stored.Stages)
	}
}

func TestFlowRunRebuildRejectsNonSuccessfulRun(t *testing.T) {
	service, run, _ := createFlowResumeFixture(t)
	if _, err := service.Rebuild(run.ID, 7, false); !isBusinessError(err, constant.ErrFlowRunRebuildUnsupported) {
		t.Fatalf("queued run should not rebuild: %v", err)
	}
	if _, err := service.Rebuild(run.ID, 9, false); !isBusinessError(err, constant.ErrFlowForbidden) {
		t.Fatalf("rebuild should enforce owner access: %v", err)
	}
}
