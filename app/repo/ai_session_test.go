package repo

import (
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGetPendingInstructionsDoesNotRepeatRunningWork(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ai-session.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIInstruction{}); err != nil {
		t.Fatal(err)
	}
	global.DB = db
	instructions := []*model.AIInstruction{
		{SessionID: 9, UserID: 1, Content: "queued", Status: "queued"},
		{SessionID: 9, UserID: 1, Content: "running", Status: "running"},
		{SessionID: 9, UserID: 1, Content: "completed", Status: "completed"},
	}
	if err := db.Create(&instructions).Error; err != nil {
		t.Fatal(err)
	}

	got, err := (&aiDevSessionRepo{}).GetPendingInstructionsBySessionID(9)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Content != "queued" {
		t.Fatalf("pending instructions = %#v, want only queued work", got)
	}
}
