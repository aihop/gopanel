//go:build !windows

package api

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestHostTerminalManagerCreateWriteAndStop(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "terminal.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.HostTerminalSession{}, &model.HostTerminalAuditEvent{}); err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = oldDB })
	manager := &hostTerminalManager{sessions: make(map[uint]*hostTerminal)}
	record, err := manager.create(createHostTerminalRequest{Shell: "sh", WorkDir: t.TempDir(), Cols: 80, Rows: 24}, 9, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	session := manager.get(record.ID)
	if session == nil || record.Status != "running" || record.PID == 0 {
		t.Fatalf("terminal did not start: %#v", record)
	}
	subscriber, _ := session.subscribe(9, "127.0.0.1", false)
	if err := session.write(subscriber.ID, []byte("printf 'gopanel-terminal-ok\\n'\r")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		session.mu.Lock()
		output := string(session.history)
		session.mu.Unlock()
		if strings.Contains(output, "gopanel-terminal-ok") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	session.mu.Lock()
	output := string(session.history)
	session.mu.Unlock()
	if !strings.Contains(output, "gopanel-terminal-ok") {
		t.Fatalf("terminal output unavailable: %q", output)
	}
	session.unsubscribe(subscriber)
	if !manager.stop(record.ID) {
		t.Fatal("terminal stop failed")
	}
	select {
	case <-session.done:
	case <-time.After(3 * time.Second):
		t.Fatal("terminal process did not exit")
	}
	var stored model.HostTerminalSession
	if err := database.First(&stored, record.ID).Error; err != nil || stored.Status != "stopped" || stored.EndedAt == nil {
		t.Fatalf("terminal final state was not persisted: %#v, %v", stored, err)
	}
}
