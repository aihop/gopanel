package api

import "testing"

func TestApplyCodexLiveEventUsesCommentaryThenFinal(t *testing.T) {
	seen := map[string]struct{}{}
	snapshot := applyCodexLiveEvent("", seen, []byte(`{"type":"event_msg","payload":{"type":"agent_message","phase":"commentary","message":"先看对话页"}}`))
	if snapshot != "先看对话页" {
		t.Fatalf("commentary = %q", snapshot)
	}
	snapshot = applyCodexLiveEvent(snapshot, seen, []byte(`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"先看对话页\n\n已经改好了"}]}}`))
	if snapshot != "先看对话页\n\n已经改好了" {
		t.Fatalf("final = %q", snapshot)
	}
	snapshot = applyCodexLiveEvent(snapshot, seen, []byte(`{"type":"event_msg","payload":{"type":"agent_message","phase":"commentary","message":"先看对话页"}}`))
	if snapshot != "先看对话页\n\n已经改好了" {
		t.Fatalf("duplicate changed snapshot: %q", snapshot)
	}
}
