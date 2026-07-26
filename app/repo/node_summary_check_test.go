package repo

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUpdateSummarySerializer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "t.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Node{}); err != nil {
		t.Fatal(err)
	}
	old := global.DB
	global.DB = db
	defer func() { global.DB = old }()

	r := NewNode()
	node := model.Node{Name: "n1", Addr: "https://1.2.3.4:5470", Status: "unknown"}
	if err := r.Create(&node); err != nil {
		t.Fatal(err)
	}

	// 旧写法（map + 结构体值）会报 unsupported type model.NodeSummary
	mapErr := db.Model(&model.Node{}).Where("id = ?", node.ID).
		Updates(map[string]interface{}{"summary": model.NodeSummary{Hostname: "x"}}).Error
	if mapErr == nil {
		t.Fatal("map 更新居然成功了，说明 GORM 行为已变，注释需要跟着改")
	}
	t.Logf("map 更新报错（预期）: %v", mapErr)

	// 新写法：结构体 + Select，serializer 生效
	now := time.Now()
	if err := r.UpdateSummary(node.ID, model.Node{
		Status:     "online",
		StatusMsg:  "",
		Version:    "1.2.1",
		LastSeenAt: now,
		Summary:    model.NodeSummary{Hostname: "node-a", CPUPercent: 12.5, MemPercent: 40},
	}); err != nil {
		t.Fatalf("结构体更新失败: %v", err)
	}

	got, err := r.GetByID(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary.Hostname != "node-a" || got.Summary.CPUPercent != 12.5 || got.Status != "online" || got.Version != "1.2.1" {
		t.Fatalf("回读结果不对: %+v", got)
	}

	// status_msg 的零值（空串）必须能被写回去，否则失败原因会一直挂着
	if err := r.UpdateStatus(node.ID, "offline", "节点不可达"); err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateSummary(node.ID, model.Node{Status: "online", StatusMsg: "", Version: "1.2.1", LastSeenAt: now, Summary: got.Summary}); err != nil {
		t.Fatal(err)
	}
	got2, _ := r.GetByID(node.ID)
	if got2.StatusMsg != "" {
		t.Fatalf("status_msg 零值没写进去: %q", got2.StatusMsg)
	}
	if got2.Summary.Hostname != "node-a" {
		t.Fatalf("summary 丢了: %+v", got2.Summary)
	}
}
