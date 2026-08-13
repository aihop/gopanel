package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestPrepareFlowCodeDeliverySourceUsesLockedCommits(t *testing.T) {
	database := flowTestDatabase(t)
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	t.Cleanup(func() { global.CONF.System.BaseDir = oldBaseDir })
	apiRepository := flowTestGitRepository(t, "api-v1")
	webRepository := flowTestGitRepository(t, "web-v1")
	apiCommit := gitInRepo(t, apiRepository, "rev-parse", "HEAD")
	webCommit := gitInRepo(t, webRepository, "rev-parse", "HEAD")
	project := model.AIProject{Name: "Multi", CreatorID: 7, SourceDirs: []string{apiRepository, webRepository}, PrimaryRepository: apiRepository}
	if err := database.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	pipeline := &model.Pipeline{PipelineKey: "flow-source", SourceType: "code", CodeProjectID: project.ID}
	if err := database.Create(pipeline).Error; err != nil {
		t.Fatal(err)
	}
	manifest := flowSourceManifest{
		SchemaVersion: flowSourceManifestSchemaVersion, DeliveryJobID: 8, SessionID: 9, TaskID: 10, TaskTitle: "locked source",
		Repositories: []flowSourceManifestRepository{
			{Name: "api", SourceDir: apiRepository, WorkspacePath: "api", TargetBranch: "main", Commit: apiCommit},
			{Name: "web", SourceDir: webRepository, WorkspacePath: "web", TargetBranch: "main", Commit: webCommit},
		},
	}
	digest, err := flowSourceManifestDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(manifest)
	run := model.FlowRun{
		ProjectID: project.ID, PipelineID: pipeline.ID, Version: "1.2.3", SourceType: "code_delivery",
		SourceCommit: apiCommit, SourceDigest: digest, SourceManifest: string(encoded), CodeDeliveryJobID: 8,
	}
	if err := database.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiRepository, "source.txt"), []byte("api-v2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, apiRepository, "add", "source.txt")
	gitInRepo(t, apiRepository, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-q", "-m", "later")
	record := &model.PipelineRecord{SourceType: "flow_run", SourceID: run.ID, ExpectedCommit: apiCommit}
	workspace := pipelineWorkspaceDir(pipeline)
	logger := GetPipelineLogger(991003)
	defer RemovePipelineLogger(991003)
	commit, actualDigest, err := NewPipelineService(database).prepareCodePipelineSource(
		context.Background(), logger, pipeline, record, workspace,
	)
	if err != nil {
		t.Fatal(err)
	}
	if commit != apiCommit || actualDigest != digest {
		t.Fatalf("source identity = %q, %q", commit, actualDigest)
	}
	content, err := os.ReadFile(filepath.Join(workspace, "api", "source.txt"))
	if err != nil || string(content) != "api-v1\n" {
		t.Fatalf("locked source content = %q, %v", content, err)
	}
	var fact flowBuildFact
	factContent, err := os.ReadFile(filepath.Join(workspace, flowBuildFactFileName))
	if err != nil || json.Unmarshal(factContent, &fact) != nil || fact.Version != "1.2.3" || len(fact.Repositories) != 2 {
		t.Fatalf("flow build fact = %+v, %v", fact, err)
	}
}

func TestFlowRepositoryWorkspacePathMatchesMultiSourceSnapshotNames(t *testing.T) {
	first := filepath.Join(t.TempDir(), "repository")
	second := filepath.Join(t.TempDir(), "repository")
	for _, repository := range []string{first, second} {
		if err := os.MkdirAll(repository, 0755); err != nil {
			t.Fatal(err)
		}
	}
	_, firstWorkspace, err := flowRepositoryWorkspacePath([]string{first, second}, first)
	if err != nil {
		t.Fatal(err)
	}
	_, secondWorkspace, err := flowRepositoryWorkspacePath([]string{first, second}, second)
	if err != nil {
		t.Fatal(err)
	}
	if firstWorkspace != "repository" || secondWorkspace != "repository-2" {
		t.Fatalf("workspace mappings = %q, %q", firstWorkspace, secondWorkspace)
	}
}
