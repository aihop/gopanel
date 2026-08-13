package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func TestParsePipelineImageInspect(t *testing.T) {
	artifact, err := parsePipelineImageInspect("registry:5000/team/app:v1", []byte(`[{"Id":"sha256:local","RepoDigests":["other/app@sha256:other","registry:5000/team/app@sha256:remote"]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Digest != "sha256:remote" || artifact.RepoDigest != "registry:5000/team/app@sha256:remote" || artifact.ImmutableRef != artifact.RepoDigest {
		t.Fatalf("unexpected image artifact: %+v", artifact)
	}

	local, err := parsePipelineImageInspect("app:v1", []byte(`[{"Id":"sha256:local","RepoDigests":[]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if local.Digest != "sha256:local" || local.ImmutableRef != "sha256:local" {
		t.Fatalf("local image ID should be immutable fallback: %+v", local)
	}
}

func TestStepBuildImageExecutesRuntimeCommands(t *testing.T) {
	original := pipelineRuntimeCommand
	defer func() { pipelineRuntimeCommand = original }()
	var calls [][]string
	pipelineRuntimeCommand = func(_ context.Context, args ...string) (*exec.Cmd, error) {
		calls = append(calls, append([]string(nil), args...))
		if len(args) > 1 && args[0] == "image" && args[1] == "inspect" {
			return exec.Command("sh", "-c", `printf '[{"Id":"sha256:built","RepoDigests":[]}]'`), nil
		}
		return exec.Command("sh", "-c", "printf build-ok"), nil
	}

	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	defer func() { global.CONF.System.BaseDir = oldBaseDir }()
	logger := GetPipelineLogger(910001)
	defer RemovePipelineLogger(910001)

	pipeline := &model.Pipeline{PipelineKey: "demo", ArtifactPath: "."}
	artifact, err := (&PipelineService{}).stepBuildImage(context.Background(), logger, pipeline, t.TempDir(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Tag != "demo:record-7" || artifact.ImmutableRef != "sha256:built" {
		t.Fatalf("unexpected artifact: %+v", artifact)
	}
	if len(calls) != 2 || calls[0][0] != "build" || !reflect.DeepEqual(calls[1], []string{"image", "inspect", "demo:record-7"}) {
		t.Fatalf("runtime commands were not executed in order: %v", calls)
	}
}

func TestStepBuildImageReturnsCommandFailure(t *testing.T) {
	original := pipelineRuntimeCommand
	defer func() { pipelineRuntimeCommand = original }()
	pipelineRuntimeCommand = func(_ context.Context, _ ...string) (*exec.Cmd, error) {
		return exec.Command("sh", "-c", "echo build-failed >&2; exit 9"), nil
	}
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	defer func() { global.CONF.System.BaseDir = oldBaseDir }()
	logger := GetPipelineLogger(910002)
	defer RemovePipelineLogger(910002)
	_, err := (&PipelineService{}).stepBuildImage(context.Background(), logger, &model.Pipeline{PipelineKey: "demo"}, t.TempDir(), 8)
	if err == nil || !strings.Contains(err.Error(), "build-failed") {
		t.Fatalf("expected command failure, got %v", err)
	}
}

func TestBuildPipelineReleaseArtifactArchiveManifest(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "app.zip")
	if err := os.WriteFile(archive, []byte("artifact-content"), 0644); err != nil {
		t.Fatal(err)
	}
	pipeline := &model.Pipeline{ID: 3, PipelineKey: "demo", ArtifactPath: "dist", RunnerMode: "runner", RunnerConfig: `{"containerPort":"3000","startCommand":"node server.js","workingDir":"/app"}`}
	record := &model.PipelineRecord{ID: 9, PipelineID: 3, CommitHash: strings.Repeat("a", 40), ArchiveFile: archive, RunnerHostPort: 32000}
	artifact, err := buildPipelineReleaseArtifact(context.Background(), pipeline, record)
	if err != nil {
		t.Fatal(err)
	}
	var manifest model.ArtifactManifest
	if err := json.Unmarshal([]byte(artifact.artifactManifest), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Type != model.ArtifactTypeStaticArchive || manifest.Digest != artifact.artifactDigest || manifest.Archive == nil || manifest.Archive.Path != archive {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	if manifest.Runtime.Port != 3000 || manifest.Runtime.StartCommand != "node server.js" || manifest.Runtime.WorkingDir != "/app" {
		t.Fatalf("runtime contract missing: %+v", manifest.Runtime)
	}
}

func TestPipelineDirectoryDigestIsStableAndContentAddressed(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "nested", "app.txt")
	if err := os.WriteFile(file, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	first, firstSize, err := pipelineDirectoryDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, secondSize, err := pipelineDirectoryDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || firstSize != secondSize {
		t.Fatalf("digest should be stable: %s/%d != %s/%d", first, firstSize, second, secondSize)
	}
	if err := os.WriteFile(file, []byte("v2"), 0644); err != nil {
		t.Fatal(err)
	}
	changed, _, err := pipelineDirectoryDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if changed == first {
		t.Fatal("digest should change when content changes")
	}
	if err := os.Chmod(file, 0755); err != nil {
		t.Fatal(err)
	}
	modeChanged, _, err := pipelineDirectoryDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if modeChanged == changed {
		t.Fatal("digest should change when executable permission changes")
	}
}

func TestPipelineArtifactRuntimeUsesApplicationPort(t *testing.T) {
	runner := pipelineArtifactRuntime(&model.Pipeline{RunnerMode: "runner", RunnerConfig: `{"containerPort":"3000"}`}, &model.PipelineRecord{RunnerHostPort: 32000})
	if runner.Port != 3000 {
		t.Fatalf("runner manifest port = %d, want container port 3000", runner.Port)
	}
	image := pipelineArtifactRuntime(&model.Pipeline{ActionParams: `{"exposePort":8080}`}, &model.PipelineRecord{})
	if image.Port != 8080 {
		t.Fatalf("image manifest port = %d, want exposePort 8080", image.Port)
	}
}

func TestPipelineImageArtifactFromRecordRequiresIdentity(t *testing.T) {
	original := pipelineRuntimeCommand
	defer func() { pipelineRuntimeCommand = original }()
	pipelineRuntimeCommand = func(_ context.Context, _ ...string) (*exec.Cmd, error) { return nil, errors.New("runtime unavailable") }
	_, err := pipelineImageArtifactFromRecord(context.Background(), &model.PipelineRecord{ImageTag: "app:latest"})
	if err == nil {
		t.Fatal("release should reject a mutable image without immutable identity")
	}
}
