package api

import (
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestShouldRequireAIApprovalOnlyForDangerousInstructions(t *testing.T) {
	if shouldRequireAIApproval("帮我修复登录页样式", true) {
		t.Fatal("ordinary development instruction should not require approval")
	}
	if !shouldRequireAIApproval("执行 git reset --hard HEAD", true) {
		t.Fatal("dangerous instruction should require approval")
	}
	if shouldRequireAIApproval("执行 git reset --hard HEAD", false) {
		t.Fatal("disabled approval policy should not require approval")
	}
}

func TestCodeInstructionRequiresRiskApprovalOnlyTightensPolicy(t *testing.T) {
	requested, disabled := true, false
	safeSession := &model.AIDevSession{ApprovalPolicy: codeApprovalPolicySafeAuto}
	fullAutoSession := &model.AIDevSession{ApprovalPolicy: codeApprovalPolicyFullAuto}
	if !codeInstructionRequiresRiskApproval(safeSession, &disabled) {
		t.Fatal("request must not disable the session approval policy")
	}
	if !codeInstructionRequiresRiskApproval(fullAutoSession, &requested) {
		t.Fatal("request should enable risk approval for this instruction")
	}
	if codeInstructionRequiresRiskApproval(fullAutoSession, &disabled) {
		t.Fatal("full-auto instruction should remain unpaused when not requested")
	}
}
