package helper

import (
	"testing"
	"time"
)

func TestParseSSHLogLineAccepted(t *testing.T) {
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.Local)
	line := "Apr 24 10:20:30 host sshd[1234]: Accepted password for root from 1.2.3.4 port 51234 ssh2"

	event, ok := parseSSHLogLine(line, "linux", "auth.log", now)
	if !ok {
		t.Fatalf("expected line to be parsed")
	}
	if event.Status != "Success" {
		t.Fatalf("expected Success, got %s", event.Status)
	}
	if event.Username != "root" {
		t.Fatalf("expected root, got %s", event.Username)
	}
	if event.SourceIP != "1.2.3.4" {
		t.Fatalf("expected 1.2.3.4, got %s", event.SourceIP)
	}
	if event.AuthMethod != "password" {
		t.Fatalf("expected password, got %s", event.AuthMethod)
	}
}

func TestParseSSHLogLineFailedInvalidUser(t *testing.T) {
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))
	line := "2026-04-24T10:20:30+0800 host sshd[4321]: Failed password for invalid user admin from 5.6.7.8 port 41234 ssh2"

	event, ok := parseSSHLogLine(line, "linux", "journalctl", now)
	if !ok {
		t.Fatalf("expected line to be parsed")
	}
	if event.Status != "Failed" {
		t.Fatalf("expected Failed, got %s", event.Status)
	}
	if event.Username != "admin" {
		t.Fatalf("expected admin, got %s", event.Username)
	}
	if event.SourcePort != "41234" {
		t.Fatalf("expected 41234, got %s", event.SourcePort)
	}
}
