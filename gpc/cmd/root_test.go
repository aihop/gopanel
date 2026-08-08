package cmd

import "testing"

func TestDefaultGoPanelServiceName(t *testing.T) {
	tests := map[string]string{
		"darwin":  "io.aihop.gopanel",
		"linux":   "gopanel.service",
		"windows": "GoPanel",
	}
	for goos, expected := range tests {
		if actual := defaultGoPanelServiceName(goos); actual != expected {
			t.Fatalf("default service name for %s: got %q, want %q", goos, actual, expected)
		}
	}
}
