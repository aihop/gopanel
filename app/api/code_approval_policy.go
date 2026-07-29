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
