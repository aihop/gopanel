package api

import (
	"errors"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

const (
	codeApprovalPolicyManual   = "manual"
	codeApprovalPolicySafeAuto = "safe_auto"
	codeApprovalPolicyFullAuto = "full_auto"
)

func normalizeCodeApprovalPolicy(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", codeApprovalPolicySafeAuto:
		return codeApprovalPolicySafeAuto, nil
	case codeApprovalPolicyManual:
		return codeApprovalPolicyManual, nil
	case codeApprovalPolicyFullAuto:
		return codeApprovalPolicyFullAuto, nil
	default:
		return "", errors.New("无效的会话审批策略")
	}
}

func codexApprovalPolicy(value string) string {
	switch value {
	case codeApprovalPolicyManual:
		return "untrusted"
	case codeApprovalPolicyFullAuto:
		return "never"
	default:
		return "on-request"
	}
}

func codeSessionRequiresRiskApproval(session *model.AIDevSession) bool {
	return session == nil || session.ApprovalPolicy != codeApprovalPolicyFullAuto
}

func validateCodeExecutorApprovalPolicy(executorID, approvalPolicy string) error {
	if approvalPolicy == codeApprovalPolicyFullAuto || executorID == "codex" {
		return nil
	}
	return errors.New("当前执行器无法保证运行时审批，请选择 Codex 或明确启用全自动模式")
}

func updateCodeApprovalPolicy(session *model.AIDevSession, value string) (bool, error) {
	approvalPolicy, err := normalizeCodeApprovalPolicy(value)
	if err != nil {
		return false, err
	}
	if session.ApprovalPolicy == approvalPolicy {
		return false, nil
	}
	session.ApprovalPolicy = approvalPolicy
	return true, nil
}
