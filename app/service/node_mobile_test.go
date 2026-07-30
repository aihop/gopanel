package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func TestMobileNodeSnapshotKeepsMonitoringFields(t *testing.T) {
	lastSeenAt := time.Now().Add(-2 * time.Hour)
	node := model.Node{
		Name:        "edge-a",
		Addr:        "https://private.example:5470",
		Entrance:    "secret-entry",
		AccessToken: "encrypted-read-token",
		Status:      NodeStatusOffline,
		StatusMsg:   "connection refused",
		LastSeenAt:  lastSeenAt,
		Summary:     model.NodeSummary{CPUPercent: 42, MemPercent: 61},
	}

	got := toMobileNodeRes(node, false)
	if got.Name != node.Name || got.Status != NodeStatusOffline {
		t.Fatalf("unexpected mobile node snapshot: %#v", got)
	}
	if got.Summary.CPUPercent != 0 || got.Summary.MemPercent != 0 {
		t.Fatalf("offline mobile node retained stale usage: %#v", got.Summary)
	}
	if got.LastSeenAt == nil || !got.LastSeenAt.Equal(lastSeenAt) {
		t.Fatalf("unexpected last seen time: %#v", got.LastSeenAt)
	}
	if len(got.Warnings) != 1 || got.Warnings[0].Type != "offline" {
		t.Fatalf("unexpected warnings: %#v", got.Warnings)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(payload)
	for _, sensitive := range []string{"private.example", "secret-entry", "encrypted-read-token", "statusMsg", "addr", "entrance", "token"} {
		if strings.Contains(serialized, sensitive) {
			t.Fatalf("mobile node payload leaked %q: %s", sensitive, serialized)
		}
	}
}

func TestLocalMobileNodeUsesReservedID(t *testing.T) {
	got := toMobileNodeRes(model.Node{Name: "controller", Status: NodeStatusOnline}, true)
	if got.ID != 0 || !got.IsLocal {
		t.Fatalf("unexpected local mobile node: %#v", got)
	}
}
