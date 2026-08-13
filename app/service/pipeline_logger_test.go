package service

import (
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/global"
)

func TestPipelineLoggerSubscribeKeepsSnapshotAndLiveLogs(t *testing.T) {
	recordID := uint(992001)
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	t.Cleanup(func() {
		RemovePipelineLogger(recordID)
		global.CONF.System.BaseDir = oldBaseDir
	})

	logger := GetPipelineLogger(recordID)
	logger.Info("snapshot line")
	subscribedLogger, logs, updates, active := SubscribePipelineLogger(recordID)
	if !active || subscribedLogger != logger || len(logs) != 1 || !strings.Contains(logs[0], "snapshot line") {
		t.Fatalf("subscription snapshot is incomplete: active=%v logs=%v", active, logs)
	}
	defer logger.Unsubscribe(updates)

	logger.Info("live line")
	select {
	case line := <-updates:
		if !strings.Contains(line, "live line") {
			t.Fatalf("unexpected live log: %q", line)
		}
	case <-time.After(time.Second):
		t.Fatal("live log was not delivered")
	}
}

func TestPipelineLoggerRestoresHistoryAndFinishesFlowStream(t *testing.T) {
	recordID := uint(992002)
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	t.Cleanup(func() {
		RemovePipelineLogger(recordID)
		global.CONF.System.BaseDir = oldBaseDir
	})

	logger := GetPipelineLogger(recordID)
	logger.Info("before restart")
	RemovePipelineLogger(recordID)

	restored := GetPipelineLogger(recordID)
	logs := restored.GetLogs()
	if len(logs) != 1 || !strings.Contains(logs[0], "before restart") {
		t.Fatalf("persisted logs were not restored: %v", logs)
	}
	_, _, updates, active := SubscribePipelineLogger(recordID)
	if !active {
		t.Fatal("restored logger should be active")
	}

	finishFlowRunLogger(recordID)
	if IsPipelineLoggerActive(recordID) {
		t.Fatal("finished Flow logger should be removed")
	}
	for _, expected := range []string{"Flow 执行结束", "EOF"} {
		select {
		case line, ok := <-updates:
			if !ok || !strings.Contains(line, expected) {
				t.Fatalf("unexpected terminal signal: %q, want %q, open=%v", line, expected, ok)
			}
		case <-time.After(time.Second):
			t.Fatalf("terminal did not receive %q", expected)
		}
	}
	history, err := ReadPipelineLogFromFile(recordID)
	if err != nil || len(history) != 2 || !strings.Contains(history[1], "Flow 执行结束") {
		t.Fatalf("finished Flow history is incomplete: %v, %v", history, err)
	}
}
