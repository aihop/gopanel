//go:build desktop

package main

import "testing"

func TestDecodeDesktopCodeSummary(t *testing.T) {
	summary, err := decodeDesktopCodeSummary(map[string]interface{}{
		"attention": float64(3), "running": float64(2), "queued": float64(-1),
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attention != 3 || summary.Running != 2 || summary.Queued != 0 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}
