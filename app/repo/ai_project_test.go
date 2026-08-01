package repo

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAIProjectRepositoryFiltersProjectsByCreator(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIProject{}, &model.AITask{}, &model.AIDevSession{}); err != nil {
		t.Fatal(err)
	}
	global.DB = db
	repository := NewAIProjectRepo()
	for _, project := range []*model.AIProject{
		{Name: "first", WorkDir: "/workspace/first", CreatorID: 1},
		{Name: "second", WorkDir: "/workspace/second", CreatorID: 2},
	} {
		if err := repository.CreateProject(project); err != nil {
			t.Fatal(err)
		}
	}

	projects, total, err := repository.GetProjects(1, false, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(projects) != 1 || projects[0].CreatorID != 1 {
		t.Fatalf("unexpected scoped projects: total=%d projects=%#v", total, projects)
	}
	projects, total, err = repository.GetProjects(1, true, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(projects) != 2 {
		t.Fatalf("unexpected super-admin projects: total=%d projects=%#v", total, projects)
	}
	projects[0].Name = "updated"
	if err := repository.UpdateProject(projects[0]); err != nil {
		t.Fatal(err)
	}
	updated, err := repository.GetProjectByID(projects[0].ID)
	if err != nil || updated.Name != "updated" {
		t.Fatalf("unexpected updated project: %#v, %v", updated, err)
	}

	session := &model.AIDevSession{UserID: 1, ProjectID: updated.ID, Title: "active session", WorkDir: "/workspace/first", CurrentStage: "executing"}
	if err := db.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	tasks := []*model.AITask{
		{UserID: 1, SessionID: session.ID, ProjectID: updated.ID, Title: "running task", WorkDir: session.WorkDir, Status: "running"},
		{UserID: 1, SessionID: session.ID, ProjectID: updated.ID, Title: "completed task", WorkDir: session.WorkDir, Status: "completed"},
		{UserID: 2, SessionID: session.ID, ProjectID: updated.ID, Title: "other user task", WorkDir: session.WorkDir, Status: "pending_approval"},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.LoadExecutionSummaries([]*model.AIProject{updated}, 1, false); err != nil {
		t.Fatal(err)
	}
	if updated.TaskCount != 2 || updated.ExecutionSummary.ActiveTaskCount != 1 {
		t.Fatalf("unexpected project task counts: project=%#v summary=%#v", updated, updated.ExecutionSummary)
	}
	if updated.ExecutionSummary.Status != "running" || updated.ExecutionSummary.CurrentStage != "executing" || updated.ExecutionSummary.CurrentTaskTitle != "running task" {
		t.Fatalf("unexpected execution summary: %#v", updated.ExecutionSummary)
	}
}
