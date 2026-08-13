package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
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
	_, apiWorkspacePath, err := flowRepositoryWorkspacePath(project.SourceDirs, apiRepository)
	if err != nil {
		t.Fatal(err)
	}
	_, webWorkspacePath, err := flowRepositoryWorkspacePath(project.SourceDirs, webRepository)
	if err != nil {
		t.Fatal(err)
	}
	pipeline := &model.Pipeline{PipelineKey: "flow-source", SourceType: "code", CodeProjectID: project.ID}
	if err := database.Create(pipeline).Error; err != nil {
		t.Fatal(err)
	}
	manifest := flowSourceManifest{
		SchemaVersion: flowSourceManifestSchemaVersion, DeliveryJobID: 8, SessionID: 9, TaskID: 10, TaskTitle: "locked source",
		Repositories: []flowSourceManifestRepository{
			{Name: "api", SourceDir: apiRepository, WorkspacePath: apiWorkspacePath, TargetBranch: "main", Commit: apiCommit},
			{Name: "web", SourceDir: webRepository, WorkspacePath: webWorkspacePath, TargetBranch: "main", Commit: webCommit},
		},
	}
	digest, err := flowSourceManifestDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(manifest)
	if strings.Contains(string(encoded), "sourceDir") {
		t.Fatalf("locked manifest persisted Code path: %s", encoded)
	}
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
	commit, actualDigest, err := NewPipelineService(database).prepareCodePipelineSource(context.Background(), logger, pipeline, record, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if commit != apiCommit || actualDigest != digest {
		t.Fatalf("source identity = %q, %q", commit, actualDigest)
	}
	content, err := os.ReadFile(filepath.Join(workspace, apiWorkspacePath, "source.txt"))
	if err != nil || string(content) != "api-v1\n" {
		t.Fatalf("locked source content = %q, %v", content, err)
	}
	var fact flowBuildFact
	factContent, err := os.ReadFile(filepath.Join(workspace, flowBuildFactFileName))
	if err != nil || json.Unmarshal(factContent, &fact) != nil || fact.Version != "1.2.3" || len(fact.Repositories) != 2 {
		t.Fatalf("flow build fact = %+v, %v", fact, err)
	}
	temporarySources, err := filepath.Glob(filepath.Join(filepath.Dir(workspace), ".code-source-*"))
	if err != nil || len(temporarySources) != 0 {
		t.Fatalf("Code source should materialize directly into workspace: %v, %v", temporarySources, err)
	}
}

func TestExtractFlowGitArchiveMaterializesCommit(t *testing.T) {
	repository := flowTestGitRepository(t, "archive-v1")
	commit := gitInRepo(t, repository, "rev-parse", "HEAD")
	destination := t.TempDir()
	logger := GetPipelineLogger(991004)
	defer RemovePipelineLogger(991004)

	if err := extractFlowGitArchive(context.Background(), logger, "archive", repository, commit, destination); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "source.txt"))
	if err != nil || string(content) != "archive-v1\n" {
		t.Fatalf("archived source content = %q, %v", content, err)
	}
	if !strings.Contains(strings.Join(logger.GetLogs(), "\n"), "Code 仓库归档已生成，正在解包: archive") {
		t.Fatal("archive and extraction boundary was not logged")
	}
}

func TestPipelineRecordRunningIncludesCodePreparation(t *testing.T) {
	for _, status := range []string{"pending", "preparing", "cloning", "building", "deploying"} {
		if !pipelineRecordRunning(status) {
			t.Fatalf("status %q should be running", status)
		}
	}
	for _, status := range []string{"success", "failed", ""} {
		if pipelineRecordRunning(status) {
			t.Fatalf("status %q should not be running", status)
		}
	}
}

func TestPipelineResolvesLegacyFlowSourceThroughCodeProject(t *testing.T) {
	repository := flowTestGitRepository(t, "legacy")
	commit := gitInRepo(t, repository, "rev-parse", "HEAD")
	legacyJSON := fmt.Sprintf(`{"schemaVersion":1,"sourceType":"code_baseline","repositories":[{"name":"repository","sourceDir":"/stale/path","workspacePath":".","targetBranch":"main","commit":%q}]}`, commit)
	var manifest flowSourceManifest
	if err := json.Unmarshal([]byte(legacyJSON), &manifest); err != nil {
		t.Fatal(err)
	}
	project := &model.AIProject{SourceDirs: []string{repository}, PrimaryRepository: repository}
	resolved, err := resolvePipelineCodeSourceManifest(project, manifest)
	if err != nil {
		t.Fatal(err)
	}
	resolvedRepository, err := filepath.EvalSymlinks(repository)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Repositories[0].SourceDir != resolvedRepository {
		t.Fatalf("legacy source resolved to %q", resolved.Repositories[0].SourceDir)
	}
}

func TestPipelineReleasePreparationTiming(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *model.Pipeline
		want     bool
	}{
		{name: "host build", pipeline: &model.Pipeline{BuildImage: "host", BuildScript: "npm run build"}, want: false},
		{name: "container build", pipeline: &model.Pipeline{BuildImage: "node:20", BuildScript: "npm run build"}, want: true},
		{name: "runner", pipeline: &model.Pipeline{BuildImage: "host", BuildScript: "npm run build", RunnerMode: constant.PipelineRunnerModeRunner}, want: true},
		{name: "no build script", pipeline: &model.Pipeline{BuildImage: "host"}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := pipelineShouldPrepareReleaseBeforeBuild(test.pipeline); got != test.want {
				t.Fatalf("prepare release before build = %v, want %v", got, test.want)
			}
		})
	}
}

func TestResetPipelineReleaseSyncMarkerPreservesRelease(t *testing.T) {
	releaseDir := t.TempDir()
	marker := filepath.Join(releaseDir, ".gopanel_release_synced")
	artifact := filepath.Join(releaseDir, "current.txt")
	if err := os.WriteFile(marker, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("current"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := resetPipelineReleaseSyncMarker(releaseDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("release marker was not removed: %v", err)
	}
	if content, err := os.ReadFile(artifact); err != nil || string(content) != "current" {
		t.Fatalf("current release was changed: %q, %v", content, err)
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
