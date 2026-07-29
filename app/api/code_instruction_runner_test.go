package api

import "testing"

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
