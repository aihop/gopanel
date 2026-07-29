package service

import (
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/repo"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUserNoteServiceRejectsOversizedContent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repository := repo.NewUserNote(db)
	if err := repository.MigrateTable(); err != nil {
		t.Fatal(err)
	}
	service := &UserNoteService{repo: repository}

	if _, err := service.Save(1, strings.Repeat("记", UserNoteMaxLength+1)); err == nil {
		t.Fatal("expected oversized note to be rejected")
	}
	if note, err := service.Save(1, strings.Repeat("记", UserNoteMaxLength)); err != nil || note.Content == "" {
		t.Fatalf("expected note at length limit to be saved: %#v, %v", note, err)
	}
}
