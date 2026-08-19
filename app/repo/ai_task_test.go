package repo

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDeleteTaskRemovesTaskRecordsAndResetsSession(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ai-task.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&model.AIDevSession{}, &model.AITask{}, &model.AIMessage{}, &model.AIInstruction{},
		&model.AIApproval{}, &model.AIExecutionRun{}, &model.AIPreview{}, &model.AITimelineEvent{},
	); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	session := &model.AIDevSession{UserID: 1, Title: "session", WorkDir: "/tmp", CurrentStage: "failed"}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, Title: "task", WorkDir: "/tmp", Status: "failed"}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	session.LastTaskID = task.ID
	if err := database.Save(session).Error; err != nil {
		t.Fatal(err)
	}
	instruction := &model.AIInstruction{SessionID: session.ID, UserID: 1, TaskID: task.ID, Content: "run", Status: "failed"}
	if err := database.Create(instruction).Error; err != nil {
		t.Fatal(err)
	}
	records := []any{
		&model.AIMessage{SessionID: session.ID, TaskID: task.ID, Role: "agent", Content: "done"},
		&model.AIApproval{SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID, RequestUserID: 1, Title: "approve", Content: "content"},
		&model.AIExecutionRun{SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID, ExecutorID: "codex", Prompt: "run", Status: "failed", StartedAt: time.Now()},
		&model.AIPreview{SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID, Title: "preview", URL: "https://example.com"},
		&model.AITimelineEvent{SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID, EventType: "result", Title: "done"},
	}
	for _, record := range records {
		if err := database.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := (&aiTaskRepo{}).DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	var taskCount int64
	if err := database.Model(&model.AITask{}).Where("id = ?", task.ID).Count(&taskCount).Error; err != nil || taskCount != 0 {
		t.Fatalf("remaining task count = %d, err = %v", taskCount, err)
	}
	for _, target := range []any{
		&model.AIMessage{}, &model.AIInstruction{}, &model.AIApproval{},
		&model.AIExecutionRun{}, &model.AIPreview{}, &model.AITimelineEvent{},
	} {
		var count int64
		if err := database.Model(target).Where("task_id = ?", task.ID).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("remaining %T count = %d, err = %v", target, count, err)
		}
	}
	if err := database.First(session, session.ID).Error; err != nil || session.LastTaskID != 0 || session.CurrentStage != "idle" {
		t.Fatalf("session was not reset: %#v, err = %v", session, err)
	}
}

