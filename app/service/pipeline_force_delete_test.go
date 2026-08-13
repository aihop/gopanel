package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

func TestPipelineForceDeleteRemovesPipelineHistoryAndOwnedDirectories(t *testing.T) {
	database := flowTestDatabase(t)
	baseDir := t.TempDir()
	originalBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = baseDir
	t.Cleanup(func() { global.CONF.System.BaseDir = originalBaseDir })

	pipeline := model.Pipeline{Name: "Disposable Pipeline", PipelineKey: "disposable", BuildImage: "host"}
	if err := database.Create(&pipeline).Error; err != nil {
		t.Fatal(err)
	}
	record := model.PipelineRecord{PipelineID: pipeline.ID, Status: "success", Version: "1.0.0"}
	if err := database.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	release := model.Release{PipelineID: pipeline.ID, PipelineRecordID: record.ID, Version: record.Version, SourceType: "archive", Status: "ready"}
	if err := database.Create(&release).Error; err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{pipelineBaseDir(&pipeline), filepath.Join(baseDir, "apps", pipeline.PipelineKey)} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(getLogFilePath(record.ID)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(getLogFilePath(record.ID), []byte("build log"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := NewPipelineApplication(database).ForceDelete(pipeline.ID, pipeline.Name)
	if err != nil {
		t.Fatal(err)
	}
	if result.RecordCount != 1 || result.ReleaseCount != 1 || len(result.CleanupWarnings) != 0 {
		t.Fatalf("unexpected force delete result: %+v", result)
	}
	for _, item := range []interface{}{&model.Pipeline{}, &model.PipelineRecord{}, &model.Release{}} {
		var count int64
		if err := database.Model(item).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("expected %T rows to be deleted, got %d", item, count)
		}
	}
	for _, directory := range []string{pipelineBaseDir(&pipeline), filepath.Join(baseDir, "apps", pipeline.PipelineKey)} {
		if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected directory %s to be removed, got %v", directory, err)
		}
	}
	if _, err := os.Stat(getLogFilePath(record.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected log file to be removed, got %v", err)
	}
}

func TestPipelineForceDeleteRequiresExactName(t *testing.T) {
	database := flowTestDatabase(t)
	pipeline := model.Pipeline{Name: "Protected Pipeline", PipelineKey: "protected", BuildImage: "host"}
	if err := database.Create(&pipeline).Error; err != nil {
		t.Fatal(err)
	}

	_, err := NewPipelineApplication(database).ForceDelete(pipeline.ID, "wrong name")
	if !isBusinessError(err, constant.ErrPipelineForceDeleteName) {
		t.Fatalf("expected confirmation mismatch, got %v", err)
	}
	if err := database.First(&model.Pipeline{}, pipeline.ID).Error; err != nil {
		t.Fatalf("pipeline should remain: %v", err)
	}
}

func TestPipelineForceDeleteRejectsRunningRecord(t *testing.T) {
	database := flowTestDatabase(t)
	pipeline := model.Pipeline{Name: "Running Pipeline", PipelineKey: "running", BuildImage: "host"}
	if err := database.Create(&pipeline).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.PipelineRecord{PipelineID: pipeline.ID, Status: "building", Version: "1.0.0"}).Error; err != nil {
		t.Fatal(err)
	}

	_, err := NewPipelineApplication(database).ForceDelete(pipeline.ID, pipeline.Name)
	if !isBusinessError(err, constant.ErrPipelineForceDeleteRunning) {
		t.Fatalf("expected running record rejection, got %v", err)
	}
}

func TestPipelineForceDeleteRejectsFlowReferences(t *testing.T) {
	database := flowTestDatabase(t)
	pipeline := model.Pipeline{Name: "Flow Pipeline", PipelineKey: "flow", BuildImage: "host"}
	if err := database.Create(&pipeline).Error; err != nil {
		t.Fatal(err)
	}
	flow := model.Flow{Name: "Delivery", ProjectID: 1, PipelineID: pipeline.ID, CreatedBy: 1}
	if err := database.Create(&flow).Error; err != nil {
		t.Fatal(err)
	}

	_, err := NewPipelineApplication(database).ForceDelete(pipeline.ID, pipeline.Name)
	if !isBusinessError(err, constant.ErrPipelineForceDeleteFlow) {
		t.Fatalf("expected flow reference rejection, got %v", err)
	}
}

func TestPipelineForceDeleteRejectsWebsiteUsingRunner(t *testing.T) {
	database := flowTestDatabase(t)
	pipeline := model.Pipeline{Name: "Website Pipeline", PipelineKey: "website", BuildImage: "host"}
	if err := database.Create(&pipeline).Error; err != nil {
		t.Fatal(err)
	}
	containerID := "runner-container"
	if err := database.Create(&model.PipelineRecord{PipelineID: pipeline.ID, Status: "success", Version: "1.0.0", RunnerContainerID: containerID}).Error; err != nil {
		t.Fatal(err)
	}
	website := model.Website{Alias: "production", PrimaryDomain: "example.com", Type: "proxy", Status: "Running", Protocol: "HTTP", ContainerID: containerID}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}

	_, err := NewPipelineApplication(database).ForceDelete(pipeline.ID, pipeline.Name)
	if !isBusinessError(err, constant.ErrPipelineForceDeleteWebsite) {
		t.Fatalf("expected website reference rejection, got %v", err)
	}
}

func isBusinessError(err error, key string) bool {
	var businessError buserr.BusinessError
	return errors.As(err, &businessError) && businessError.Msg == key
}

func TestSafePipelineOwnedDirectoryRejectsEscapes(t *testing.T) {
	baseDir := t.TempDir()
	if _, err := safePipelineOwnedDirectory(baseDir, baseDir); err == nil {
		t.Fatal("expected base directory rejection")
	}
	if _, err := safePipelineOwnedDirectory(baseDir, filepath.Join(baseDir, "..", "outside")); err == nil {
		t.Fatal("expected escaped directory rejection")
	}
	valid, err := safePipelineOwnedDirectory(baseDir, filepath.Join(baseDir, "pipeline-a"))
	if err != nil || valid == "" {
		t.Fatalf("expected valid child directory, got %q, %v", valid, err)
	}
}

func TestRemovePipelineOwnedDirectoriesPreservesSharedLogDirectory(t *testing.T) {
	baseDir := t.TempDir()
	originalBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = baseDir
	t.Cleanup(func() { global.CONF.System.BaseDir = originalBaseDir })
	logDir := filepath.Join(baseDir, "pipelines", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "other.log"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	err := removePipelineOwnedDirectories(&model.Pipeline{ID: 1, PipelineKey: "logs"})
	if err == nil {
		t.Fatal("expected reserved log directory cleanup rejection")
	}
	if _, err := os.Stat(filepath.Join(logDir, "other.log")); err != nil {
		t.Fatalf("shared log file should remain: %v", err)
	}
}
