package main

import (
	"os"
	"strings"
	"testing"
)

func validContract() taskContract {
	approvalRequired := false
	required := true
	return taskContract{
		SchemaVersion:   schemaVersion,
		ID:              "contract-test",
		Title:           "测试契约",
		Problem:         "任务缺少明确契约",
		ExpectedOutcome: "契约可以被校验",
		Constraints:     []string{"不执行声明的命令"},
		Repositories: []repositoryClaim{{
			ID: "gopanel", Path: ".", Role: "primary",
			DependsOn: []string{}, ReadScope: []string{"app/**"}, WriteScope: []string{".ai/**"},
		}},
		Risks: []risk{{
			ID: "write", Level: "medium", Description: "写入契约",
			Actions: []string{"filesystem_write"}, ApprovalRequired: &approvalRequired,
		}},
		AcceptanceCriteria: []acceptanceCriterion{{
			ID: "validated", Description: "契约通过校验",
			Evidence: []evidence{{Type: "test", Reference: "go test ./tools/contractcheck"}},
		}},
		Verification: []verification{{
			ID: "tests", Repository: "gopanel", Kind: "command",
			Command: "go test ./tools/contractcheck", WorkDir: ".", Required: &required,
		}},
		MobileSummary: mobileSummary{Goal: "建立契约", Success: "校验通过", Attention: "尚未接入数据库"},
	}
}

func TestValidateContractAcceptsValidContract(t *testing.T) {
	if err := validateContract(validContract()); err != nil {
		t.Fatalf("valid contract rejected: %v", err)
	}
}

func TestValidateContractAcceptsReadOnlyRepository(t *testing.T) {
	contract := validContract()
	contract.Repositories[0].WriteScope = []string{}
	if err := validateContract(contract); err != nil {
		t.Fatalf("read-only repository rejected: %v", err)
	}
}

func TestValidateContractRequiresExplicitWriteScope(t *testing.T) {
	contract := validContract()
	contract.Repositories[0].WriteScope = nil
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "writeScope 必须显式声明") {
		t.Fatalf("expected explicit write scope error, got %v", err)
	}
}

func TestValidateContractRequiresAcceptanceEvidence(t *testing.T) {
	contract := validContract()
	contract.AcceptanceCriteria[0].Evidence = nil
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "evidence") {
		t.Fatalf("expected evidence error, got %v", err)
	}
}

func TestValidateContractRequiresApprovalForHighRisk(t *testing.T) {
	contract := validContract()
	contract.Risks[0].Level = "high"
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "必须要求审批") {
		t.Fatalf("expected approval error, got %v", err)
	}
}

func TestValidateContractRequiresApprovalForRiskyAction(t *testing.T) {
	contract := validContract()
	contract.Risks[0].Actions = []string{"git_push"}
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "高风险动作") {
		t.Fatalf("expected risky action approval error, got %v", err)
	}
}

func TestValidateContractRequiresExplicitRiskApproval(t *testing.T) {
	contract := validContract()
	contract.Risks[0].ApprovalRequired = nil
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "approvalRequired 必须显式声明") {
		t.Fatalf("expected explicit approval error, got %v", err)
	}
}

func TestValidateContractRequiresExplicitVerificationRequirement(t *testing.T) {
	contract := validContract()
	contract.Verification[0].Required = nil
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "required 必须显式声明") {
		t.Fatalf("expected explicit required error, got %v", err)
	}
}

func TestValidateContractRequiresMandatoryVerification(t *testing.T) {
	contract := validContract()
	optional := false
	contract.Verification[0].Required = &optional
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "required: true") {
		t.Fatalf("expected mandatory verification error, got %v", err)
	}
}

func TestValidateContractRejectsUnknownRepositoryReference(t *testing.T) {
	contract := validContract()
	contract.Verification[0].Repository = "missing"
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "未知仓库") {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestValidateContractRejectsAmbiguousVerification(t *testing.T) {
	contract := validContract()
	contract.Verification[0].Instruction = "also manual"
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "command 必须声明") {
		t.Fatalf("expected command shape error, got %v", err)
	}
}

func TestValidateContractFileRejectsUnknownField(t *testing.T) {
	path := t.TempDir() + "/unknown.json"
	content := `{
		"schemaVersion": 1,
		"id": "unknown-field",
		"unexpected": true
	}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	err := validateContractFile(path)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}
