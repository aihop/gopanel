package repo

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPipelineRecordPersistsExpectedCommit(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "pipeline.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repository := NewPipelineRecord(database)
	if err := repository.MigrateTable(); err != nil {
		t.Fatal(err)
	}
	expectedCommit := strings.Repeat("a", 40)
	record := &model.PipelineRecord{PipelineID: 1, Status: "pending", Version: "1.0.0", ExpectedCommit: expectedCommit, ImageID: "sha256:local", ImageDigest: "sha256:digest", ImageRef: "repo/app@sha256:digest"}
	if err := repository.Create(record); err != nil {
		t.Fatal(err)
	}
	stored, err := repository.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ExpectedCommit != expectedCommit {
		t.Fatalf("expectedCommit = %q, want %q", stored.ExpectedCommit, expectedCommit)
	}
	if stored.ImageID != record.ImageID || stored.ImageDigest != record.ImageDigest || stored.ImageRef != record.ImageRef {
		t.Fatalf("image identity not persisted: %+v", stored)
	}
}

func TestReleasePersistsArtifactManifest(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "release.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repository := NewRelease(database)
	if err := repository.MigrateTable(); err != nil {
		t.Fatal(err)
	}
	release := &model.Release{PipelineID: 1, PipelineRecordID: 2, Version: "1.0.0", SourceType: "image", ImageDigest: "sha256:image", ArtifactDigest: "sha256:image", ArtifactManifest: `{"schemaVersion":1}`, Status: "ready"}
	if err := repository.Create(release); err != nil {
		t.Fatal(err)
	}
	stored, err := repository.Get(release.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ImageDigest != release.ImageDigest || stored.ArtifactDigest != release.ArtifactDigest || stored.ArtifactManifest != release.ArtifactManifest {
		t.Fatalf("release artifact contract not persisted: %+v", stored)
	}
}
