package api

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestNormalizeCodeApprovalPolicy(t *testing.T) {
	tests := map[string]string{
		"":          codeApprovalPolicySafeAuto,
		"safe_auto": codeApprovalPolicySafeAuto,
		"manual":    codeApprovalPolicyManual,
		"full_auto": codeApprovalPolicyFullAuto,
	}
	for input, expected := range tests {
		actual, err := normalizeCodeApprovalPolicy(input)
		if err != nil || actual != expected {
			t.Fatalf("normalize %q = %q, %v", input, actual, err)
		}
	}
	if _, err := normalizeCodeApprovalPolicy("always"); err == nil {
		t.Fatal("expected invalid approval policy to be rejected")
	}
}

func TestCodexApprovalPolicy(t *testing.T) {
	tests := map[string]string{
		codeApprovalPolicyManual:   "untrusted",
		codeApprovalPolicySafeAuto: "on-request",
		codeApprovalPolicyFullAuto: "never",
		"":                         "on-request",
	}
	for input, expected := range tests {
		if actual := codexApprovalPolicy(input); actual != expected {
			t.Fatalf("codex policy for %q = %q", input, actual)
		}
	}
}

func TestCodeSessionRequiresRiskApproval(t *testing.T) {
	if !codeSessionRequiresRiskApproval(nil) {
		t.Fatal("missing session should require approval")
	}
	if !codeSessionRequiresRiskApproval(&model.AIDevSession{ApprovalPolicy: codeApprovalPolicySafeAuto}) {
		t.Fatal("safe auto session should require high-risk approval")
	}
	if codeSessionRequiresRiskApproval(&model.AIDevSession{ApprovalPolicy: codeApprovalPolicyFullAuto}) {
		t.Fatal("full auto session should skip high-risk approval")
	}
}

func TestUpdateCodeApprovalPolicy(t *testing.T) {
	session := &model.AIDevSession{ApprovalPolicy: codeApprovalPolicySafeAuto}
	changed, err := updateCodeApprovalPolicy(session, codeApprovalPolicyFullAuto)
	if err != nil || !changed || session.ApprovalPolicy != codeApprovalPolicyFullAuto {
		t.Fatalf("unexpected policy update: changed=%v policy=%q err=%v", changed, session.ApprovalPolicy, err)
	}
	changed, err = updateCodeApprovalPolicy(session, codeApprovalPolicyFullAuto)
	if err != nil || changed {
		t.Fatalf("unchanged policy should be a no-op: changed=%v err=%v", changed, err)
	}
	if _, err = updateCodeApprovalPolicy(session, "unknown"); err == nil {
		t.Fatal("expected invalid policy to be rejected")
	}
}
