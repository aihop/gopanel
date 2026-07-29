package repo

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAIGroupRepositoryFiltersProjectsByCreator(t *testing.T) {
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = db
	repository := NewAIGroupRepo()
	for _, project := range []*model.AIGroup{
		{Name: "first", WorkDir: "/workspace/first", CreatorID: 1},
		{Name: "second", WorkDir: "/workspace/second", CreatorID: 2},
	} {
		if err := repository.CreateGroup(project); err != nil {
			t.Fatal(err)
		}
	}

	projects, total, err := repository.GetGroups(1, false, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(projects) != 1 || projects[0].CreatorID != 1 {
		t.Fatalf("unexpected scoped projects: total=%d projects=%#v", total, projects)
	}
	projects, total, err = repository.GetGroups(1, true, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(projects) != 2 {
		t.Fatalf("unexpected super-admin projects: total=%d projects=%#v", total, projects)
	}
	projects[0].Name = "updated"
	if err := repository.UpdateGroup(projects[0]); err != nil {
		t.Fatal(err)
	}
	updated, err := repository.GetGroupByID(projects[0].ID)
	if err != nil || updated.Name != "updated" {
		t.Fatalf("unexpected updated project: %#v, %v", updated, err)
	}
}
