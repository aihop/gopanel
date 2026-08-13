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
	record := &model.PipelineRecord{PipelineID: 1, Status: "pending", Version: "1.0.0", ExpectedCommit: expectedCommit}
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
}
