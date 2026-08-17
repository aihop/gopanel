package api

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestLoadCodeDesktopSummary(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AITask{}, &model.AICodeDeliveryJob{}); err != nil {
		t.Fatal(err)
	}
	previousDB := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previousDB })

	tasks := []model.AITask{
		{UserID: 7, SessionID: 101, Title: "approval", Status: "pending_approval"},
		{UserID: 7, SessionID: 102, Title: "running", Status: "running"},
		{UserID: 7, SessionID: 103, Title: "queued", Status: "queued"},
		{UserID: 7, SessionID: 104, Title: "delivery conflict", Status: "completed"},
		{UserID: 7, SessionID: 105, Title: "terminal", AgentName: "terminal", Status: "failed"},
		{UserID: 8, SessionID: 106, Title: "other user", Status: "failed"},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AICodeDeliveryJob{
		UserID: 7, SessionID: 104, ProjectID: 1, Status: "conflict", Stage: "merge",
	}).Error; err != nil {
		t.Fatal(err)
	}

	summary, err := loadCodeDesktopSummary(7)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attention != 2 || summary.Running != 1 || summary.Queued != 1 {
		t.Fatalf("unexpected desktop summary: %#v", summary)
	}
}
