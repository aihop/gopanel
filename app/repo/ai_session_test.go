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

func TestQueuedInstructionQueriesAndCancellation(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "queued-instructions.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIInstruction{}); err != nil {
		t.Fatal(err)
	}
	global.DB = db
	instructions := []*model.AIInstruction{
		{SessionID: 7, UserID: 1, Content: "first", Status: "queued"},
		{SessionID: 7, UserID: 1, Content: "running", Status: "running"},
		{SessionID: 8, UserID: 1, Content: "second", Status: "queued"},
	}
	if err := db.Create(&instructions).Error; err != nil {
		t.Fatal(err)
	}
	repository := &aiDevSessionRepo{}
	ids, err := repository.GetQueuedInstructionIDs(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != instructions[0].ID || ids[1] != instructions[2].ID {
		t.Fatalf("queued IDs = %v", ids)
	}
	cancelled, err := repository.CancelQueuedInstructions(7)
	if err != nil || cancelled != 1 {
		t.Fatalf("cancelled = %d, err = %v", cancelled, err)
	}
	var first model.AIInstruction
	if err := db.First(&first, instructions[0].ID).Error; err != nil || first.Status != "cancelled" {
		t.Fatalf("cancelled instruction = %#v, err = %v", first, err)
	}
}

func TestGetSessionsLoadsUpdatedCurrentTaskTitle(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "session-title.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIDevSession{}, &model.AITask{}); err != nil {
		t.Fatal(err)
	}
	global.DB = db
	session := &model.AIDevSession{UserID: 1, Title: "session title", WorkDir: "/tmp"}
	if err := db.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, Title: "first task title", WorkDir: "/tmp"}
	if err := db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(session).Update("last_task_id", task.ID).Error; err != nil {
		t.Fatal(err)
	}

	repository := &aiDevSessionRepo{}
	sessions, _, err := repository.GetSessionsByUserID(1, 0, 1, 20)
	if err != nil || len(sessions) != 1 || sessions[0].CurrentTaskTitle != task.Title {
		t.Fatalf("current task title = %#v, err = %v", sessions, err)
	}
	if err := db.Model(task).Update("title", "updated task title").Error; err != nil {
		t.Fatal(err)
	}
	sessions, _, err = repository.GetSessionsByUserID(1, 0, 1, 20)
	if err != nil || sessions[0].CurrentTaskTitle != "updated task title" {
		t.Fatalf("updated current task title = %#v, err = %v", sessions, err)
	}
}

func TestGetSessionsHidesFailedInitializationWithoutTask(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "session-filter.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIDevSession{}, &model.AITask{}); err != nil {
		t.Fatal(err)
	}
	global.DB = db
	sessions := []*model.AIDevSession{
		{UserID: 1, ProjectID: 4, Title: "invalid", WorkDir: "/tmp", Status: "failed", CurrentStage: "initialization_failed"},
		{UserID: 1, ProjectID: 4, Title: "initializing", WorkDir: "/tmp", Status: "initializing", CurrentStage: "syncing_base"},
		{UserID: 1, ProjectID: 4, Title: "task failed", WorkDir: "/tmp", Status: "failed", CurrentStage: "initialization_failed", LastTaskID: 9},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AITask{ID: 9, UserID: 1, SessionID: sessions[2].ID, Title: "task failed", WorkDir: "/tmp"}).Error; err != nil {
		t.Fatal(err)
	}

	got, total, err := (&aiDevSessionRepo{}).GetSessionsByUserID(1, 4, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	titles := make(map[string]bool, len(got))
	for _, session := range got {
		titles[session.Title] = true
	}
	if total != 2 || len(got) != 2 || !titles["task failed"] || !titles["initializing"] || titles["invalid"] {
		t.Fatalf("filtered sessions = %#v, total = %d", got, total)
	}
}

func TestGetSessionsByProjectIncludesAllUsersAndTaskTitles(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "project-sessions.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIDevSession{}, &model.AITask{}); err != nil {
		t.Fatal(err)
	}
	global.DB = db
	sessions := []*model.AIDevSession{
		{UserID: 1, ProjectID: 7, Title: "first", WorkDir: "/tmp"},
		{UserID: 2, ProjectID: 7, Title: "second", WorkDir: "/tmp"},
		{UserID: 3, ProjectID: 8, Title: "other project", WorkDir: "/tmp"},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatal(err)
	}
	tasks := []*model.AITask{
		{UserID: 1, SessionID: sessions[0].ID, ProjectID: 7, Title: "first task", WorkDir: "/tmp"},
		{UserID: 2, SessionID: sessions[1].ID, ProjectID: 7, Title: "second task", WorkDir: "/tmp"},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	for index := range tasks {
		if err := db.Model(sessions[index]).Update("last_task_id", tasks[index].ID).Error; err != nil {
			t.Fatal(err)
		}
	}

	got, total, err := (&aiDevSessionRepo{}).GetSessionsByProjectID(7, 1, 20)
	if err != nil || total != 2 || len(got) != 2 {
		t.Fatalf("project sessions = %#v, total = %d, err = %v", got, total, err)
	}
	titles := map[string]bool{got[0].CurrentTaskTitle: true, got[1].CurrentTaskTitle: true}
	if !titles["first task"] || !titles["second task"] {
		t.Fatalf("project task titles = %#v", titles)
	}
}

// 会话列表要和 PC 端的任务列表同口径：没有任务的会话不展示，
// 否则隔离会话惰性建任务留下的残骸会只在会话列表里堆积，两端数量对不上。
func TestGetSessionsHidesOrphanSessionsWithoutTask(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "session-orphan.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIDevSession{}, &model.AITask{}); err != nil {
		t.Fatal(err)
	}
	global.DB = db
	sessions := []*model.AIDevSession{
		{UserID: 1, ProjectID: 4, Title: "有任务", WorkDir: "/tmp", Status: "active", CurrentStage: "completed", LastTaskID: 21},
		{UserID: 1, ProjectID: 4, Title: "初始化中", WorkDir: "/tmp", Status: "initializing", CurrentStage: "syncing_base"},
		// 以下三种都没有任务，都不该出现在列表里。
		{UserID: 1, ProjectID: 4, Title: "卡住的交互会话", WorkDir: "/tmp", Status: "active", CurrentStage: "interactive"},
		{UserID: 1, ProjectID: 4, Title: "空闲无任务", WorkDir: "/tmp", Status: "active", CurrentStage: "idle"},
		// 指针还在但任务已被删掉，也就是「删了却还显示」的那种。
		{UserID: 1, ProjectID: 4, Title: "任务已删除", WorkDir: "/tmp", Status: "active", CurrentStage: "interactive", LastTaskID: 99},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AITask{ID: 21, UserID: 1, SessionID: sessions[0].ID, Title: "有任务", WorkDir: "/tmp"}).Error; err != nil {
		t.Fatal(err)
	}

	got, total, err := (&aiDevSessionRepo{}).GetSessionsByUserID(1, 4, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	titles := make(map[string]bool, len(got))
	for _, session := range got {
		titles[session.Title] = true
	}
	if total != 2 || len(got) != 2 {
		t.Fatalf("orphan sessions leaked into the list: %#v (total=%d)", got, total)
	}
	if !titles["有任务"] || !titles["初始化中"] {
		t.Fatalf("expected the task-backed and initializing sessions: %#v", titles)
	}
	if titles["任务已删除"] || titles["卡住的交互会话"] || titles["空闲无任务"] {
		t.Fatalf("orphan sessions must be hidden: %#v", titles)
	}
}
