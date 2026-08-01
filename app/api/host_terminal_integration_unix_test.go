//go:build !windows

package api

import (
	"errors"
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
	if err := database.AutoMigrate(&model.AIDevSession{}, &model.HostTerminalSession{}, &model.HostTerminalAuditEvent{}); err != nil {
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
	resumed, err := manager.resume(record.ID)
	if err != nil || resumed.ID != record.ID || resumed.PID != record.PID {
		t.Fatalf("disconnected terminal should resume the original process: %#v, %v", resumed, err)
	}
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

func TestCodeDeliveryRejectsRunningHostTerminalWithoutStoppingIt(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "terminal-delivery.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIDevSession{}, &model.HostTerminalSession{}, &model.HostTerminalAuditEvent{}); err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = oldDB })
	manager := &hostTerminalManager{sessions: make(map[uint]*hostTerminal)}
	record, err := manager.create(createHostTerminalRequest{Shell: "sh", WorkDir: t.TempDir()}, 9, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.validateStoppedForCodeDelivery(); err == nil {
		t.Fatal("delivery should reject a running host terminal")
	}
	if manager.get(record.ID) == nil {
		t.Fatal("delivery stopped the host terminal process")
	}
	var stored model.HostTerminalSession
	if err := database.First(&stored, record.ID).Error; err != nil || stored.Status != "running" {
		t.Fatalf("delivery changed the terminal state: %#v, %v", stored, err)
	}
	if !manager.stop(record.ID) {
		t.Fatal("test cleanup could not stop host terminal")
	}
	if manager.get(record.ID) != nil {
		t.Fatal("test cleanup returned before host terminal exit")
	}
}

func TestDeleteHostTerminalRecordUsesSoftDelete(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "terminal-delete.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIDevSession{}, &model.HostTerminalSession{}, &model.HostTerminalAuditEvent{}); err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = oldDB })
	record := &model.HostTerminalSession{UserID: 9, Status: "interrupted", Shell: "sh", WorkDir: t.TempDir(), StartedAt: time.Now()}
	if err := database.Create(record).Error; err != nil {
		t.Fatal(err)
	}
	recordHostTerminalAudit(record.ID, record.UserID, "delete", "success", "127.0.0.1", "test")
	if err := deleteHostTerminalRecord(record); err != nil {
		t.Fatal(err)
	}
	if err := database.First(&model.HostTerminalSession{}, record.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("soft-deleted session should be hidden: %v", err)
	}
	var deleted model.HostTerminalSession
	if err := database.Unscoped().First(&deleted, record.ID).Error; err != nil || !deleted.DeletedAt.Valid {
		t.Fatalf("soft-deleted session should remain auditable: %#v, %v", deleted, err)
	}
	var auditCount int64
	if err := database.Model(&model.HostTerminalAuditEvent{}).Where("session_id = ?", record.ID).Count(&auditCount).Error; err != nil || auditCount != 1 {
		t.Fatalf("terminal audit should be retained: count=%d err=%v", auditCount, err)
	}
}
