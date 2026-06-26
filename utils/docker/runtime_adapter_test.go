package docker

import (
	"strings"
	"testing"
)

func TestCommandNameUsesBaseName(t *testing.T) {
	if got := commandName("/opt/homebrew/bin/podman-compose"); got != "podman-compose" {
		t.Fatalf("expected podman-compose, got %q", got)
	}
	if got := commandName("/opt/homebrew/bin/podman"); got != "podman" {
		t.Fatalf("expected podman, got %q", got)
	}
}

func TestWithPatchedPathPrependsRuntimeDirs(t *testing.T) {
	env := withPatchedPath([]string{"PATH=/usr/bin:/bin"}, "/opt/homebrew/bin", "/usr/local/bin")

	var pathValue string
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			pathValue = strings.TrimPrefix(kv, "PATH=")
			break
		}
	}
	if pathValue == "" {
		t.Fatal("expected PATH to be present")
	}

	parts := strings.Split(pathValue, ":")
	if len(parts) < 4 {
		t.Fatalf("expected enriched PATH, got %q", pathValue)
	}
	if parts[0] != "/opt/homebrew/bin" {
		t.Fatalf("expected /opt/homebrew/bin first, got %q", parts[0])
	}
	if parts[1] != "/usr/local/bin" {
		t.Fatalf("expected /usr/local/bin second, got %q", parts[1])
	}

	count := 0
	for _, part := range parts {
		if part == "/opt/homebrew/bin" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected /opt/homebrew/bin once, got %d in %q", count, pathValue)
	}
}
