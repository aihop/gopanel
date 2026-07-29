package api

import (
	"strings"
	"testing"
	"time"
)

func TestParseCodexRuntimeCompletedTurn(t *testing.T) {
	transcript := strings.Join([]string{
		`{"timestamp":"2026-07-29T10:00:00Z","type":"event_msg","payload":{"type":"task_started","started_at":1785319200}}`,
		`{"timestamp":"2026-07-29T10:00:01Z","type":"turn_context","payload":{"model":"gpt-5.6-sol","approval_policy":"on-request"}}`,
		`{"timestamp":"2026-07-29T10:00:02Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":120,"cached_input_tokens":80,"output_tokens":30,"reasoning_output_tokens":10,"total_tokens":150}}}}`,
		`{"timestamp":"2026-07-29T10:00:03Z","type":"event_msg","payload":{"type":"agent_message","phase":"final_answer","message":"修复完成"}}`,
		`{"timestamp":"2026-07-29T10:00:04Z","type":"event_msg","payload":{"type":"task_complete","completed_at":1785319204}}`,
	}, "\n")
	state, err := parseCodexRuntime(strings.NewReader(transcript), time.Date(2026, 7, 29, 10, 0, 10, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if state.ResponseState != "completed" || !state.NeedsInput || state.AwaitingApproval {
		t.Fatalf("unexpected completed state: %#v", state)
	}
	if state.Model != "gpt-5.6-sol" || state.TotalTokens != 150 || state.CachedInputTokens != 80 {
		t.Fatalf("unexpected runtime metadata: %#v", state)
	}
	if state.LastAssistantPreview != "修复完成" {
		t.Fatalf("unexpected assistant preview: %q", state.LastAssistantPreview)
	}
}

func TestParseCodexRuntimeDetectsApprovalWait(t *testing.T) {
	transcript := strings.Join([]string{
		`{"timestamp":"2026-07-29T10:00:00Z","type":"event_msg","payload":{"type":"task_started","started_at":1785319200}}`,
		`{"timestamp":"2026-07-29T10:00:01Z","type":"turn_context","payload":{"approval_policy":"on-request"}}`,
		`{"timestamp":"2026-07-29T10:00:02Z","type":"response_item","payload":{"type":"custom_tool_call","name":"exec","call_id":"call-1"}}`,
	}, "\n")
	state, err := parseCodexRuntime(strings.NewReader(transcript), time.Date(2026, 7, 29, 10, 0, 7, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if state.ResponseState != "needsInput" || !state.NeedsInput || !state.AwaitingApproval {
		t.Fatalf("unexpected approval state: %#v", state)
	}
}

func TestParseCodexRuntimeDoesNotMisclassifyAnsweredCall(t *testing.T) {
	transcript := strings.Join([]string{
		`{"timestamp":"2026-07-29T10:00:00Z","type":"event_msg","payload":{"type":"task_started","started_at":1785319200}}`,
		`{"timestamp":"2026-07-29T10:00:01Z","type":"turn_context","payload":{"approval_policy":"on-request"}}`,
		`{"timestamp":"2026-07-29T10:00:02Z","type":"response_item","payload":{"type":"function_call","name":"shell","call_id":"call-1"}}`,
		`{"timestamp":"2026-07-29T10:00:03Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call-1"}}`,
	}, "\n")
	state, err := parseCodexRuntime(strings.NewReader(transcript), time.Date(2026, 7, 29, 10, 0, 20, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if state.ResponseState != "responding" || state.NeedsInput || state.AwaitingApproval {
		t.Fatalf("unexpected responding state: %#v", state)
	}
}

func TestParseCodexRuntimeTailInfersActiveTurn(t *testing.T) {
	transcript := strings.Join([]string{
		`{"timestamp":"2026-07-29T10:00:02Z","type":"turn_context","payload":{"model":"gpt-5.6-sol","approval_policy":"never"}}`,
		`{"timestamp":"2026-07-29T10:00:03Z","type":"event_msg","payload":{"type":"agent_message","phase":"commentary","message":"继续处理中"}}`,
		`{"timestamp":"2026-07-29T10:00:04Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}}`,
	}, "\n")
	state, err := parseCodexRuntime(strings.NewReader(transcript), time.Date(2026, 7, 29, 10, 0, 5, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if state.ResponseState != "responding" || state.NeedsInput {
		t.Fatalf("unexpected tail state: %#v", state)
	}
}
