package repo

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUserNoteRepositoryIsolatesAndUpdatesUsers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repository := NewUserNote(db)
	if err := repository.MigrateTable(); err != nil {
		t.Fatal(err)
	}

	if _, err := repository.Save(1, "first note"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Save(2, "second note"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Save(1, "updated note"); err != nil {
		t.Fatal(err)
	}

	first, err := repository.GetByUserID(1)
	if err != nil || first.Content != "updated note" {
		t.Fatalf("unexpected first user note: %#v, %v", first, err)
	}
	second, err := repository.GetByUserID(2)
	if err != nil || second.Content != "second note" {
		t.Fatalf("unexpected second user note: %#v, %v", second, err)
	}
	var count int64
	if err := db.Model(&model.UserNote{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("expected one note per user, got %d: %v", count, err)
	}
}

func TestUserNoteRepositoryReturnsEmptyNote(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repository := NewUserNote(db)
	if err := repository.MigrateTable(); err != nil {
		t.Fatal(err)
	}

	note, err := repository.GetByUserID(9)
	if err != nil || note.UserID != 9 || note.Content != "" {
		t.Fatalf("unexpected empty note: %#v, %v", note, err)
	}
}