func TestGetTasksByProjectIDPrioritizesActiveTasks(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "task-list.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AITask{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	now := time.Now()
	tasks := []*model.AITask{
		{CreatedAt: now, UserID: 1, ProjectID: 1, Title: "recent completed", WorkDir: "/tmp", Status: "completed"},
		{CreatedAt: now.Add(-time.Hour), UserID: 2, ProjectID: 1, Title: "old running", WorkDir: "/tmp", Status: "running"},
		{CreatedAt: now.Add(-2 * time.Hour), UserID: 3, ProjectID: 1, Title: "old approval", WorkDir: "/tmp", Status: "pending_approval"},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	items, total, err := (&aiTaskRepo{}).GetTasksByProjectID(1, 1, 2, false, AITaskListOrderMonitor)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(items) != 2 || items[0].Status != "pending_approval" || items[1].Status != "running" {
		t.Fatalf("unexpected task order: total=%d items=%#v", total, items)
	}
}

func TestGetTasksByUserIDRecentOrderUsesCreationTime(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "recent-task-list.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AITask{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	now := time.Now()
	tasks := []*model.AITask{
		{CreatedAt: now.Add(-time.Hour), UserID: 1, Title: "old running", WorkDir: "/tmp", Status: "running"},
		{CreatedAt: now, UserID: 1, Title: "new completed", WorkDir: "/tmp", Status: "completed"},
		{CreatedAt: now.Add(-2 * time.Hour), UserID: 1, Title: "old approval", WorkDir: "/tmp", Status: "pending_approval"},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}

	items, total, err := (&aiTaskRepo{}).GetTasksByUserID(1, 1, 2, false, AITaskListOrderRecent)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(items) != 2 || items[0].Title != "new completed" || items[1].Title != "old running" {
		t.Fatalf("unexpected recent task order: total=%d items=%#v", total, items)
	}
}

func TestListCodeTasksSeparatesArchived(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "task-archive.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AITask{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	now := time.Now()
	archivedAt := now.Add(-time.Minute)
	tasks := []*model.AITask{
		{CreatedAt: now, UserID: 1, ProjectID: 1, Title: "visible", WorkDir: "/tmp", Status: "running"},
		{CreatedAt: now, UserID: 1, ProjectID: 1, Title: "archived running", WorkDir: "/tmp", Status: "running", ArchivedAt: &archivedAt},
		{CreatedAt: now, UserID: 1, ProjectID: 2, Title: "other project", WorkDir: "/tmp", Status: "completed"},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}

	// 默认列表：归档的不出现，而且总数也不能把它算进去，否则前端分页对不上。
	items, total, err := (&aiTaskRepo{}).GetTasksByUserID(1, 1, 50, false, AITaskListOrderMonitor)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("active list should exclude archived: total=%d items=%d", total, len(items))
	}
	for _, item := range items {
		if item.ArchivedAt != nil {
			t.Fatalf("archived task leaked into active list: %#v", item)
		}
	}

	// 归档列表：只出归档的，这是找回误归档任务的唯一入口。
	archivedItems, archivedTotal, err := (&aiTaskRepo{}).GetTasksByUserID(1, 1, 50, true, AITaskListOrderMonitor)
	if err != nil {
		t.Fatal(err)
	}
	if archivedTotal != 1 || len(archivedItems) != 1 || archivedItems[0].Title != "archived running" {
		t.Fatalf("unexpected archived list: total=%d items=%#v", archivedTotal, archivedItems)
	}

	// 归档过滤要和项目过滤叠加，不能互相覆盖。
	_, projectTotal, err := (&aiTaskRepo{}).GetTasksByProjectAndUserID(1, 1, 1, 50, false, AITaskListOrderMonitor)
	if err != nil {
		t.Fatal(err)
	}
	if projectTotal != 1 {
		t.Fatalf("project + archived filters should combine: total=%d", projectTotal)
	}
}

func TestDeleteTaskAndSessionIsAtomic(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "delete-session.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&model.AIDevSession{}, &model.AITask{}, &model.AIMessage{}, &model.AIInstruction{},
		&model.AIApproval{}, &model.AIExecutionRun{}, &model.AIPreview{}, &model.AITimelineEvent{},
		&model.AICodeDelivery{}, &model.AICodeDeliveryJob{}, &model.AICodeDeliveryLease{},
		&model.AIDevSessionRepository{},
	); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	session := &model.AIDevSession{UserID: 1, Title: "session", WorkDir: "/tmp"}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, Title: "task", WorkDir: "/tmp"}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	message := &model.AIMessage{SessionID: session.ID, TaskID: 0, Role: "system", Content: "legacy session record"}
	if err := database.Create(message).Error; err != nil {
		t.Fatal(err)
	}
	repositoryKeys, _ := json.Marshal([]string{"repository"})
	job := &model.AICodeDeliveryJob{
		SessionID: session.ID, ProjectID: 1, UserID: 1, Status: "running", Stage: "pushing",
		RepositoryKeys: string(repositoryKeys), LeaseOwner: "runner",
	}
	if err := database.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AICodeDeliveryLease{RepositoryKey: "repository", JobID: job.ID, LeaseOwner: "runner"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := (&aiTaskRepo{}).DeleteTaskAndSession(task.ID, session.ID); err != nil {
		t.Fatal(err)
	}
	for _, target := range []any{
		&model.AITask{}, &model.AIDevSession{}, &model.AIMessage{},
		&model.AICodeDeliveryJob{}, &model.AICodeDeliveryLease{},
	} {
		var count int64
		if err := database.Model(target).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("remaining %T count = %d, err = %v", target, count, err)
		}
	}
}
