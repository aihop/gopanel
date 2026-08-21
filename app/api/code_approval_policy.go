package api

import (
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
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
		return "", buserr.New(constant.ErrCodeInvalidApprovalPolicy)
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

func allCodeApprovalPolicies() []string {
	return []string{codeApprovalPolicyManual, codeApprovalPolicySafeAuto, codeApprovalPolicyFullAuto}
}

func validateCodeExecutorApprovalPolicy(executorID, approvalPolicy string) error {
	definition, err := getCodeExecutorDefinition(executorID)
	if err != nil {
		return err
	}
	for _, supportedPolicy := range definition.ApprovalPolicies {
		if approvalPolicy == supportedPolicy {
			return nil
		}
	}
	return buserr.New(constant.ErrCodeApprovalPolicyUnsupported)
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
